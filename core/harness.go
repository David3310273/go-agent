package core

// ToolConfirmCall describes one destructive tool call that waits for a user answer.
// a whole batch of them is confirmed in a single round trip, so the harness receives
// the slice and builds one response carrying every item.
type ToolConfirmCall struct {
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

// PromptBuilder handles system prompt and query preparation.
type PromptBuilder interface {
	AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool
	AddAgentHistory(agentHistory *[]byte, maxSize int)
	// GenerateFinalPrompt simplified to only accept context and maxSize.
	GenerateFinalPrompt(context Context, maxSize int) string
}

type ReActAnswerBase interface {
	// GetDefaultAnswer returns the default answer when agent fails to produce a valid response
	GetDefaultAnswer() AgentResponse
	// GetCancelledAnswer returns the answer when question is cancelled by user
	GetCancelledAnswer() AgentResponse
	// GetUserToolConfirmMessage returns the confirmation message for user to confirm destructive tool call
	GetUserToolConfirmMessage(toolName string) string
	// Get tool call loop error message
	GetToolCallLoopMessage(toolCall ToolCall) string
}

// MessageManager handles conversation window and message rotation.
type MessageManager interface {
	SetCurrRoundMessages(messages *Conversation, message ReActMessage, windowSize int, skip int)
	SetNextRoundMessages(question *Question, messages *Conversation)
	// HandleUserQuestion handles question types and returns the user message to append
	// returns (*ReActMessage, Diagnostic) where Diagnostic indicates special cases like already confirmed
	HandleUserQuestion(session Session, question Question) (*ReActMessage, Diagnostic)
}

type ReActSupervisor interface {
	// check if there is a tool call loop in the reAct process
	CheckIfHasLoop(question Question, toolCall ToolCall) bool
	// handle the case when max ReAct rounds are reached: injects a force message,
	// calls the model one more time for a final answer, and appends it to conversation.
	// returns (answer, handled) — if handled is false, caller should fall back to default.
	HandleForceFinalAnswer(session Session, messages *Conversation, loadedTools *[]Tool, question Question) (AgentResponse, bool)
}

// ToolRunner handles tool loading and execution.
type ToolRunner interface {
	// LoadTools loads tools based on skill name and registers them into session's loaded tools.
	// If skillName is empty, loads default tools. Deduplication is handled by session.SetLoadTools.
	LoadTools(skillName string, session Session)
	// RunToolCall executes one tool call already resolved by the reAct loop and returns
	RunToolCall(session Session, toolCall ToolCall, targetTool Tool, args map[string]any, model string) (string, *Diagnostic)
	// RunSubSession creates a one-shot sub-session for the CreateSubSession tool
	RunSubSession(session Session, args map[string]any, model string) (string, *Diagnostic)
}

// ConfirmHandler handles user questions, confirmations and tool confirm responses.
type ConfirmHandler interface {
	// GetDestructiveConfirmMessage returns the confirmation message for destructive tools
	GetConfirmDestructiveToolAnswer(toolName string) string
	// GenerateToolConfirmResponse generates the confirmation response for a batch of destructive
	// tool calls
	GenerateToolConfirmResponse(
		session Session,
		calls []ToolConfirmCall,
		usage Usage,
	) Answer
	// HandleUserToolConfirm replays the confirmed batch and returns one tool response message per
	// answered call, in the order the user answered them.
	// returns a slice, a batch of destructive calls is confirmed in one request now
	HandleUserToolConfirm(session Session, question Question) []ReActMessage
}

// Harness defines the way of preparing the context during reAct.
type Harness interface {
	PromptBuilder
	MessageManager
	ToolRunner
	ConfirmHandler
	ReActSupervisor
	ReActAnswerBase
}
