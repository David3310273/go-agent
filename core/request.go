package core

import json "encoding/json"

// message format send to llm, refer to openai api request format
type ReActMessage struct {
	Role       RoleType    `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  *[]ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"` // for tool response
}

type Conversation []ReActMessage

func (r ReActMessage) ToString() string {
	response, err := json.Marshal(r)
	if err != nil {
		return ""
	}

	return string(response)
}

func (c Conversation) ToString() string {
	response, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	return string(response)
}
