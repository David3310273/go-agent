package qwen

// QwenResponse represents Qwen API specific response structure
type QwenResponse struct {
	ID      string       `json:"id"`
	Created int          `json:"created"`
	Model   string       `json:"model"`
	Choices []QwenChoice `json:"choices"`
	Usage   QwenUsage    `json:"usage"`
}

// QwenChoice represents a choice in Qwen response
type QwenChoice struct {
	Index        int         `json:"index"`
	FinishReason string      `json:"finish_reason"`
	Message      QwenMessage `json:"message"`
}

// QwenMessage represents message in Qwen response with reasoning_content
type QwenMessage struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content"`
	ToolCalls        []QwenToolCall `json:"tool_calls"`
}

// QwenToolCall represents a tool call in Qwen response
type QwenToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Index    int              `json:"index"`
	Function QwenToolFunction `json:"function"`
}

// QwenToolFunction represents function details in tool call
type QwenToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// QwenUsage represents token usage in Qwen response
type QwenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}
