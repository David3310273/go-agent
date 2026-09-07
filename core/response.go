package core

import "encoding/json"

type RoleType string

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Uniformed AgentResponse for all agents in this framework, refer to openai api response format.
// BE CAREFUL if you want to update/delete
type AgentResponse struct {
	Thought  string    `json:"thought"` // thought of agent
	Usage    Usage     `json:"usage"`
	Response string    `json:"response"` // final answer to user
	Choices  []Choices `json:"choices"`
}

func (a AgentResponse) ToString() string {
	response, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(response)
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     uint64 `json:"promptTokens"`
	CompletionTokens uint64 `json:"completionTokens"`
	TotalTokens      uint64 `json:"totalTokens"`
	CachedTokens     uint64 `json:"cachedTokens"`
}

type FinishReasonType string

const (
	FinishReasonStop     FinishReasonType = "stop"
	FinishReasonLength   FinishReasonType = "length"
	FinishReasonFunction FinishReasonType = "tool_calls"
)

// Choices represents a choice in the response
type Choices struct {
	FinishReason FinishReasonType `json:"finishReason"`
	Index        int              `json:"index"`
	Text         string           `json:"text"`
	Message      *ResponseMessage `json:"message"`
}

// Message represents a message in the choice
type ResponseMessage struct {
	Role             RoleType    `json:"role"`
	Content          string      `json:"content"`
	ReasoningContent *string     `json:"reasoningContent,omitempty"`
	ToolCalls        *[]ToolCall `json:"toolCalls,omitempty"`
}

// Function represents a function call
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall represents a tool call in the message
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
	Index    int      `json:"index"`
}
