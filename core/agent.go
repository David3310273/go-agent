package core

import (
	json "encoding/json"
	"fmt"
	"slices"

	// the tool calls of one assistant message run concurrently now
	"sync"
)

// QuestionType distinguishes between normal questions and tool confirmation questions
type QuestionType string

const (
	QuestionTypeNormal      = "normal"
	QuestionTypeToolConfirm = "tool_confirm"
)

// question to agent
type Question interface {
	Serializable
	// get ID
	GetID() string
	// rules that what model would be used to answer this question
	GetModelName() string
	// get original question
	GetQuery() string
	// set original question for reAct
	SetQuery(string)
	// get retry query for reAct if llm returns wrong format
	GetRetryQuery() string
	// get session Name
	GetSessionID() string
	// get stream flag
	GetStreaming() bool
	// get enable thinking flag
	GetEnableThinking() bool
	// get response channel for returning the final Answer
	GetResponseChan() chan Answer
	// get hint channel for returning intermediate status (e.g. thinking...)
	GetHintChan() chan Answer
	// get question type (normal or tool confirm)
	GetType() QuestionType
}

// ToolConfirmAnswer is one user answer for one pending destructive tool call.
// replaces the single tool name, server name and answer triple, a batch of destructive
// calls is answered in one request now.
type ToolConfirmAnswer struct {
	// ToolCallID identifies the pending call, it is the key the session stores it under
	ToolCallID string `json:"toolCallID"`
	ServerName string `json:"serverName"`
	ToolName   string `json:"toolName"`
	// Answer is Yes or No
	Answer string `json:"answer"`
}

type ToolConfirmable interface {
	// GetConfirmAnswers returns one answer per pending destructive tool call
	GetConfirmAnswers() []ToolConfirmAnswer
	// ValidateConfirmAnswers reports whether every answer of the batch is usable
	ValidateConfirmAnswers() bool
}

type AnswerType int

const (
	AnswerTypeData AnswerType = iota
	AnswerTypeError
)

// answer from agent
type Answer interface {
	Serializable
}

type LockManager interface {
	Acquire() *Diagnostic
	Release() *Diagnostic
}

type Configurable interface {
	GetConfigPath() string
}

type WorkFlow interface {
	// before start
	BeforeStart(AgentCoreConfig) []Diagnostic
	// start the agent
	Start(AgentCoreConfig) []Diagnostic

	// before stop
	BeforeStop(AgentCoreConfig) []Diagnostic
	// stop the agent
	Stop(AgentCoreConfig) []Diagnostic
}

// CRUD for sessions map of agent
type SessionManager interface {
	// stop a session
	StopSession(string) *Diagnostic
	// get session, if not exist, create a new one with options
	GetSessionOnCreate(id string, forceCreate bool) (Session, *Diagnostic)
	// load conversation from file/db, take care of size
	RecoverConversation(sessionID string) *Conversation
}

type AgentCore interface {
	Configurable
	Context
	WorkFlow
	Observable
	EventManager
	LockManager
	SessionManager
	// get ID
	GetID() string
	// set ID
	SetID() *Diagnostic
	// set language type for answer
	SetLanguage(LanguageType)
	// get session config
	GetSessionConfig() SessionConfig
	// load config
	LoadConfigs() AgentCoreConfig
	// get root path
	GetRootPath() string
	// get question chan
	GetQuestionChan() chan Question
}

type AgentStatus int

const (
	AgentStatusActive AgentStatus = iota
	AgentStatusSuspended
	AgentStatusExpired // plan expired such as doesn't renew
)

// InitContext initializes the agent context with configs
// loads history, skills, prompt, knowledge base, and tools
func InitContext(agent AgentCore, agentConfigs AgentCoreConfig) []Diagnostic {
	diagnostics := []Diagnostic{}

	// load history
	err := agent.SetHistory(agentConfigs.Agent.History)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load skill definitions for dynamic tool loading
	agent.SetSkills(agentConfigs.Agent.Skill)

	// load prompt
	err = agent.SetPrompt(agentConfigs.Agent.Prompt)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load knowledge base
	err = agent.SetKnowledgeBase(agentConfigs.Agent.KnowledgeBase)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load tool config
	err = agent.SetToolsConfig(agentConfigs.Agent.Tool)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	err = agent.SetMCPClient(agentConfigs.Agent.MCPServer)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	return diagnostics
}

func StartAgentCore(agent AgentCore, appConfigs AppConfig) []Diagnostic {
	diagnostics := []Diagnostic{}

	// DON'T modify the init order here.

	// load all configs
	agentConfigs := agent.LoadConfigs()

	// set ID
	err := agent.SetID()
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// initialize context (history, skills, prompt, kb, tools, mcp clients)
	contextDiagnostics := InitContext(agent, agentConfigs)
	if len(contextDiagnostics) > 0 {
		diagnostics = append(diagnostics, contextDiagnostics...)
	}

	beforeStartDiagnostics := agent.BeforeStart(agentConfigs)
	if len(beforeStartDiagnostics) > 0 {
		diagnostics = append(diagnostics, beforeStartDiagnostics...)
		return diagnostics
	}

	startDiagnostics := agent.Start(agentConfigs)
	if len(startDiagnostics) > 0 {
		diagnostics = append(diagnostics, startDiagnostics...)
		return diagnostics
	}

	return diagnostics
}

func StopAgentCore(agent AgentCore) []Diagnostic {
	// load all configs
	diagnostics := []Diagnostic{}
	agentConfigs := agent.LoadConfigs()

	beforeStopDiagnostics := agent.BeforeStop(agentConfigs)
	if len(beforeStopDiagnostics) > 0 {
		diagnostics = append(diagnostics, beforeStopDiagnostics...)
		return diagnostics
	}

	stopDiagnostics := agent.Stop(agentConfigs)
	if len(stopDiagnostics) > 0 {
		diagnostics = append(diagnostics, stopDiagnostics...)
		return diagnostics
	}

	return diagnostics
}

// =============================================================================
// ReAct loop functions
// =============================================================================

// preparedToolCall is one tool call resolved by the serial prepare pass.
// everything that touches shared state (the loaded tool list, the session confirm maps,
// the skill name of the next round) is resolved on the caller goroutine, so the concurrent workers
// only run a tool and write their own result slot.
type preparedToolCall struct {
	toolCall   ToolCall
	targetTool Tool
	args       map[string]any
	// result is the tool response text, pre-filled by the prepare pass when the call cannot run at
	// all: unknown tool, invalid arguments, or waiting for a destructive confirmation
	result string
	// runnable reports whether a worker goroutine has to execute this call
	runnable bool
}

// prepareToolCalls resolves every tool call of one assistant message on the caller goroutine.
// it looks the tool up by name, parses the arguments, and collects the destructive calls that still wait for a user answer. A destructive call
// is never executed here, its tool response says a confirmation is pending and it comes back in
// confirmCalls so the caller can ask for the whole batch at once.
func prepareToolCalls(
	session Session,
	harness Harness,
	toolCalls []ToolCall,
	loadedTools *[]Tool,
) ([]preparedToolCall, []ToolConfirmCall) {
	prepared := make([]preparedToolCall, 0, len(toolCalls))
	confirmCalls := []ToolConfirmCall{}

	for _, toolCall := range toolCalls {
		item := preparedToolCall{toolCall: toolCall}

		// find the tool from loaded tools by name
		for _, tool := range *loadedTools {
			if tool.GetName() == toolCall.Function.Name {
				item.targetTool = tool
				break
			}
		}
		if item.targetTool == nil {
			item.result = "Error: tool not found - " + toolCall.Function.Name
			prepared = append(prepared, item)
			continue
		}

		// parse arguments
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &item.args); err != nil {
			item.result = "Error: invalid arguments format - " + err.Error()
			prepared = append(prepared, item)
			continue
		}

		// check if tool is destructive and needs user confirmation
		isDestructive := item.targetTool.IsDestructive()
		serverName := ""
		mcpToolName := ""
		// for remote mcp call, the isDestructive should be determined by inner tool
		if toolCall.Function.Name == "UseMCPServerTools" {
			innerDestructive, ok := item.args["isDestructive"].(bool)
			isDestructive = ok && innerDestructive
			// checked assertions, a missing key used to panic here and a panic inside a
			// worker goroutine takes the whole process down
			serverName, _ = item.args["serverName"].(string)
			mcpToolName, _ = item.args["toolName"].(string)
		}

		// keyed by the tool call ID, the server and tool name pair is not unique inside
		// one batch of destructive calls
		if isDestructive && !session.IsToolConfirmed(toolCall.ID) {
			item.result = harness.GetConfirmDestructiveToolAnswer(toolCall.Function.Name)
			confirmCalls = append(confirmCalls, ToolConfirmCall{
				ToolCallID: toolCall.ID,
				ServerName: serverName,
				ToolName:   item.targetTool.GetName(),
				MCPTool:    mcpToolName,
				Args:       item.args,
				Tool:       item.targetTool,
			})
			prepared = append(prepared, item)
			continue
		}

		item.runnable = true
		prepared = append(prepared, item)
	}

	return prepared, confirmCalls
}

// runToolCalls executes a prepared batch concurrently and returns one tool response message per
// call, in tool call order.
// every worker writes only its own slot and the messages are built here, so the caller
// appends them to the conversation on a single goroutine, the events stay ordered and each tool
// response stays matched with its tool_call ID.
func runToolCalls(session Session, harness Harness, prepared []preparedToolCall, model string) []ReActMessage {
	toolResultMessages := make([]ReActMessage, len(prepared))
	var wg sync.WaitGroup

	for idx := range prepared {
		toolResultMessages[idx] = ReActMessage{
			Role:       RoleTool,
			Content:    prepared[idx].result,
			ToolCallID: prepared[idx].toolCall.ID,
		}
		if !prepared[idx].runnable {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// a panicking tool must not take the process down, and this slot must still
			// carry a tool response so every tool_call keeps its matching response
			defer func() {
				if recovered := recover(); recovered != nil {
					LogStd(LogLevelWarn, "[session=%s] tool panicked: name=%s, panic=%v", session.GetID(), prepared[idx].toolCall.Function.Name, recovered)
					toolResultMessages[idx].Content = fmt.Sprintf("Error: tool panicked - %v", recovered)
				}
			}()

			item := prepared[idx]
			// use result string from RunToolCall, check diagnostic level for error.
			result, diag := harness.RunToolCall(session, item.toolCall, item.targetTool, item.args, model)
			if diag != nil && diag.Level == SeverityError {
				toolResultMessages[idx].Content = "Error: " + diag.Message
				return
			}
			toolResultMessages[idx].Content = result
		}(idx)
	}

	wg.Wait()
	return toolResultMessages
}

func watchCancel(session Session, toolCalls []ToolCall) bool {
	ctx := session.GetQueryCtx()
	select {
	case <-ctx.Done():
		LogStd(LogLevelInfo, "[session=%s] cancelled by user", session.GetID())
		messages := session.GetConversation()
		if len(toolCalls) == 0 {
			cancelMessage := ReActMessage{
				Role:    RoleTool,
				Content: "Operation has been canceled by user",
			}
			*messages = append(*messages, cancelMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       cancelMessage,
			})
		} else {
			for _, toolCall := range toolCalls {
				cancelMessage := ReActMessage{
					Role:       RoleTool,
					Content:    fmt.Sprintf("Tool call %s cancelled by user", toolCall.Function.Name),
					ToolCallID: toolCall.ID,
				}
				*messages = append(*messages, cancelMessage)
				Emit(session, CommonEvent[ReActMessage]{
					SourceType: SessionHistory,
					Data:       cancelMessage,
				})
			}
		}
		return true
	default:
		return false
	}
}

// ProcessQuestion is the reAct loop of AskQuestion
func ProcessQuestion(session Session, question Question, harness Harness) (Answer, []Diagnostic) {
	config := session.GetConfigs()
	contextMessages := session.GetContext()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	// generate prompt context and build initial messages
	prompt := harness.GenerateFinalPrompt(contextMessages, int(config.PromptFileMaxSize))

	// use pointer to conversation so modifications are reflected in session
	messages := session.GetConversation()
	if len(*messages) == 0 {
		contextMessage := ReActMessage{Role: RoleSystem, Content: prompt}
		*messages = append(*messages, contextMessage)
		Emit(session, CommonEvent[ReActMessage]{
			SourceType: SessionHistory,
			Data:       contextMessage,
		})
	}

	// use harness to get default answer for fallback
	defaultAnswer := harness.GetDefaultAnswer()
	// check if query has been cancelled at the start of each round
	if watchCancel(session, nil) {
		cancelledMessage := harness.GetCancelledAnswer()
		return cancelledMessage, diagnostics
	}

	// harness constructs user message from question (handles both normal and confirm types)
	userMessage, handleDiag := harness.HandleUserQuestion(session, question)
	// check if harness returned a diagnostic indicating special case
	switch handleDiag.Code {
	case MessageCodeInvalidConfirmAnswer:
		return defaultAnswer, diagnostics
	case MessageCodeToolAlreadyConfirmed:
		return AgentResponse{Response: "This tool operation has already been confirmed and executed, do you want to execute it again?"}, nil
	}

	harness.SetCurrRoundMessages(messages, *userMessage, int(config.MemoryWindowSize), 1)

	Emit(session, CommonEvent[ReActMessage]{
		SourceType: SessionHistory,
		Data:       *userMessage,
	})

	// for ToolConfirm questions with UseMCPServerTools, execute the pending tool directly
	// a batch of destructive calls is confirmed in one request now, so the harness
	// returns one tool response message per answered call and they are appended in that order
	if question.GetType() == QuestionTypeToolConfirm {
		toolResultMessages := harness.HandleUserToolConfirm(session, question)
		for _, toolResultMessage := range toolResultMessages {
			*messages = append(*messages, toolResultMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       toolResultMessage,
			})
		}
	}

	loadedTools := session.GetLoadTools()
	if len(*loadedTools) == 0 {
		harness.LoadTools("", session)
	}

	for i := 0; i < maxReActRounds; i++ {
		response, errs := AskQuestion(session, *messages, *loadedTools, question)
		if len(errs) > 0 {
			diagnostics = append(diagnostics, errs...)
			return defaultAnswer, diagnostics
		}

		answer, ok := response.(AgentResponse)

		if !ok || len(answer.Choices) == 0 || answer.Choices[0].Message == nil {
			// retry with retry query when response format is invalid
			question.SetQuery(question.GetRetryQuery())
			retryMessage := ReActMessage{Role: RoleUser, Content: question.GetRetryQuery()}
			*messages = append(*messages, retryMessage)
			// emit each message event before returning
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       retryMessage,
			})
			continue
		} else {
			// send thoughts to hint chan only when thinking is enabled
			if question.GetEnableThinking() {
				if hintChan := question.GetHintChan(); hintChan != nil {
					hintChan <- answer
				}
			}
			// for simplicity, only use the first choice
			// TODO: support multiple choices
			operation := answer.Choices[0]
			if operation.Message.ToolCalls != nil && len(*operation.Message.ToolCalls) > 0 {
				// handle tool calls
				// 1. append assistant message with tool_calls
				toolMessage := ReActMessage{
					Role:      RoleAssistant,
					Content:   operation.Message.Content,
					ToolCalls: operation.Message.ToolCalls,
				}

				*messages = append(*messages, toolMessage)
				Emit(session, CommonEvent[ReActMessage]{
					SourceType: SessionHistory,
					Data:       toolMessage,
				})

				// 2. execute the tool calls of this assistant message and append every response
				// the prompt lets the model return several tool calls that are meant to
				// run concurrently, so the batch is resolved serially, executed in parallel and
				// appended in tool call order, which keeps a single writer for the conversation.
				toolCalls := *operation.Message.ToolCalls

				// check if session has been cancelled, once for the whole batch
				if watchCancel(session, toolCalls) {
					cancelledMessage := harness.GetCancelledAnswer()
					return cancelledMessage, diagnostics
				}

				// resolve the batch on this goroutine, it reads the loaded tools and the session
				// confirm state, and collects the destructive calls that wait for an answer
				prepared, confirmCalls := prepareToolCalls(session, harness, toolCalls, loadedTools)

				// 3. run the batch concurrently, then append the tool responses in tool call order
				toolResultMessages := runToolCalls(session, harness, prepared, question.GetModelName())
				for _, toolResultMessage := range toolResultMessages {
					*messages = append(*messages, toolResultMessage)
					Emit(session, CommonEvent[ReActMessage]{
						SourceType: SessionHistory,
						Data:       toolResultMessage,
					})
				}

				// 4. ask the user about the whole destructive batch at once and terminate this
				// round. The harness registers every pending call, keyed by its tool call ID.
				if len(confirmCalls) > 0 {
					// delegate to harness to generate the confirmation response for the batch
					confirmResponse := harness.GenerateToolConfirmResponse(session, confirmCalls, answer.Usage)
					if responseChan := question.GetResponseChan(); responseChan != nil {
						responseChan <- confirmResponse
					}
					// return to terminate reAct loop
					return confirmResponse, diagnostics
				}
			} else if operation.FinishReason == FinishReasonStop {
				// append final assistant message to conversation so next question has full history
				finalMessage := ReActMessage{
					Role:    RoleAssistant,
					Content: operation.Message.Content,
				}

				*messages = append(*messages, finalMessage)
				Emit(session, CommonEvent[ReActMessage]{
					SourceType: SessionHistory,
					Data:       finalMessage,
				})

				return answer, nil
			}
		}
	}

	return defaultAnswer, diagnostics
}

// AskQuestion sends messages to the provider and returns response
func AskQuestion(session Session, messages []ReActMessage, tools []Tool, question Question) (Answer, []Diagnostic) {
	diagnostics := []Diagnostic{}

	// pick the model provider
	providers := session.GetContext().GetModelProviders()
	modelName := question.GetModelName()

	// debug log for providers
	LogDebug(LogLevelDebug, "[session=%s] AskQuestion: providers_count=%d, model=%s", session.GetID(), len(providers), modelName)

	var modelProvider Provider
	// use the first provider that has the model from request
	for _, provider := range providers {
		models := provider.GetModelConfig().Models
		if slices.Contains(models, modelName) {
			modelProvider = provider
			break
		}
	}

	if modelProvider == nil {
		diagnostics = append(diagnostics, Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeSystemError,
			Message: fmt.Sprintf("no provider found for model: %s", modelName),
		})
		return nil, diagnostics
	}

	LogStd(LogLevelInfo, "AskQuestion: using provider = %s, model = %s", modelProvider.GetName(), modelName)
	response, errFromLLM := modelProvider.Complete(messages, tools, modelName)
	if len(errFromLLM) > 0 {
		diagnostics = append(diagnostics, errFromLLM...)
	}

	return response, diagnostics
}

// collector for streaming response chunks
type streamAccumulator struct {
	content          string
	toolCalls        []ToolCall
	finishReason     FinishReasonType
	reasoningContent string
	usage            Usage
}

// merge a streaming chunk into the collector
func (acc *streamAccumulator) addChunk(answer Answer) {
	resp, ok := answer.(AgentResponse)
	if !ok {
		return
	}
	// capture usage even when choices is empty (usage-only final chunk)
	if resp.Usage.TotalTokens > 0 {
		acc.usage = resp.Usage
	}
	if len(resp.Choices) == 0 {
		return
	}
	delta := resp.Choices[0].Message
	if delta != nil {
		acc.content += delta.Content
		if delta.ReasoningContent != nil {
			acc.reasoningContent += *delta.ReasoningContent
		}
		if delta.ToolCalls != nil {
			for _, tc := range *delta.ToolCalls {
				if tc.Index < len(acc.toolCalls) {
					acc.toolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
					if tc.Function.Name != "" {
						acc.toolCalls[tc.Index].Function.Name = tc.Function.Name
					}
					if tc.ID != "" {
						acc.toolCalls[tc.Index].ID = tc.ID
					}
				} else {
					acc.toolCalls = append(acc.toolCalls, tc)
				}
			}
		}
	}
	if resp.Choices[0].FinishReason != "" {
		acc.finishReason = resp.Choices[0].FinishReason
	}
}

// ProcessQuestionStream is the stream version of ProcessQuestion, sends partial answers via hintChan
func ProcessQuestionStream(session Session, question Question, harness Harness) (Answer, []Diagnostic) {
	config := session.GetConfigs()
	contextMessages := session.GetContext()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	// generate prompt context and build initial messages
	prompt := harness.GenerateFinalPrompt(contextMessages, int(config.PromptFileMaxSize))

	// use pointer to conversation so modifications are reflected in session
	messages := session.GetConversation()
	if len(*messages) == 0 {
		contextMessage := ReActMessage{Role: RoleSystem, Content: prompt}
		*messages = append(*messages, contextMessage)
		Emit(session, CommonEvent[ReActMessage]{
			SourceType: SessionHistory,
			Data:       contextMessage,
		})
	}

	// use harness to get default answer for fallback
	defaultAnswer := harness.GetDefaultAnswer()
	// check if query has been cancelled at the start of each round
	if watchCancel(session, nil) {
		cancelledMessage := harness.GetCancelledAnswer()
		return cancelledMessage, diagnostics
	}

	// harness constructs user message from question (handles both normal and confirm types)
	userMessage, handleDiag := harness.HandleUserQuestion(session, question)
	// check if harness returned a diagnostic indicating special case
	switch handleDiag.Code {
	case MessageCodeInvalidConfirmAnswer:
		return defaultAnswer, diagnostics
	case MessageCodeToolAlreadyConfirmed:
		return AgentResponse{Response: "This tool operation has already been confirmed and executed, please send the new message if you want to execute it again."}, nil
	}

	LogStd(LogLevelDebug, "[session=%s] current message: role=%s, content_len=%d", session.GetID(), userMessage.Role, len(userMessage.Content))
	harness.SetCurrRoundMessages(messages, *userMessage, int(config.MemoryWindowSize), 1)

	Emit(session, CommonEvent[ReActMessage]{
		SourceType: SessionHistory,
		Data:       *userMessage,
	})

	// for ToolConfirm questions with UseMCPServerTools, execute the pending tool directly
	// a batch of destructive calls is confirmed in one request now and every answer is
	// handled per call, so the Yes/No check moved into the harness. A declined call still gets its
	// tool response, otherwise its tool_call stays unmatched in the conversation.
	if question.GetType() == QuestionTypeToolConfirm {
		toolResultMessages := harness.HandleUserToolConfirm(session, question)
		for _, toolResultMessage := range toolResultMessages {
			*messages = append(*messages, toolResultMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       toolResultMessage,
			})
		}
	}

	loadedTools := session.GetLoadTools()
	if len(*loadedTools) == 0 {
		harness.LoadTools("", session)
	}

	for i := 0; i < maxReActRounds; i++ {
		LogStd(LogLevelDebug, "[session=%s] init messages (stream): count=%d", session.GetID(), len(*messages))

		acc, errs := AskQuestionStream(session, *messages, *loadedTools, question)
		if len(errs) > 0 {
			diagnostics = append(diagnostics, errs...)
			return defaultAnswer, diagnostics
		}

		// build final accumulated response
		var toolCallsPtr *[]ToolCall
		if len(acc.toolCalls) > 0 {
			toolCallsPtr = &acc.toolCalls
		}
		var reasoningPtr *string
		if acc.reasoningContent != "" {
			reasoningPtr = &acc.reasoningContent
		}

		accumulatedResp := AgentResponse{
			Response: acc.content,
			Usage:    acc.usage,
			Choices: []Choices{
				{
					FinishReason: acc.finishReason,
					Message: &ResponseMessage{
						Role:             RoleAssistant,
						Content:          acc.content,
						ToolCalls:        toolCallsPtr,
						ReasoningContent: reasoningPtr,
					},
				},
			},
		}

		if acc.finishReason == FinishReasonFunction && len(acc.toolCalls) > 0 {
			toolMessage := ReActMessage{
				Role:      RoleAssistant,
				Content:   acc.content,
				ToolCalls: &acc.toolCalls,
			}
			*messages = append(*messages, toolMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       toolMessage,
			})

			// same batch handling as ProcessQuestion, the tool calls of one assistant
			// message are resolved serially, executed concurrently and appended in tool call order.
			toolCalls := acc.toolCalls

			// check if session has been cancelled, once for the whole batch
			if watchCancel(session, toolCalls) {
				cancelledMessage := harness.GetCancelledAnswer()
				return cancelledMessage, diagnostics
			}

			// resolve the batch on this goroutine, it reads the loaded tools and the session
			// confirm state, and collects the destructive calls that wait for an answer
			prepared, confirmCalls := prepareToolCalls(session, harness, toolCalls, loadedTools)

			// run the batch concurrently, then append the tool responses in tool call order
			toolResultMessages := runToolCalls(session, harness, prepared, question.GetModelName())
			for _, toolResultMessage := range toolResultMessages {
				*messages = append(*messages, toolResultMessage)
				Emit(session, CommonEvent[ReActMessage]{
					SourceType: SessionHistory,
					Data:       toolResultMessage,
				})
			}

			// ask the user about the whole destructive batch at once and terminate this round.
			// The harness registers every pending call, keyed by its tool call ID.
			if len(confirmCalls) > 0 {
				// delegate to harness to generate the confirmation response for the batch
				confirmResponse := harness.GenerateToolConfirmResponse(session, confirmCalls, acc.usage)
				if responseChan := question.GetResponseChan(); responseChan != nil {
					responseChan <- confirmResponse
				}
				// return to terminate reAct loop
				return confirmResponse, diagnostics
			}
		} else if acc.finishReason == FinishReasonStop {
			finalMessage := ReActMessage{
				Role:    RoleAssistant,
				Content: acc.content,
			}
			*messages = append(*messages, finalMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       finalMessage,
			})
			return accumulatedResp, nil
		}
	}

	return defaultAnswer, diagnostics
}

// AskQuestionStream is the stream version of AskQuestion, accumulates chunks internally and returns the accumulator
func AskQuestionStream(session Session, messages []ReActMessage, tools []Tool, question Question) (*streamAccumulator, []Diagnostic) {
	diagnostics := []Diagnostic{}
	acc := &streamAccumulator{}

	providers := session.GetContext().GetModelProviders()
	modelName := question.GetModelName()

	LogStd(LogLevelDebug, "[session=%s] AskQuestionStream: providers_count=%d, model=%s", session.GetID(), len(providers), modelName)

	var modelProvider Provider
	// use the first provider that has the model from request
	for _, provider := range providers {
		models := provider.GetModelConfig().Models
		if slices.Contains(models, modelName) {
			modelProvider = provider
			break
		}
	}

	if modelProvider == nil {
		return acc, diagnostics
	}

	streamChan, errs := modelProvider.CompleteStream(messages, tools, modelName)
	if len(errs) > 0 {
		diagnostics = append(diagnostics, errs...)
	}

	// collect chunk for reAct process check
	for answer := range streamChan {
		acc.addChunk(answer)
		if resp, ok := answer.(AgentResponse); ok && len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
			// push reasoning/thinking content to hintChan only when thinking is enabled
			if question.GetEnableThinking() && resp.Choices[0].Message.ReasoningContent != nil && *resp.Choices[0].Message.ReasoningContent != "" {
				if hintChan := question.GetHintChan(); hintChan != nil {
					hintChan <- answer
				}
			}
			// push answer content to responseChan for streaming output
			if resp.Choices[0].Message.Content != "" {
				if responseChan := question.GetResponseChan(); responseChan != nil {
					responseChan <- answer
				}
			}
		}
	}

	if acc.content == "" && len(acc.toolCalls) == 0 {
		LogStd(LogLevelWarn, "[session=%s] stream produced no content, returning empty accumulator", session.GetID())
	}

	return acc, diagnostics
}
