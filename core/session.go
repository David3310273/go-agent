package core

type SessionStatus int

const (
	SessionStatusRunning SessionStatus = iota
	SessionStatusIdle
	SessionStatusSuspended
	SessionStatusKilled
)

type Session interface {
	SessionContext
	WorkFlow
	Observable
	EventManager

	GetID() string
	// get status of the session
	GetStatus() SessionStatus
	// set status of the session
	SetStatus(status SessionStatus) *Diagnostic
	// set logger
	SetLogger(SessionConfig) *Diagnostic
	// inherit from agent
	GetConfigs() SessionConfig
	// get available providers
	GetModelProviders() []Provider
	// session support tree structure
	NewSubSession() Session
	// get question chan
	GetQuestionChan() chan Question
	// process query
	ProcessQuery(query Question)
	// dynamically select tools given question and latest message response from llm
	SelectTools(Question, ReActMessage) []Tool
	// dynamically select local kb given question
	SelectLocalKB(Question) string
	// save history
	SaveHistory(Conversation) *Diagnostic
	// return pointer so ProcessQuestion can modify session conversation in place
	GetConversation() *Conversation
}

type SessionAnswer interface {
	Answer
	// get session id
	GetSessionID() string
}

// main func for start and stop session
func StartSession(session Session, config AgentCoreConfig) []Diagnostic {
	diagnostics := session.BeforeStart(config)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	diagnostics = session.Start(config)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	return diagnostics
}

func StopSession(session Session, config AgentCoreConfig) []Diagnostic {
	diagnostics := session.BeforeStop(config)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	diagnostics = session.Stop(config)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	return diagnostics
}
