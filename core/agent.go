package core

import (
	json "encoding/json"
	"fmt"
	"log"
)

// question to agent
type Question interface {
	Serializable
	// get ID
	GetID() string
	// rules that what model would be used to answer this question
	GetProviderName() string
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
	// get default answer
	GetDefaultAnswer() Answer
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
	GetSessionOnCreate(id string, streaming bool, enableThinking bool, forceCreate bool) (Session, *Diagnostic)
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
}

type AgentStatus int

const (
	AgentStatusActive AgentStatus = iota
	AgentStatusSuspended
	AgentStatusExpired // plan expired such as doesn't renew
)

func StartAgentCore(agent AgentCore, appConfigs AppConfig) []Diagnostic {
	diagnostics := []Diagnostic{}

	// DON'T modify the init order here.

	// load all configs
	agentConfigs := agent.LoadConfigs()

	// inject RootPath from AppConfig to all sub-configs
	agentConfigs.Session.RootPath = appConfigs.RootPath
	for i := range agentConfigs.Tool {
		agentConfigs.Tool[i].RootPath = appConfigs.RootPath
	}

	// set ID
	err := agent.SetID()
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// set logger
	err = agent.SetLogger(agentConfigs.Agent)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load history
	err = agent.SetHistory(agentConfigs.Agent.History)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load skills
	err = agent.SetSkills(agentConfigs.Agent.Skill)
	if err != nil {
		diagnostics = append(diagnostics, *err)
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
	err = agent.SetToolsConfig(agentConfigs.Agent.Tool)
	if err != nil {
		diagnostics = append(diagnostics, *err)
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

// ProcessQuestion is the reAct loop of AskQuestion
func ProcessQuestion(session Session, question Question) (Answer, []Diagnostic) {
	config := session.GetConfigs()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	// generate prompt context and build initial messages
	contextContent := session.GenerateFinalContext(question)
	prompt := contextContent + config.MemoryFileSplitter + "\n"

	// use pointer to conversation so modifications are reflected in session
	messages := session.GetConversation()
	if len(*messages) == 0 {
		contextMessage := ReActMessage{Role: RoleSystem, Content: prompt}
		*messages = append(*messages, contextMessage)
		// emit each message event before returning
		Emit(session, CommonEvent[ReActMessage]{
			SourceType: SessionHistory,
			Data:       contextMessage,
		})
	}

	userMessage := ReActMessage{Role: RoleUser, Content: question.GetQuery()}
	*messages = append(*messages, userMessage)
	// emit each message event before returning
	Emit(session, CommonEvent[ReActMessage]{
		SourceType: SessionHistory,
		Data:       userMessage,
	})

	defaultAnswer := AgentResponse{
		Response: question.GetDefaultAnswer().ToString(),
	}

	// set max reAct rounds
	for i := 0; i < maxReActRounds; i++ {
		// TODO: using harness to uniform the context build
		// dynamically load tools based on latest message and original question
		tools := session.SelectTools(question, (*messages)[len(*messages)-1])
		log.Printf("init messages: %v", messages.ToString())

		// get local knowledge and inject into query
		knowledge := session.SelectLocalKB(question)
		log.Printf("search from local kb: %s", knowledge)

		if len(knowledge) > 0 {
			userQuery := question.GetQuery()
			question.SetQuery(fmt.Sprintf("%s\n%s", userQuery, knowledge))
		}

		response, errs := AskQuestion(session, *messages, tools, question)
		if len(errs) > 0 {
			diagnostics = append(diagnostics, errs...)
			return defaultAnswer, diagnostics
		}

		answer, ok := response.(AgentResponse)

		if !ok || len(answer.Choices) == 0 || answer.Choices[0].Message == nil {
			// retry with retry query when response format is invalid
			question.SetQuery(question.GetRetryQuery())
			retryMessage := ReActMessage{Role: RoleUser, Content: question.GetRetryQuery()}
			(*messages)[len(*messages)-1] = retryMessage
			// emit each message event before returning
			Emit(session, CommonEvent[ReActMessage]{
				SourceType: SessionHistory,
				Data:       retryMessage,
			})
			continue
		} else {
			// for simplicity, only use the first choice
			// TODO: support multiple choices
			operation := answer.Choices[0]

			if operation.Message.ToolCalls != nil && len(*operation.Message.ToolCalls) > 0 {
				// send thoughts to hint chan only when thinking is enabled
				if question.GetEnableThinking() {
					question.GetHintChan() <- answer
				}
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

				// 2. execute each tool and append tool response
				for _, toolCall := range *operation.Message.ToolCalls {
					// find the tool from loaded tools by name
					var targetTool Tool
					for _, tool := range tools {
						if tool.GetName() == toolCall.Function.Name {
							targetTool = tool
							break
						}
					}

					var toolResult string
					if targetTool != nil {
						// parse arguments
						var args map[string]any
						if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
							toolResult = "Error: invalid arguments format - " + err.Error()
						} else {
							// use result string from CallTool, check diagnostic level for error.
							result, diag := CallTool(targetTool, args)
							if diag != nil && diag.Level == SeverityError {
								toolResult = "Error: " + diag.Message
							} else {
								toolResult = result
							}
						}
					} else {
						toolResult = "Error: tool not found - " + toolCall.Function.Name
					}

					// append tool response
					toolResultMessage := ReActMessage{
						Role:       RoleTool,
						Content:    toolResult,
						ToolCallID: toolCall.ID,
					}
					*messages = append(*messages, toolResultMessage)
					Emit(session, CommonEvent[ReActMessage]{
						SourceType: SessionHistory,
						Data:       toolResultMessage,
					})
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
				// send thoughts to hint chan
				if question.GetEnableThinking() {
					question.GetHintChan() <- answer
				}
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
	providers := session.GetModelProviders()
	modelName := question.GetProviderName()

	// debug log for providers
	log.Printf("AskQuestion: providers count = %d, modelName = %s", len(providers), modelName)

	var modelProvider Provider
	if len(providers) > 0 {
		modelProvider = providers[0]
	}

	for _, provider := range providers {
		if modelName != "" && provider.GetName() == modelName {
			modelProvider = provider
			break
		}
	}

	if modelProvider == nil {
		return nil, diagnostics
	}

	response, errFromLLM := modelProvider.Complete(messages, tools)
	if len(errFromLLM) > 0 {
		diagnostics = append(diagnostics, errFromLLM...)
	}

	return response, diagnostics
}

// accumulator for streaming response chunks
type streamAccumulator struct {
	content          string
	toolCalls        []ToolCall
	finishReason     FinishReasonType
	reasoningContent string
	usage            Usage
}

// merge a streaming chunk into the accumulator
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
func ProcessQuestionStream(session Session, question Question) (Answer, []Diagnostic) {
	config := session.GetConfigs()
	maxReActRounds := config.ReActMaxRounds
	diagnostics := []Diagnostic{}

	contextContent := session.GenerateFinalContext(question)
	prompt := contextContent + config.MemoryFileSplitter + "\n"

	messages := session.GetConversation()

	if len(*messages) == 0 {
		contextMessage := ReActMessage{Role: RoleSystem, Content: prompt}
		*messages = append(*messages, contextMessage)
		Emit(session, CommonEvent[ReActMessage]{
			SourceType: SessionHistory,
			Data:       contextMessage,
		})
	}

	userMessage := ReActMessage{Role: RoleUser, Content: question.GetQuery()}
	*messages = append(*messages, userMessage)
	Emit(session, CommonEvent[ReActMessage]{
		SourceType: SessionHistory,
		Data:       userMessage,
	})

	defaultAnswer := AgentResponse{
		Response: question.GetDefaultAnswer().ToString(),
	}

	for i := 0; i < maxReActRounds; i++ {
		tools := session.SelectTools(question, (*messages)[len(*messages)-1])
		log.Printf("init messages (stream): %v", messages.ToString())

		acc, errs := AskQuestionStream(session, *messages, tools, question)
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

			for _, toolCall := range acc.toolCalls {
				var targetTool Tool
				for _, tool := range tools {
					if tool.GetName() == toolCall.Function.Name {
						targetTool = tool
						break
					}
				}

				var toolResult string
				if targetTool != nil {
					var args map[string]any
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						toolResult = "Error: invalid arguments format - " + err.Error()
					} else {
						// use result string from CallTool, check diagnostic level for error.
						result, diag := CallTool(targetTool, args)
						if diag != nil && diag.Level == SeverityError {
							toolResult = "Error: " + diag.Message
						} else {
							toolResult = result
						}
					}
				} else {
					toolResult = "Error: tool not found - " + toolCall.Function.Name
				}

				toolResultMessage := ReActMessage{
					Role:       RoleTool,
					Content:    toolResult,
					ToolCallID: toolCall.ID,
				}
				*messages = append(*messages, toolResultMessage)
				Emit(session, CommonEvent[ReActMessage]{
					SourceType: SessionHistory,
					Data:       toolResultMessage,
				})
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

	providers := session.GetModelProviders()
	modelName := question.GetProviderName()

	log.Printf("AskQuestionStream: providers count = %d, modelName = %s", len(providers), modelName)

	var modelProvider Provider
	if len(providers) > 0 {
		modelProvider = providers[0]
	}
	for _, provider := range providers {
		if modelName != "" && provider.GetName() == modelName {
			modelProvider = provider
			break
		}
	}

	if modelProvider == nil {
		return acc, diagnostics
	}

	streamChan, errs := modelProvider.CompleteStream(messages, tools)
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
		log.Printf("[AskQuestionStream] stream produced no content, returning empty accumulator")
	}

	return acc, diagnostics
}
