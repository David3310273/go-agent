package core

type SessionStatus int

const (
	SessionStatusRunning SessionStatus = iota
	SessionStatusIdle
	SessionStatusSuspended
	SessionStatusKilled
)

type Session interface {
	WorkFlow
	Observable
	ToolConfirmManager
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
	// get session context
	GetContext() Context
	// get available providers
	GetModelProviders() []Provider
	// session support tree structure
	NewSubSession() Session
	// get question chan
	GetQuestionChan() chan Question
	// process query
	ProcessQuery(query Question)
	// dynamically select local kb given question
	SelectLocalKB(Question) string
	// save reAct message to a storage, not harness
	SaveMemory(ReActMessage) *Diagnostic
	// return pointer so ProcessQuestion can modify session conversation in place
	GetConversation() *Conversation
	// TODO: get loaded tools in session
	GetLoadTools() *[]Tool
	// set load tools
	SetLoadTools(tool Tool)
}

type ToolConfirmManager interface {
	// auto-add: check if a destructive tool has been confirmed by user (answered Yes or No)
	IsToolConfirmed(serverName string, toolName string) bool
	// auto-add: record user's answer for a destructive tool (Yes or No)
	SetToolConfirmed(serverName string, toolName string, answer string)
	// clear tool confirmed
	ClearToolConfirmed(serverName string, toolName string)
	// auto-add: get pending MCP tool call info for confirmation flow
	GetPendingMCPToolCall(serverName, toolName string) *PendingMCPToolCall
	// auto-add: save pending MCP tool call info
	SetPendingMCPToolCall(serverName, toolName string, pending *PendingMCPToolCall)
	// auto-add: delete pending MCP tool call info after tool execution
	DeletePendingMCPToolCall(serverName, toolName string)
}

// auto-add: PendingMCPToolCall stores info about a destructive MCP tool waiting for user confirmation
type PendingMCPToolCall struct {
	Args       map[string]any // inner tool args
	Tool       Tool           // inner tool
	ToolCallID string
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
