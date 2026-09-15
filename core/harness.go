package core

// define the way of preparing the context during reAct
type Harness interface {
	AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool
	AddAgentHistory(agentHistory *[]byte, maxSize int)
	// GenerateFinalPrompt simplified to only accept context and maxSize.
	GenerateFinalPrompt(context Context, maxSize int) string
	SetFinalQuery(question *Question, knowledge string, splitter string)
	SetCurrRoundMessages(messages *Conversation, message ReActMessage, windowSize int, skip int)
	// LoadTools loads tools based on skill name. If skillName is empty, loads default tools.
	LoadTools(skillName string, context Context, rootPath string) []Tool
	GetCurrRoundKnowledges(question Question) string
	SetNextRoundMessages(question *Question, messages *Conversation)
}
