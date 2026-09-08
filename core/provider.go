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

	messages := []ReActMessage{
		{Role: RoleSystem, Content: prompt},
		{Role: RoleUser, Content: question.GetQuery()},
	}

	defaultAnswer := AgentResponse{
		Response: question.GetDefaultAnswer().ToString(),
	}

	var tokenUsage uint64 = 0

	// set max reAct rounds
	for i := 0; i < maxReActRounds; i++ {
		// dynamically load tools based on latest message and original question
		tools := session.SelectTools(question, messages[len(messages)-1])
		response, errs := AskQuestion(session, messages, tools, question)
		if len(errs) > 0 {
			diagnostics = append(diagnostics, errs...)
			return defaultAnswer, diagnostics
		}

		answer, ok := response.(AgentResponse)
		tokenUsage += answer.Usage.TotalTokens

		if !ok || len(answer.Choices) == 0 || answer.Choices[0].Message == nil {
			// retry with retry query when response format is invalid
			question.SetQuery(question.GetRetryQuery())
			messages[len(messages)-1] = ReActMessage{Role: RoleUser, Content: question.GetRetryQuery()}
			continue
		} else {
			// for simplicity, only use the first choice
			// TODO: support multiple choices
			operation := answer.Choices[0]

			// TODO: can send thoughts to hint chan here after supporting thinking
			// session.getHintChan() <- AgentResponse{
			// 	Thought: operation.Message.ReasoningContent,
			// }
			if operation.Message.ToolCalls != nil && len(*operation.Message.ToolCalls) > 0 {
				// handle tool calls
				// 1. append assistant message with tool_calls
				messages = append(messages, ReActMessage{
					Role:      RoleAssistant,
					Content:   operation.Message.Content,
					ToolCalls: operation.Message.ToolCalls,
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
							// call tool using CallTool
							if diag := CallTool(targetTool, args); diag != nil {
								toolResult = "Error: " + diag.Message
							} else {
								toolResult = "Success"
							}
						}
					} else {
						toolResult = "Error: tool not found - " + toolCall.Function.Name
					}

					// append tool response
					messages = append(messages, ReActMessage{
						Role:       RoleTool,
						Content:    toolResult,
						ToolCallID: toolCall.ID,
					})
				}
			} else if operation.FinishReason == FinishReasonStop {
				// session reached final answer, implementation layer should handle event emission
				// send stop event
				log.Printf("final answer: %s", answer.Choices[0].Message.Content)
				log.Printf("token used: %d", tokenUsage)
				//  emit final answer event before returning
				Emit(session, CommonEvent[Conversation]{
					SourceType: SessionFinalAnswer,
					Data:       messages,
				})
				return answer, nil
			}
		}
	}

	Emit(session, CommonEvent[Conversation]{
		SourceType: SessionFinalAnswer,
		Data:       messages,
	})

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
