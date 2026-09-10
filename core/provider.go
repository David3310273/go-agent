package core

import (
	json "encoding/json"
	"log"
)

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

type Answer interface {
	Serializable
}

// only oriented to llm, should not define client components in the struct, such as tools/skills
type Provider interface {
	// get id of the provider.
	GetID() string
	// get name
	GetName() string
	// auth the model using the given API keys and return the auth result.
	Auth(ModelConfig) *Diagnostic
	// Complete the conversation with messages and tools, return the result.
	Complete(messages []ReActMessage, tools []Tool) (Answer, []Diagnostic)
	// stream version of Complete, returns a channel of partial answers
	CompleteStream(messages []ReActMessage, tools []Tool) (<-chan Answer, []Diagnostic)
	// get static configuration of the provider.
	GetModelConfig() ModelConfig
	// Init the runtime env for provider if needed.
	Init(ModelConfig) *Diagnostic
}

// the reAct loop of AskQuestion
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
		// dynamically load tools based on latest message and original question
		tools := session.SelectTools(question, (*messages)[len(*messages)-1])
		log.Printf("init messages: %v", messages.ToString())
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
							// auto-added: use result string from CallTool, check diagnostic level for error.
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
	//  capture usage even when choices is empty (usage-only final chunk)
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

// stream version of ProcessQuestion, sends partial answers via hintChan
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
						// auto-added: use result string from CallTool, check diagnostic level for error.
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

// stream version of AskQuestion, accumulates chunks internally and returns the accumulator
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

func ValidateProviders(providers []Provider) []Diagnostic {
	diagnostics := []Diagnostic{}
	hasAvailableModels := false

	for _, model := range providers {
		if err := model.Auth(model.GetModelConfig()); err != nil {
			diagnostics = append(diagnostics, *err)
		} else {
			if err := model.Init(model.GetModelConfig()); err != nil {
				diagnostics = append(diagnostics, *err)
			} else {
				hasAvailableModels = true
			}
		}
	}

	if !hasAvailableModels {
		diagnostics = append(diagnostics, Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeNoAvailableProvider,
		})
		return diagnostics
	}

	return diagnostics
}

// Abstract ProviderFactory creates a Provider single instance
// rootPath parameter for resolving provider config file paths
type ProviderFactory func(rootPath string) (Provider, *Diagnostic)

// providerFactories registry for independent provider components
var providerFactories = map[string]ProviderFactory{}

// RegisterProviderFactory registers a provider factory by name
func RegisterProviderFactory(name string, factory ProviderFactory) {
	providerFactories[name] = factory
}

// GetProviderFactory returns a registered provider factory by name
func GetProviderFactory(name string) (ProviderFactory, bool) {
	factory, ok := providerFactories[name]
	return factory, ok
}

// GetProviderFactories returns all registered provider factories
func GetProviderFactories() map[string]ProviderFactory {
	return providerFactories
}
