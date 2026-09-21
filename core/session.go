package core

import "context"

type SessionStatus int

const (
	SessionStatusRunning SessionStatus = iota
	SessionStatusIdle                  // session is idle, waiting for user input
	SessionStatusKilled                // deleted from agent session manager
)

type Session interface {
	WorkFlow
	Observable
	LockManager
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
	// session support tree structure
	NewSubSession() Session
	// runtime query related
	GetQueryCtx() context.Context
	// set query ctx for query cancellation
	SetQueryContext(ctx context.Context, cancel context.CancelFunc) *Diagnostic
	// cancel current query processing
	CancelQuery()
	// process query
	ProcessQuery(query Question)
	// get question chan
	GetQuestionChan() chan Question
	// runtime message related
	// return pointer so ProcessQuestion can modify session conversation in place
	GetConversation() *Conversation
	// save reAct message to a storage, not harness
	SaveMemory(ReActMessage) *Diagnostic
	// remove session history from memory
	DeleteMemory()
	// runtime tools related
	// get loaded tools in session
	GetLoadTools() *[]Tool
	// set load tools
	SetLoadTools(tool Tool)
}

type ToolConfirmManager interface {
	// check if a destructive tool has been confirmed by user (answered Yes or No)
	IsToolConfirmed(serverName string, toolName string) bool
	// record user's answer for a destructive tool (Yes or No)
	SetToolConfirmed(serverName string, toolName string, answer string)
	// clear tool confirmed
	ClearToolConfirmed(serverName string, toolName string)
	// get pending MCP tool call info for confirmation flow
	GetPendingMCPToolCall(serverName, toolName string) *PendingMCPToolCall
	// save pending MCP tool call info
	SetPendingMCPToolCall(serverName, toolName string, pending *PendingMCPToolCall)
	// delete pending MCP tool call info after tool execution
	DeletePendingMCPToolCall(serverName, toolName string)
}

// PendingMCPToolCall stores info about a destructive MCP tool waiting for user confirmation
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
