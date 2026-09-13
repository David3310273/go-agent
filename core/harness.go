package core

// define the way of preparing the context during reAct
type Harness interface {
	AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool
	AddAgentHistory(agentHistory *[]byte, maxSize int)
	GenerateFinalPrompt(systemPrompt string, agentHistory string, maxSize int) string
	SetFinalQuery(question *Question, knowledge string, splitter string)
	SetCurrRoundMessages(messages *Conversation, message ReActMessage, windowSize int, skip int)
	GetNextRoundTools(skillName string) []Tool
	GetCurrRoundKnowledges(question Question) string
	SetNextRoundMessages(question Question, messages *Conversation)
}
