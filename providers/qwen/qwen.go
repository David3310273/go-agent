package qwen

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"path"

	"github.com/David3310273/go-agent/core"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

type QwenProvider struct {
	// config file path
	Configs core.ModelConfig
	// client
	Client *openai.Client
}

const (
	ConfigFileName = "qwen.json"
	ApiKeyEnvName  = "DASHSCOPE_API_KEY"
)

// register QwenProvider factory so core can create singleton
func init() {
	core.RegisterProviderFactory("qwen", func(rootPath string) (core.Provider, *core.Diagnostic) {
		return NewQwenProvider(rootPath)
	})
}

type SupportModelName string

const (
	Qwen38MaxModelName SupportModelName = "qwen3.8-max"
)

func NewQwenProvider(rootPath string) (*QwenProvider, *core.Diagnostic) {
	// use rootPath instead of hardcoded relative path
	filePath := path.Join(rootPath, "providers/qwen", ConfigFileName)
	log.Printf("provider config path: %s", filePath)

	var configs []byte
	var err error
	if configs, err = os.ReadFile(filePath); err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeProviderCreateError,
			Message: err.Error(),
		}
	}

	var qwenConfig core.ModelConfig
	if err := json.Unmarshal(configs, &qwenConfig); err != nil {
		log.Printf("read file error: %v", err)
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeProviderCreateError,
			Message: err.Error(),
		}
	}

	return &QwenProvider{
		Configs: qwenConfig,
	}, nil
}

func (q *QwenProvider) GetID() string {
	// mock ID for simplicity
	return "Qwen"
}

func (q *QwenProvider) GetName() string {
	// permanent name for simplicity
	return string(Qwen38MaxModelName)
}

func (q *QwenProvider) Init(config core.ModelConfig) *core.Diagnostic {
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv(ApiKeyEnvName)),
		option.WithBaseURL(q.Configs.BaseUrl),
	)

	q.Client = &client

	return nil
}

func (q *QwenProvider) Auth(config core.ModelConfig) *core.Diagnostic {
	// set env and check
	apiKey := config.APIKey.Key
	if err := os.Setenv(ApiKeyEnvName, apiKey); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeProviderCreateError,
			Message: "Failed to set env variable: " + err.Error(),
		}
	}

	apiKeyEnv := os.Getenv(ApiKeyEnvName)
	if apiKeyEnv != config.APIKey.Key {
		return &core.Diagnostic{
			Level: core.SeverityWarn,
			Code:  core.MessageCodeProviderAuthError,
		}
	}

	return nil
}

func (q *QwenProvider) GetModelConfig() core.ModelConfig {
	return q.Configs
}

// adapter function between framework tools and specific provider
func (q *QwenProvider) CreateAvailableTools(tools []core.Tool) []openai.ChatCompletionToolParam {
	toolParam := make([]openai.ChatCompletionToolParam, 0, len(tools))

	for _, tool := range tools {
		schema := tool.GetSchema()
		toolParam = append(toolParam, openai.ChatCompletionToolParam{
			Type: constant.Function(openai.AssistantToolChoiceTypeFunction),
			Function: shared.FunctionDefinitionParam{
				Name:        schema.Function.Name,
				Description: openai.String(schema.Function.Description),
				Parameters:  schema.Function.Parameters,
			},
		})
	}

	return toolParam
}

// convertMessages converts core.ReActMessage to openai message format
func (q *QwenProvider) convertRequestMessages(messages []core.ReActMessage) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case core.RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))

		case core.RoleUser:
			result = append(result, openai.UserMessage(msg.Content))

		case core.RoleAssistant:
			// build assistant message with optional tool_calls
			var assistant openai.ChatCompletionAssistantMessageParam
			assistant.Content.OfString = openai.Opt(msg.Content)
			if msg.ToolCalls != nil && len(*msg.ToolCalls) > 0 {
				toolCallParams := make([]openai.ChatCompletionMessageToolCallParam, 0, len(*msg.ToolCalls))
				for _, tc := range *msg.ToolCalls {
					toolCallParams = append(toolCallParams, openai.ChatCompletionMessageToolCallParam{
						ID:   tc.ID,
						Type: constant.Function(openai.AssistantToolChoiceTypeFunction),
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
				assistant.ToolCalls = toolCallParams
			}
			result = append(result, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})

		case core.RoleTool:
			result = append(result, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}

	return result
}

// stream version of Complete, returns a channel of partial answers
func (q *QwenProvider) CompleteStream(messages []core.ReActMessage, tools []core.Tool) (<-chan core.Answer, []core.Diagnostic) {
	qwenTools := q.CreateAvailableTools(tools)
	qwenMessages := q.convertRequestMessages(messages)

	timeout := time.Duration(q.Configs.MaxWaitingTime) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	stream := q.Client.Chat.Completions.NewStreaming(
		ctx, openai.ChatCompletionNewParams{
			Messages: qwenMessages,
			Model:    string(Qwen38MaxModelName),
			Tools:    qwenTools,
			// return usage info
			StreamOptions: openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.Opt(true),
			},
		},
	)

	ch := make(chan core.Answer, 32)

	go func() {
		defer close(ch)
		defer cancel()

		for stream.Next() {
			chunk := stream.Current()

			//  check for usage in the final chunk (choices may be empty)
			if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 || chunk.Usage.TotalTokens > 0 {
				answer := core.AgentResponse{
					Usage: core.Usage{
						PromptTokens:     uint64(chunk.Usage.PromptTokens),
						CompletionTokens: uint64(chunk.Usage.CompletionTokens),
						TotalTokens:      uint64(chunk.Usage.TotalTokens),
					},
				}
				ch <- answer
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			finishReason := core.FinishReasonType(chunk.Choices[0].FinishReason)

			//  extract reasoning_content from raw delta JSON (not in SDK typed fields)
			var reasoningContent string
			var rawDelta map[string]json.RawMessage

			if err := json.Unmarshal([]byte(delta.RawJSON()), &rawDelta); err == nil {
				if raw, ok := rawDelta["reasoning_content"]; ok {
					var s string
					if err := json.Unmarshal(raw, &s); err == nil {
						reasoningContent = s
					}
				}
			}

			//  log raw chunk details

			var toolCallsPtr *[]core.ToolCall
			if len(delta.ToolCalls) > 0 {
				toolCalls := make([]core.ToolCall, 0, len(delta.ToolCalls))
				for _, tc := range delta.ToolCalls {
					toolCalls = append(toolCalls, core.ToolCall{
						ID:    tc.ID,
						Type:  tc.Type,
						Index: int(tc.Index),
						Function: core.Function{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					})
				}
				toolCallsPtr = &toolCalls
			}

			answer := core.AgentResponse{
				Choices: []core.Choices{
					{
						FinishReason: finishReason,
						Message: &core.ResponseMessage{
							Role:      core.RoleType(delta.Role),
							Content:   delta.Content,
							ToolCalls: toolCallsPtr,
						},
					},
				},
			}
			//  set reasoning content if present
			if reasoningContent != "" {
				answer.Choices[0].Message.ReasoningContent = &reasoningContent
			}

			ch <- answer
		}

		if err := stream.Err(); err != nil {
			log.Printf("[CompleteStream] stream error: %s", err.Error())
		}
	}()

	return ch, nil
}

func (q *QwenProvider) Complete(messages []core.ReActMessage, tools []core.Tool) (core.Answer, []core.Diagnostic) {
	qwenTools := q.CreateAvailableTools(tools)
	qwenMessages := q.convertRequestMessages(messages)

	// use context with timeout from config
	timeout := time.Duration(q.Configs.MaxWaitingTime) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // default 60s if not configured
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	chatCompletion, err := q.Client.Chat.Completions.New(
		ctx, openai.ChatCompletionNewParams{
			Messages: qwenMessages,
			Model:    string(Qwen38MaxModelName),
			Tools:    qwenTools,
		},
	)

	if err != nil {
		log.Printf("[Complete]raw response err: %s", err.Error())
		return core.AgentResponse{}, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeErrorFromLLM,
				Message: err.Error(),
			},
		}
	}

	log.Printf("raw response: %s", chatCompletion.RawJSON())

	// parse raw JSON to QwenResponse for proper field extraction
	var qwenResp QwenResponse
	if err := json.Unmarshal([]byte(chatCompletion.RawJSON()), &qwenResp); err != nil {
		log.Printf("failed to parse qwen response: %v", err)
	}

	// convert QwenResponse to core.AgentResponse
	response := q.convertResponse(&qwenResp)

	return response, nil
}

// convertResponse converts QwenResponse to core.AgentResponse
func (q *QwenProvider) convertResponse(qwenResp *QwenResponse) core.AgentResponse {
	choices := make([]core.Choices, 0, len(qwenResp.Choices))
	answer := ""
	thought := ""

	for _, choice := range qwenResp.Choices {
		coreChoice := core.Choices{
			FinishReason: core.FinishReasonType(choice.FinishReason),
			Index:        choice.Index,
			Message: &core.ResponseMessage{
				Role:    core.RoleType(choice.Message.Role),
				Content: choice.Message.Content,
			},
		}

		// handle reasoning_content (thought) for thinking models
		if choice.Message.ReasoningContent != "" {
			coreChoice.Message.ReasoningContent = &choice.Message.ReasoningContent
			if thought == "" {
				thought = choice.Message.ReasoningContent
			}
		}

		if core.RoleType(choice.Message.Role) == core.RoleAssistant && core.FinishReasonType(choice.FinishReason) == core.FinishReasonStop {
			answer = choice.Message.Content
		}

		// convert tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			toolCalls := make([]core.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				toolCalls = append(toolCalls, core.ToolCall{
					ID:    tc.ID,
					Type:  tc.Type,
					Index: tc.Index,
					Function: core.Function{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			coreChoice.Message.ToolCalls = &toolCalls
		}

		choices = append(choices, coreChoice)
	}

	return core.AgentResponse{
		Thought: thought,
		Usage: core.Usage{
			PromptTokens:     uint64(qwenResp.Usage.PromptTokens),
			CompletionTokens: uint64(qwenResp.Usage.CompletionTokens),
			TotalTokens:      uint64(qwenResp.Usage.TotalTokens),
		},
		Choices:  choices,
		Response: answer,
	}
}
