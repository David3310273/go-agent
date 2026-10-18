package core

// ToolConfirmCall describes one destructive tool call that waits for a user answer.
// a whole batch of them is confirmed in a single round trip, so the harness receives
// the slice and builds one response carrying every item.
type ConfirmToolCall struct {
	// ToolCallID is the provider tool call ID, it keys the pending call in the session
	ToolCallID string
	// ServerName is the MCP server name, empty for a local tool
	ServerName string
	// ToolName is the outer tool name, e.g. UseMCPServerTools
	ToolName string
	// MCPTool is the inner tool name reported by the MCP server, empty for a local tool
	MCPTool string
	// Args are the outer tool arguments, they are replayed once the user answers Yes
	Args map[string]any
	// Tool is the outer tool instance that runs the call
	Tool Tool
}

// preparedToolCall is one tool call resolved by the serial prepare pass.
// everything that touches shared state (the loaded tool list, the session confirm maps,
// the skill name of the next round) is resolved on the caller goroutine, so the concurrent workers
// only run a tool and write their own result slot.
type PreparedToolCall struct {
	ToolCall   ToolCall
	TargetTool Tool
	Args       map[string]any
	// Result is the tool response text, pre-filled by the prepare pass when the call cannot run at
	// all: unknown tool, invalid arguments, or waiting for a destructive confirmation
	Result string
	// runnable reports whether a worker goroutine has to execute this call
	Runnable bool
}

// PromptBuilder handles system prompt and query preparation.
type PromptBuilder interface {
	AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool
	AddAgentHistory(agentHistory *[]byte, maxSize int)
	// GenerateFinalPrompt simplified to only accept context and maxSize.
	GenerateFinalPrompt(context Context, maxSize int) string
}

type ReActDefaultContentBase interface {
	// GetDefaultAnswer returns the default answer when agent fails to produce a valid response
	GetDefaultAnswer() AgentResponse
	// GetCancelledAnswer returns the answer when question is cancelled by user
	GetCancelledAnswer() AgentResponse
	// get retry question content
	GetRetryQuery() string
	// GetUserToolConfirmMessage returns the confirmation message for user to confirm destructive tool call
	GetUserToolConfirmMessage(toolName string) string
	// Get tool call loop error message
	GetToolCallLoopMessage(toolCall ToolCall) string
	// GetConfirmToolMessage returns the confirmation message for destructive tools
	GetConfirmToolMessage(toolCall ConfirmToolCall) string
	// GenerateToolConfirmResponse generates the confirmation response for a batch of destructive
	// tool calls
	GetAlreadyConfirmedMessage(toolCallID string) string
	GenerateToolConfirmResponse(
		session Session,
		calls []ConfirmToolCall,
		usage Usage,
	) Answer
}

// MessageManager handles conversation window and message rotation.
type ReActMessageManager interface {
	SetCurrRoundMessages(messages *Conversation, message *ReActMessage, windowSize int, skip int)
	SetNextRoundMessages(question *Question, messages *Conversation)
	// HandleUserQuestion handles question types and returns the user message to append.
	// For ToolConfirm questions it also executes pending tool calls and appends results to messages.
	HandleUserQuestion(session Session, question Question, messages *Conversation) *Diagnostic
}

type ReActSupervisor interface {
	// check if there is a tool call loop in the reAct process
	CheckIfHasLoop(question Question, toolCall ToolCall) bool
	WatchReActCancel(session Session, toolCalls []PreparedToolCall) bool
	// handle the case when max ReAct rounds are reached: injects a force message,
	HandleForceTerminate(session Session, messages *Conversation, RuntimeTools *[]Tool, question Question) (AgentResponse, bool)
}

// ToolRunner handles tool loading and execution.
type ToolRunner interface {
	// RunToolCall executes one tool call already resolved by the reAct loop and returns
	RunToolCall(session Session, targetTool Tool, args map[string]any, model string) (string, *Diagnostic)
	// RunToolCalls executes a batch of tool calls already prepared by the reAct loop and returns
	RunToolCalls(session Session, question Question, toolCalls []PreparedToolCall) ([]ReActMessage, *Diagnostic)
	// RunSubSession creates a one-shot sub-session for the CreateSubSession tool
	RunSubSession(session Session, args map[string]any, model string) (string, *Diagnostic)
}

// ConfirmHandler handles user questions, confirmations and tool confirm responses.
type ConfirmHandler interface {
	// HandleUserToolConfirm replays the confirmed batch and returns all messages and tool calls
	HandleUserToolConfirmMessage(session Session, question Question) (ReActMessage, []PreparedToolCall, *Diagnostic)
}

// Harness defines the way of preparing the context during reAct.
type Harness interface {
	PromptBuilder
	ReActMessageManager
	ToolRunner
	ConfirmHandler
	ReActSupervisor
	ReActDefaultContentBase
}
