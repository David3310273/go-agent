package core

// define the way of preparing the context during reAct
type Harness interface {
	AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool
	AddAgentHistory(agentHistory *[]byte, maxSize int)
	// GenerateFinalPrompt simplified to only accept context and maxSize.
	GenerateFinalPrompt(context Context, maxSize int) string
	SetFinalQuery(question *Question, knowledge string, splitter string)
	SetCurrRoundMessages(messages *Conversation, message ReActMessage, windowSize int, skip int)
	// LoadTools loads tools based on skill name and registers them into session's loaded tools.
	// If skillName is empty, loads default tools. Deduplication is handled by session.SetLoadTools.
	LoadTools(skillName string, session Session, rootPath string)
	GetCurrRoundKnowledges(question Question) string
	SetNextRoundMessages(question *Question, messages *Conversation)
	// auto-add: GetDefaultAnswer returns the default answer when agent fails to produce a valid response
	GetDefaultAnswer() Answer
	GetUserToolConfirmMessage(toolName string) string
	// auto-add: GetDestructiveConfirmMessage returns the confirmation message for destructive tools
	GetConfirmDestructiveToolResult(toolName string) string
	// auto-add: GenerateToolConfirmResponse generates the confirmation response for destructive tools
	// saves pending tool call info and returns confirmation response with usage info
	GenerateToolConfirmResponse(
		session Session,
		toolName string,
		tool Tool,
		args map[string]any,
		usage Usage,
	) Answer
	// auto-add: HandleUserQuestion handles question types and returns the user message to append
	// for normal questions: constructs message from query
	// for confirm questions: records answer and constructs confirmation message
	HandleUserQuestion(session Session, question Question) *ReActMessage

	HandleUserToolConfirm(session Session, question Question) *ReActMessage
}
