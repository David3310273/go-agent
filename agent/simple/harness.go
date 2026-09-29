package simple

import (
	"fmt"
	"log"
	"strings"

	// the confirmed destructive calls of one batch run concurrently
	"sync"

	"github.com/David3310273/go-agent/core"
)

type SimpleHarness struct {
}

var SimpleHarnessInstance = NewSimpleHarness()

func NewSimpleHarness() *SimpleHarness {
	return &SimpleHarness{}
}

var _ core.Harness = (*SimpleHarness)(nil)

func (h SimpleHarness) AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool {
	if len(document) == 0 || len(*systemPrompt)+len(document) > maxSize {
		return false
	}

	*systemPrompt = append(*systemPrompt, document...)

	return true
}

func (h SimpleHarness) AddAgentHistory(agentHistory *[]byte, maxSize int) {
}

// GenerateFinalPrompt generates the final prompt from context.
// simplified to only accept context and maxSize, extracts all data from context internally.
func (h SimpleHarness) GenerateFinalPrompt(context core.Context, maxSize int) string {
	var prompt strings.Builder

	// get system prompt from context
	prompt.WriteString(string(context.GetPrompt()))

	// append skill definitions to prompt
	if skillDefs := context.GetSkills(); len(skillDefs) > 0 {
		prompt.WriteString("\n\n# Available Skills\n")
		for _, skill := range skillDefs {
			log.Printf("Found skills name: %s, tools: %v", skill.Name, skill.Tools)
			fmt.Fprintf(&prompt, "- name: **%s**\n", skill.Name)
			fmt.Fprintf(&prompt, "- description: %s\n", skill.Description)
		}
	}

	// append MCP server definitions to prompt
	if mcpConfigs := context.GetMCPServerConfigs(); len(mcpConfigs) > 0 {
		prompt.WriteString("\n\n# Available MCP Servers\n")
		for _, config := range mcpConfigs {
			fmt.Fprintf(&prompt, "- serverName: **%s**\n", config.Name)
			fmt.Fprintf(&prompt, "- description: %s\n", config.Description)
		}
	}

	// get agent history from context
	prompt.WriteString("\n")
	prompt.WriteString(string(context.GetHistory()))

	return prompt.String()
}

func (h SimpleHarness) SetCurrRoundMessages(messages *core.Conversation, message core.ReActMessage, windowSize int, skip int) {
	if messages == nil || windowSize < 1 || skip < 0 {
		return
	}

	msgs := *messages
	end := len(msgs)
	start := max(skip, end-windowSize+1)

	// not full, keep appending
	if start <= skip {
		*messages = append(*messages, message)
		return
	}

	// move to next complete user message
	for start < end && msgs[start].Role != core.RoleUser {
		start += 1
	}

	copy(msgs[skip:], msgs[start:end])

	newLen := skip + (end - start) + 1
	msgs[skip+(end-start)] = message

	*messages = msgs[:newLen]
}

// LoadTools loads tools based on skill name and registers them into session's loaded tools.
// If skillName is empty, loads default tools.
// Uses tool registry to create tools dynamically, no switch needed.
// Deduplication is handled by session.SetLoadTools internally.
func (h SimpleHarness) LoadTools(skillName string, session core.Session) {
	context := session.GetContext()
	rootPath := session.GetConfigs().RootPath
	// if skillName is empty, load default tools directly
	if skillName == "" {
		for _, cfg := range context.GetToolsConfig() {
			if tool := core.CreateTool(cfg.Name, rootPath, session); tool != nil {
				session.SetLoadTools(tool)
			}
		}
		return
	}

	// load tools from skill definition
	skillDef := context.GetSkill(skillName)
	if skillDef == nil {
		return
	}

	for _, toolName := range skillDef.Tools {
		if tool := core.CreateTool(toolName, rootPath, session); tool != nil {
			session.SetLoadTools(tool)
		}
	}
}

func (h SimpleHarness) GetCurrRoundKnowledges(question core.Question) string {
	return ""
}

func (h SimpleHarness) SetNextRoundMessages(question *core.Question, messages *core.Conversation) {
}

// special message management
func (h SimpleHarness) GetDefaultAnswer() core.AgentResponse {
	return core.AgentResponse{
		Response: "Sorry I don't understand your question, and I don't know how to do next, please ask me something else.",
	}
}

func (h SimpleHarness) GetUserToolConfirmMessage(toolName string) string {
	return fmt.Sprintf("The tool %s may be destructive, are you sure you want to proceed?", toolName)
}

func (h SimpleHarness) GetConfirmDestructiveToolAnswer(toolName string) string {
	return fmt.Sprintf("The tool %s is destructive, should make sure if user want to use. Keep running if user responses yes.", toolName)
}

// GenerateToolConfirmResponse generates one confirmation response for a batch of destructive tool
// calls, saves every pending call into the session and returns the response with usage info
// takes the whole batch and keys the pending calls by tool call ID, two destructive
// calls of the same inner tool on the same server used to overwrite each other. Registering the
// pending calls moved here, the reAct loop does not write them anymore.
func (h SimpleHarness) GenerateToolConfirmResponse(
	session core.Session,
	calls []core.ToolConfirmCall,
	usage core.Usage,
) core.Answer {
	items := make([]ToolConfirmItem, 0, len(calls))
	messages := make([]string, 0, len(calls))

	for _, call := range calls {
		// save pending MCP tool call info to session for later reference
		session.SetPendingMCPToolCall(call.ToolCallID, &core.PendingMCPToolCall{
			Args:       call.Args,
			Tool:       call.Tool,
			ToolCallID: call.ToolCallID,
		})

		// a local destructive tool has no server and no inner tool name
		toolLabel := call.MCPTool
		if toolLabel == "" {
			toolLabel = call.ToolName
		}
		if call.ServerName != "" {
			toolLabel = fmt.Sprintf("%s:%s", call.ServerName, toolLabel)
		}
		message := h.GetUserToolConfirmMessage(toolLabel)

		messages = append(messages, message)
		items = append(items, ToolConfirmItem{
			ToolCallID: call.ToolCallID,
			ServerName: call.ServerName,
			ToolName:   call.ToolName,
			MCPTool:    call.MCPTool,
			Message:    message,
		})
	}

	// return confirmation response carrying the whole batch
	return SimpleToolConfirmResponse{
		Response: core.AgentResponse{
			Response: strings.Join(messages, "\n"),
			Usage:    usage,
		},
		Confirms:  items,
		SessionID: session.GetID(),
	}
}

// HandleUserQuestion handles question types and returns the user message to append
// for normal questions: constructs message from query
// for confirm questions: records every answer of the batch and constructs the confirmation message
func (h SimpleHarness) HandleUserQuestion(session core.Session, question core.Question) (*core.ReActMessage, core.Diagnostic) {
	if question.GetType() == core.QuestionTypeToolConfirm {
		confirmQuestion, ok := question.(core.ToolConfirmable)
		// not a tool confirm question, treat it as normal question
		if !ok || !confirmQuestion.ValidateConfirmAnswers() {
			return nil, core.Diagnostic{
				Code:    core.MessageCodeInvalidConfirmAnswer,
				Level:   core.SeverityError,
				Message: "Invalid confirm answers, every item needs a toolCallID and Yes or No.",
			}
		}

		// the answers of one batch are recorded together. An answer whose call was
		// already executed is skipped instead of failing the whole batch.
		answers := confirmQuestion.GetConfirmAnswers()
		confirmMessages := make([]string, 0, len(answers))
		for _, answer := range answers {
			if pending := session.GetPendingMCPToolCall(answer.ToolCallID); pending == nil {
				log.Printf("HandleUserQuestion: no pending tool call %s, skipped", answer.ToolCallID)
				continue
			}

			// record user's answer (Yes or No) - either way counts as confirmed
			session.SetToolConfirmed(answer.ToolCallID, answer.Answer)

			// simple confirmation message - the tools are executed directly by HandleUserToolConfirm
			confirmMessages = append(confirmMessages, fmt.Sprintf(
				"The user's answer about the tool call %s (%s from mcp server %s) is: %s",
				answer.ToolCallID, answer.ToolName, answer.ServerName, answer.Answer))
		}

		if len(confirmMessages) == 0 {
			// every call of the batch is gone, return diagnostic to skip LLM
			diag := core.Diagnostic{
				Code:    core.MessageCodeToolAlreadyConfirmed,
				Level:   core.SeverityInfo,
				Message: "Every tool call of this confirmation batch has already been confirmed and executed.",
			}
			return nil, diag
		}

		return &core.ReActMessage{Role: core.RoleUser, Content: strings.Join(confirmMessages, "\n")}, core.Diagnostic{}
	}

	// normal question: construct message from query
	return &core.ReActMessage{Role: core.RoleUser, Content: question.GetQuery()}, core.Diagnostic{}
}

// HandleUserToolConfirm replays a confirmed batch and returns one tool response message per
// answered call, in the order the user answered them.
// the calls answered Yes run concurrently, they are independent tool calls the user just
// approved as one batch. A call answered No only produces a declined tool response, so its
// tool_call stays matched in the conversation.
func (h SimpleHarness) HandleUserToolConfirm(session core.Session, question core.Question) []core.ReActMessage {
	confirmQuestion, ok := question.(core.ToolConfirmable)
	if !ok {
		return nil
	}

	answers := confirmQuestion.GetConfirmAnswers()
	// one slot per answer, filled in answer order so the conversation keeps the batch order
	toolResultMessages := make([]core.ReActMessage, len(answers))
	pendings := make([]*core.PendingMCPToolCall, len(answers))
	var wg sync.WaitGroup

	for idx, answer := range answers {
		pending := session.GetPendingMCPToolCall(answer.ToolCallID)
		if pending == nil {
			log.Printf("HandleUserToolConfirm: no pending tool call found for %s, skipped", answer.ToolCallID)
			continue
		}
		pendings[idx] = pending

		if answer.Answer == "No" {
			// user declined - return cancellation message, nothing runs for this call
			toolResultMessages[idx] = core.ReActMessage{
				Role:       core.RoleTool,
				Content:    fmt.Sprintf("Tool call %s has been declined by user", pending.ToolCallID),
				ToolCallID: pending.ToolCallID,
			}
			continue
		}

		// user confirmed - execute the tool concurrently, each worker writes only its own slot
		wg.Add(1)
		go func(idx int, pending *core.PendingMCPToolCall) {
			defer wg.Done()
			// a panicking tool must not take the process down, and this slot must still
			// carry a tool response so the tool_call stays matched
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Printf("HandleUserToolConfirm: tool %s panicked: %v", pending.Tool.GetName(), recovered)
					toolResultMessages[idx] = core.ReActMessage{
						Role:       core.RoleTool,
						Content:    fmt.Sprintf("Error: tool execution failed for %s with args %v - %v", pending.ToolCallID, pending.Args, recovered),
						ToolCallID: pending.ToolCallID,
					}
				}
			}()

			result, diag := core.CallTool(pending.Tool, pending.Args)
			content := result
			if diag != nil && diag.Level == core.SeverityError {
				content = diag.Message
			}
			toolResultMessages[idx] = core.ReActMessage{
				Role:       core.RoleTool,
				Content:    content,
				ToolCallID: pending.ToolCallID,
			}
		}(idx, pending)
	}

	wg.Wait()

	// the pending state is cleared only once every worker is done, these maps carry no
	// lock. Clearing an already declined call again is a no-op.
	confirmedMessages := make([]core.ReActMessage, 0, len(answers))
	for idx, answer := range answers {
		if pendings[idx] == nil {
			continue
		}
		session.DeletePendingMCPToolCall(answer.ToolCallID)
		session.ClearToolConfirmed(answer.ToolCallID)
		confirmedMessages = append(confirmedMessages, toolResultMessages[idx])
	}

	return confirmedMessages
}

// IsSessionCancelled checks if the session has been cancelled. If cancelled, it builds cancel
// messages and appends them to the conversation, then returns true. Otherwise returns false.
// if toolCalls is empty, builds a single generic cancel message. If toolCalls is
// non-empty, builds one cancel message per tool call.
func (h SimpleHarness) IsSessionCancelled(session core.Session, toolCalls []core.ToolCall) bool {
	ctx := session.GetQueryCtx()
	select {
	case <-ctx.Done():
		log.Printf("IsSessionCancelled: session %s cancelled by user", session.GetID())
		messages := session.GetConversation()
		if len(toolCalls) == 0 {
			cancelMessage := core.ReActMessage{
				Role:    core.RoleTool,
				Content: "Operation has been canceled by user",
			}
			*messages = append(*messages, cancelMessage)
			core.Emit(session, core.CommonEvent[core.ReActMessage]{
				SourceType: core.SessionHistory,
				Data:       cancelMessage,
			})
		} else {
			for _, toolCall := range toolCalls {
				cancelMessage := core.ReActMessage{
					Role:       core.RoleTool,
					Content:    fmt.Sprintf("Tool call %s cancelled by user", toolCall.Function.Name),
					ToolCallID: toolCall.ID,
				}
				*messages = append(*messages, cancelMessage)
				core.Emit(session, core.CommonEvent[core.ReActMessage]{
					SourceType: core.SessionHistory,
					Data:       cancelMessage,
				})
			}
		}
		return true
	default:
		return false
	}
}

// RunToolCall executes one tool call already resolved by the reAct loop and returns the text that
// becomes its tool result.
// implements core.Harness. It touches no shared state and emits no event, because the
// reAct loop runs a whole batch of them concurrently and appends the tool responses itself, in
// tool call order.
func (h SimpleHarness) RunToolCall(
	session core.Session,
	toolCall core.ToolCall,
	targetTool core.Tool,
	args map[string]any,
	model string,
) (string, *core.Diagnostic) {
	// CreateSubSession is intercepted instead of going through CallTool. The tool only
	// declares the intent, the harness creates and drives the sub-session, because the tool package
	// cannot import the agent package that owns the concrete Session and Question types.
	// Only the model name crosses the boundary, never the caller question.
	if toolCall.Function.Name == core.SubSessionToolName {
		return h.RunSubSession(session, args, model)
	}

	return core.CallTool(targetTool, args)
}

// RunSubSession creates a one-shot sub-session and drives it through the sub-session's own
// RunQuery, then returns the text that becomes the tool result in the parent conversation.
// implements core.Harness. The CreateSubSession tool only declares the intent, the
// sub-session is created and driven here because agent/simple/tools cannot import this package.
// model is the provider name the sub-session runs on, empty means the first available provider.
// The caller question is not accepted on purpose: it is the user's own request and it carries
// the user's response and hint channels.
func (h SimpleHarness) RunSubSession(session core.Session, args map[string]any, model string) (string, *core.Diagnostic) {
	runError := func(message string) *core.Diagnostic {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: message,
		}
	}

	// only one level of sub-session is allowed
	if parent, ok := session.(*SimpleAgentSession); ok && parent.ParentSession != nil {
		return "", runError("a sub-session cannot create another sub-session")
	}

	query, _ := args["query"].(string)
	if query == "" {
		return "", runError("query is required to run a sub-session")
	}

	// the schema declares tools as a JSON array, so json.Unmarshal gives []any.
	// The previous map[string]any assertion never matched, and the sub-session was created
	// without any tool, which made the model write tool calls as plain text.
	tools := []core.Tool{}
	if toolList, ok := args["tools"].([]any); ok {
		for _, toolItem := range toolList {
			toolObject, ok := toolItem.(map[string]any)
			if !ok {
				continue
			}
			// checked assertion, a missing or non-string name used to panic here
			toolName, ok := toolObject["name"].(string)
			if !ok || toolName == "" {
				continue
			}
			tool := core.CreateTool(toolName, session.GetConfigs().RootPath, session)
			// CreateTool returns nil for an unregistered name, a nil tool in the
			// list would break the reAct loop when it looks the tool up by name
			if tool == nil {
				log.Printf("RunSubSession: tool %s is not registered, skipped", toolName)
				continue
			}
			tools = append(tools, tool)
		}
	}

	// a sub-session runs only with the tools the caller specified, it never inherits
	// the parent tool config. With no tool at all the reAct loop sends no function schema to the
	// provider and the model degrades to writing its tool calls as plain text, which used to come
	// back to the parent as the tool result. Fail here instead, before creating the sub-session.
	if len(tools) == 0 {
		return "", runError("no usable tool in the tools argument, a sub-session runs only with " +
			"the tools explicitly specified and every name must be a registered tool")
	}

	subSession := session.NewSubSession(tools)
	sub, ok := subSession.(*SimpleAgentSession)
	if !ok {
		return "", runError(fmt.Sprintf("unexpected sub-session type %T", subSession))
	}

	log.Printf("RunSubSession: session %s created sub-session %s", session.GetID(), sub.GetID())

	// the sub-session thinking goes to the standard output. core.ProcessQuestion pushes
	// the reasoning only when the question carries a hint channel, so give it one and drain it
	// here instead of writing it into the sub-session log file.
	hintChan := make(chan core.Answer, 10)
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for hint := range hintChan {
			hintResponse, ok := hint.(core.AgentResponse)
			if !ok || len(hintResponse.Choices) == 0 || hintResponse.Choices[0].Message == nil {
				continue
			}
			reasoning := hintResponse.Choices[0].Message.ReasoningContent
			if reasoning == nil || *reasoning == "" {
				continue
			}
			fmt.Printf("[sub-session %s][thinking] %s\n", sub.GetID(), *reasoning)
		}
	}()

	// build sub-session question, only the response channel stays nil
	subQuestion := NewSimpleQuestion(
		query,
		sub.GetID(),
		model,
		nil,      // response channel: RunQuery returns answer directly
		hintChan, // hint channel: drained above, the thinking is printed to the standard output
		false,    // streaming: the answer becomes a tool result in the parent
		true,     // enableThinking: the reasoning reaches the hint channel
		core.QuestionTypeNormal,
	)

	// run the sub-session synchronously, every hint send happens inside this call
	response, diagnostics := sub.RunQuery(subQuestion)

	// RunQuery has returned, so nothing sends to hintChan anymore. Close it to let the
	// drainer finish and wait for it, otherwise the last thinking lines can be lost.
	close(hintChan)
	<-drainDone

	if len(diagnostics) > 0 {
		return "", runError(fmt.Sprintf("sub-session failed: %v", diagnostics))
	}

	// extract answer content
	if answer, ok := response.(core.AgentResponse); ok {
		// a real reAct answer carries the text in the first choice
		if len(answer.Choices) > 0 && answer.Choices[0].Message != nil && answer.Choices[0].Message.Content != "" {
			return answer.Choices[0].Message.Content, nil
		}
		// fall back to Response field
		if answer.Response != "" {
			return answer.Response, nil
		}
	}

	return "", runError("sub-session returned no answer")
}
