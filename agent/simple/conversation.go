package simple

import "github.com/David3310273/go-agent/core"

// SimpleConversation implements core.Conversation interface
type SimpleConversation struct {
	ID string
	// Question
	Question string
	// answer
	Answer string
}

// GetContent returns content of current round including answer and question
func (c *SimpleConversation) GetContent() core.Serializable {
	// TODO: implement get content
	return nil
}

// SaveConversation saves the conversation result
func (c *SimpleConversation) SaveConversation(result core.Serializable) *core.Diagnostic {
	// TODO: implement save conversation
	return nil
}

// GetID returns the conversation id
func (c *SimpleConversation) GetID() string {
	return c.ID
}
