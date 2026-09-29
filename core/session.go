package core

import (
	"context"
	"log"
)

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
	GetLogger() *log.Logger
	// inherit from agent
	GetConfigs() SessionConfig
	// get session context
	GetContext() Context
	// session support tree structure
	NewSubSession(tool []Tool) Session
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

// ToolConfirmManager tracks the destructive tool calls waiting for a user answer.
// every method is keyed by the tool call ID instead of the server and tool name pair.
// A batch of destructive calls in one assistant message usually hits the same MCP server and the
// same outer tool name, so that pair was not unique and the second call overwrote the first one.
type ToolConfirmManager interface {
	// check if a destructive tool call has been confirmed by user (answered Yes or No)
	IsToolConfirmed(toolCallID string) bool
	// record user's answer for a destructive tool call (Yes or No)
	SetToolConfirmed(toolCallID string, answer string)
	// clear tool confirmed
	ClearToolConfirmed(toolCallID string)
	// get pending MCP tool call info for confirmation flow
	GetPendingMCPToolCall(toolCallID string) *PendingMCPToolCall
	// save pending MCP tool call info
	SetPendingMCPToolCall(toolCallID string, pending *PendingMCPToolCall)
	// delete pending MCP tool call info after tool execution
	DeletePendingMCPToolCall(toolCallID string)
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
