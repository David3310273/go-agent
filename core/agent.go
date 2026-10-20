package core

import (
	json "encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/David3310273/go-agent/benchmarker/metrics"
	// the tool calls of one assistant message run concurrently now
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
	// get send timestamp
	GetSendTimestamp() int64
	// get called tools
	GetToolCalls() map[string]any
	// set called tools
	SetToolCalls(ToolCall)
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
	// get available tools
	GetToolsConfig() []ToolConfig
	// get mcp config
	GetMCPServerConfigs() []MCPConfig
	SetMCPServerConfigs([]MCPConfig)
	// get skill
	GetSkillConfig(name string) SkillDefinition
	// get all skills config
	GetSkillConfigs() []SkillDefinition
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
	for _, skillDef := range agentConfigs.Agent.Skill {
		err := agent.AddSkill(skillDef)
		if err != nil {
			diagnostics = append(diagnostics, *err)
		}
	}

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
	err = agent.LoadTools(agentConfigs.Agent.Tool)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// set MCP client
	agent.SetMCPClient(agentConfigs.Agent.MCPServer)

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

	workflowDiagnostics := StartWorkFlow(agent)
	if len(workflowDiagnostics) > 0 {
		diagnostics = append(diagnostics, workflowDiagnostics...)
		return diagnostics
	}

	return diagnostics
}

func StopAgentCore(agent AgentCore) []Diagnostic {
	// load all configs
	workflowDiagnostics := StopWorkFlow(agent)

	return workflowDiagnostics
}

// =============================================================================
// ReAct loop functions
// =============================================================================

func processResponse(
	answer AgentResponse,
	session Session,
	question Question,
	harness Harness,
) (Answer, []Diagnostic) {
	diagnostics := []Diagnostic{}
	messages := session.GetConversation()
	defaultAnswer := harness.GetDefaultAnswer()

	// send thoughts to hint chan only when thinking is enabled
	if question.GetEnableThinking() {
		if hintChan := question.GetHintChan(); hintChan != nil {
			hintChan <- answer
		}
	}

	// TODO: support multiple choices
	operation := answer.Choices[0]
	if operation.FinishReason == FinishReasonStop {
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
	} else if operation.Message.ToolCalls != nil && len(*operation.Message.ToolCalls) > 0 {
		// append assistant message with tool_calls
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

		// prepare tool calls is as same size as original tool calls
		preparedCalls, confirmCalls := prepareToolCalls(session, harness, *operation.Message.ToolCalls)
		// so does toolResultMessages
		toolResultMessages, diag := harness.RunToolCalls(session, question, preparedCalls)
		if diag != nil {
			diagnostics = append(diagnostics, *diag)
			return defaultAnswer, diagnostics
		}

		for _, toolMessage := range toolResultMessages {
			*messages = append(*messages, toolMessage)
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       toolMessage,
			})
		}

		if len(confirmCalls) > 0 {
			for _, confirmCall := range confirmCalls {
				session.SetPendingToolCall(confirmCall.ToolCallID, &confirmCall)
			}

			confirmAnswer := harness.GenerateToolConfirmResponse(session, confirmCalls, answer.Usage)
			return confirmAnswer, nil
		}

		return nil, nil
	} else {
		// append answer directly
		assistantMessage := ReActMessage{
			Role:    RoleAssistant,
			Content: operation.Message.Content,
		}
		*messages = append(*messages, assistantMessage)
		Emit(session, CommonEvent[ReActMessage]{
			SourceType: SessionHistory,
			Data:       assistantMessage,
		})
		return nil, nil
	}
}

// prepareToolCalls resolves every tool call of one assistant message on the caller goroutine.
// it looks the tool up by name, parses the arguments, and collects the destructive calls that still wait for a user answer. A destructive call
// is never executed here, its tool response says a confirmation is pending and it comes back in
// confirmCalls so the caller can ask for the whole batch at once.
func prepareToolCalls(
	session Session,
	harness Harness,
	toolCalls []ToolCall,
) ([]PreparedToolCall, []ConfirmToolCall) {
	prepared := make([]PreparedToolCall, 0, len(toolCalls))
	confirmCalls := []ConfirmToolCall{}
	RuntimeTools := session.GetLoadTools()

	for _, toolCall := range toolCalls {
		// find the tool from loaded tools by name
		item := PreparedToolCall{
			ToolCall: toolCall,
		}
		var targetTool Tool
		for _, tool := range *RuntimeTools {
			if tool.GetName() == toolCall.Function.Name {
				targetTool = tool
				break
			}
		}
		if targetTool == nil {
			item.Result = "Error: tool not found - " + toolCall.Function.Name
			prepared = append(prepared, item)
			continue
		}

		// parse arguments
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &item.Args); err != nil {
			item.Result = "Error: invalid arguments format - " + err.Error()
			prepared = append(prepared, item)
			continue
		}

		// check if tool is destructive and needs user confirmation
		item.TargetTool = targetTool
		isDestructive := item.TargetTool.IsDestructive()
		serverName := ""
		mcpToolName := ""
		// for remote mcp call, the isDestructive should be determined by inner tool
		if toolCall.Function.Name == "UseMCPServerTools" {
			innerDestructive, ok := item.Args["isDestructive"].(bool)
			isDestructive = ok && innerDestructive
			// checked assertions, a missing key used to panic here and a panic inside a
			// worker goroutine takes the whole process down
			serverName, _ = item.Args["serverName"].(string)
			mcpToolName, _ = item.Args["toolName"].(string)
		}

		// keyed by the tool call ID, the server and tool name pair is not unique inside
		// one batch of destructive calls
		if isDestructive && !session.IsToolConfirmed(toolCall.ID) {
			confirmedCall := ConfirmToolCall{
				ToolCallID: toolCall.ID,
				ServerName: serverName,
				ToolName:   item.TargetTool.GetName(),
				MCPTool:    mcpToolName,
				Args:       item.Args,
				Tool:       item.TargetTool,
			}
			item.Result = harness.GetConfirmToolMessage(confirmedCall)
			confirmCalls = append(confirmCalls, confirmedCall)
			prepared = append(prepared, item)
			continue
		}

		item.Runnable = true
		prepared = append(prepared, item)
	}

	return prepared, confirmCalls
}

// ProcessQuestion is the reAct loop of AskQuestion
func ProcessQuestion(session Session, question Question, harness Harness) (Answer, []Diagnostic) {
	config := session.GetConfigs()
	reActContext := session.GetContext()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	// generate prompt context and build initial messages
	prompt := harness.GenerateFinalPrompt(reActContext, int(config.PromptFileMaxSize))

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
	if harness.WatchReActCancel(session, nil) {
		cancelledMessage := harness.GetCancelledAnswer()
		return cancelledMessage, diagnostics
	}

	// harness constructs user message from question (handles both normal and confirm types)
	// for ToolConfirm questions it also executes pending tool calls internally
	handleDiag := harness.HandleUserQuestion(session, question, messages)
	// check if harness returned a diagnostic indicating special case
	if handleDiag != nil {
		switch handleDiag.Code {
		case MessageCodeInvalidConfirmAnswer:
			return defaultAnswer, diagnostics
		case MessageCodeToolAlreadyConfirmed:
			return AgentResponse{Response: "This tool operation has already been confirmed, do you want to execute it again?"}, nil
		}
	}

	// apply sliding window
	harness.SetCurrRoundMessages(messages, nil, int(session.GetConfigs().MemoryWindowSize), 1)

	runtimeTools := session.GetLoadTools()
	if len(*runtimeTools) == 0 {
		runtimeTools = reActContext.GetTools()
	}

	var tokenUsage uint64 = 0

	// main loop of reAct
	for round := 0; round < maxReActRounds; round++ {
		response, errs := AskQuestion(session, *messages, *runtimeTools, question)
		if len(errs) > 0 {
			diagnostics = append(diagnostics, errs...)
			return defaultAnswer, diagnostics
		}

		answer, ok := response.(AgentResponse)
		if ok {
			tokenUsage += answer.Usage.TotalTokens
		}
		if !ok || len(answer.Choices) == 0 || answer.Choices[0].Message == nil {
			// force final answer if reached max rounds
			if round == maxReActRounds-1 {
				if answer, handled := harness.HandleForceTerminate(session, messages, runtimeTools, question); handled {
					return answer, diagnostics
				}
				return harness.GetDefaultAnswer(), diagnostics
			}
			// retry with retry query when response format is invalid
			question.SetQuery(harness.GetRetryQuery())
			retryMessage := ReActMessage{Role: RoleUser, Content: harness.GetRetryQuery()}
			*messages = append(*messages, retryMessage)
			// emit each message event before returning
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       retryMessage,
			})
			continue
		} else {
			operation := answer.Choices[0]
			// force final answer if reached max rounds
			if operation.FinishReason != FinishReasonStop && round == maxReActRounds-1 {
				if answer, handled := harness.HandleForceTerminate(session, messages, runtimeTools, question); handled {
					return answer, diagnostics
				}
				return harness.GetDefaultAnswer(), diagnostics
			}

			finalResponse, diag := processResponse(answer, session, question, harness)
			if diag != nil {
				diagnostics = append(diagnostics, diag...)
				return defaultAnswer, diagnostics
			} else if finalResponse != nil {
				// emit final message event
				if session.GetConfigs().EnableBenchmark {
					benchmarkChans := session.GetBenchmarkListeningChannels()
					SendBenchmarkEvent(benchmarkChans[MetricQueryReActRound], BenchmarkEvent[any]{
						TimeStamp: time.Now().UnixMilli(),
						Type:      MetricQueryReActRound,
						Data: metrics.QueryReActRoundMetric{
							SessionID:  session.GetID(),
							QueryID:    question.GetID(),
							ReActRound: round,
							TimeStamp:  time.Now().UnixMilli(),
						},
					})
					SendBenchmarkEvent(benchmarkChans[MetricQueryTokenUsage], BenchmarkEvent[any]{
						TimeStamp: time.Now().UnixMilli(),
						Type:      MetricQueryTokenUsage,
						Data: metrics.QueryTokenUsageMetric{
							SessionID:  session.GetID(),
							QueryID:    question.GetID(),
							TokenUsage: tokenUsage,
							Model:      question.GetModelName(),
							TimeStamp:  time.Now().UnixMilli(),
						},
					})
				}
				return finalResponse, nil
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
		// if modelName is empty, use the first provider
		if modelName == "" || slices.Contains(models, modelName) {
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
	reActContext := session.GetContext()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	// generate prompt context and build initial messages
	prompt := harness.GenerateFinalPrompt(reActContext, int(config.PromptFileMaxSize))
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
	if harness.WatchReActCancel(session, nil) {
		cancelledMessage := harness.GetCancelledAnswer()
		return cancelledMessage, diagnostics
	}

	// harness constructs user message from question (handles both normal and confirm types)
	// for ToolConfirm questions it also executes pending tool calls internally
	handleDiag := harness.HandleUserQuestion(session, question, messages)
	// check if harness returned a diagnostic indicating special case
	if handleDiag != nil {
		switch handleDiag.Code {
		case MessageCodeInvalidConfirmAnswer:
			return defaultAnswer, diagnostics
		case MessageCodeToolAlreadyConfirmed:
			return AgentResponse{Response: "This tool operation has already been confirmed, do you want to execute it again?"}, nil
		}
	}

	// apply sliding window
	harness.SetCurrRoundMessages(messages, nil, int(session.GetConfigs().MemoryWindowSize), 1)

	runtimeTools := session.GetLoadTools()
	if len(*runtimeTools) == 0 {
		runtimeTools = reActContext.GetTools()
	}

	var tokenUsage uint64 = 0

	// main loop of reAct process
	for round := 0; round < maxReActRounds; round++ {
		LogStd(LogLevelDebug, "[session=%s] init messages (stream): count=%d", session.GetID(), len(*messages))
		acc, errs := AskQuestionStream(session, *messages, *runtimeTools, question)
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

		tokenUsage += acc.usage.TotalTokens

		if acc.finishReason != FinishReasonStop && round == maxReActRounds-1 {
			if session.GetConfigs().EnableBenchmark {
				benchmarkChans := session.GetBenchmarkListeningChannels()
				SendBenchmarkEvent(benchmarkChans[MetricQueryReActRound], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryReActRound,
					Data: metrics.QueryReActRoundMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						ReActRound: round,
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQueryTokenUsage], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryTokenUsage,
					Data: metrics.QueryTokenUsageMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						TokenUsage: tokenUsage,
						Model:      question.GetModelName(),
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQueryFailedCount], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryFailedCount,
					Data: metrics.QueryFailedCountMetric{
						SessionID: session.GetID(),
						Count:     1,
						TimeStamp: time.Now().UnixMilli(),
					},
				})
			}
			if answer, handled := harness.HandleForceTerminate(session, messages, runtimeTools, question); handled {
				return answer, diagnostics
			}
			return harness.GetDefaultAnswer(), diagnostics
		}

		finalAnswer, diag := processResponse(accumulatedResp, session, question, harness)
		if diag != nil {
			if session.GetConfigs().EnableBenchmark {
				benchmarkChans := session.GetBenchmarkListeningChannels()
				SendBenchmarkEvent(benchmarkChans[MetricQueryReActRound], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryReActRound,
					Data: metrics.QueryReActRoundMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						ReActRound: round,
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQueryTokenUsage], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryTokenUsage,
					Data: metrics.QueryTokenUsageMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						TokenUsage: tokenUsage,
						Model:      question.GetModelName(),
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQueryFailedCount], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryFailedCount,
					Data: metrics.QueryFailedCountMetric{
						SessionID: session.GetID(),
						Count:     1,
						TimeStamp: time.Now().UnixMilli(),
					},
				})
			}
			diagnostics = append(diagnostics, diag...)
			return defaultAnswer, diagnostics
		} else if finalAnswer != nil {
			if session.GetConfigs().EnableBenchmark {
				benchmarkChans := session.GetBenchmarkListeningChannels()
				SendBenchmarkEvent(benchmarkChans[MetricQueryReActRound], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryReActRound,
					Data: metrics.QueryReActRoundMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						ReActRound: round,
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQueryTokenUsage], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQueryTokenUsage,
					Data: metrics.QueryTokenUsageMetric{
						SessionID:  session.GetID(),
						QueryID:    question.GetID(),
						TokenUsage: tokenUsage,
						Model:      question.GetModelName(),
						TimeStamp:  time.Now().UnixMilli(),
					},
				})
				SendBenchmarkEvent(benchmarkChans[MetricQuerySuccessCount], BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      MetricQuerySuccessCount,
					Data: metrics.QuerySuccessCountMetric{
						SessionID: session.GetID(),
						Count:     1,
						TimeStamp: time.Now().UnixMilli(),
					},
				})
			}
			return finalAnswer, nil
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
		// if modelName is empty, use the first provider
		if modelName == "" || slices.Contains(models, modelName) {
			modelProvider = provider
			break
		}
	}

	if modelProvider == nil {
		LogStd(LogLevelError, "[session=%s] AskQuestionStream: no provider found for model=%s", session.GetID(), modelName)
		return acc, diagnostics
	}

	LogStd(LogLevelDebug, "[session=%s] AskQuestionStream: found provider=%s, calling CompleteStream", session.GetID(), modelProvider.GetName())

	streamChan, errs := modelProvider.CompleteStream(messages, tools, modelName)
	if len(errs) > 0 {
		LogStd(LogLevelError, "[session=%s] AskQuestionStream: errors from CompleteStream: %v", session.GetID(), errs)
		diagnostics = append(diagnostics, errs...)
	}

	LogStd(LogLevelDebug, "[session=%s] AskQuestionStream: streamChan created, waiting for chunks", session.GetID())

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
