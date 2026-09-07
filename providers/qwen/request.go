package qwen

import (
	"github.com/David3310273/go-agent/core"
)

type RoleType string

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Request struct for Qwen API request
type Request struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
}

// RequestMessage represents a message in the request
type RequestMessage struct {
	Role      RoleType         `json:"role"`
	Content   string           `json:"content"`
	ToolCalls *[]core.ToolCall `json:"tool_calls,omitempty"`
}
