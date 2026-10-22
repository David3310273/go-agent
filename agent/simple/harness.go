package simple

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	if skillDefs := context.GetSkills(); len(*skillDefs) > 0 {
		prompt.WriteString("\n\n# Available Skills\n")
		for _, skill := range *skillDefs {
			fmt.Fprintf(&prompt, "- name: **%s**\n", skill.GetName())
			fmt.Fprintf(&prompt, "- description: %s\n", skill.GetDescription())
		}
	}

	// get agent history from context
	prompt.WriteString("\n")
	prompt.WriteString(string(context.GetHistory()))

	return prompt.String()
}

func (h SimpleHarness) SetCurrRoundMessages(messages *core.Conversation, message *core.ReActMessage, windowSize int, skip int) {
	if windowSize < 1 || skip < 0 {
		return
	}

	msgs := *messages
	end := len(msgs)
	start := max(skip, end-windowSize+1)

	// not full, keep appending
	if start <= skip {
		if message != nil {
			*messages = append(*messages, *message)
		}
		return
	}

	// move to next complete user message
	for start < end && msgs[start].Role != core.RoleUser {
		start += 1
	}

	copy(msgs[skip:], msgs[start:end])

	newLen := skip + (end - start)
	if message != nil {
		newLen = skip + (end - start) + 1
		msgs[skip+(end-start)] = *message
	}

	*messages = msgs[:newLen]
}

func (h SimpleHarness) GetRetryQuery() string {
	return "No actions found, please provide a valid response."
}

func (h SimpleHarness) CheckIfHasLoop(question core.Question, toolCall core.ToolCall) bool {
	toolCallsMap := question.GetToolCalls()
	functionCalls, ok := toolCallsMap[toolCall.Function.Name]
	if !ok {
		return false
	}

	functionCallsItem := functionCalls.(map[string]int)

	return functionCallsItem[toolCall.Function.Arguments] > 3
}

func (h SimpleHarness) GetForceFinalAnswerMessage() string {
	return "You have reached the maximum number of thinking rounds. Please provide your final answer now based on all the information gathered so far."
}

func (h SimpleHarness) HandleForceTerminate(session core.Session, messages *core.Conversation, RuntimeTools *[]core.Tool, question core.Question) (core.AgentResponse, bool) {
	forceMessage := core.ReActMessage{
		Role:    core.RoleUser,
		Content: h.GetForceFinalAnswerMessage(),
	}
	*messages = append(*messages, forceMessage)
	core.Emit(session, core.CommonEvent[core.ReActMessage]{
		SourceType: core.SessionHistory,
		Data:       forceMessage,
	})

	response, errs := core.AskQuestion(session, *messages, *RuntimeTools, question)
	if len(errs) > 0 {
		return core.AgentResponse{}, false
	}
	answer, ok := response.(core.AgentResponse)
	if !ok || len(answer.Choices) == 0 || answer.Choices[0].Message == nil || answer.Choices[0].Message.Content == "" {
		return core.AgentResponse{}, false
	}

	finalMessage := core.ReActMessage{
		Role:    core.RoleAssistant,
		Content: answer.Choices[0].Message.Content,
	}
	*messages = append(*messages, finalMessage)
	core.Emit(session, core.CommonEvent[core.ReActMessage]{
		SourceType: core.SessionHistory,
		Data:       finalMessage,
	})

	return answer, true
}

func (h SimpleHarness) SetNextRoundMessages(question *core.Question, messages *core.Conversation) {
}

func (h SimpleHarness) GetCancelledAnswer() core.AgentResponse {
	return core.AgentResponse{
		Response: "You have cancelled the conversation. Please tell me what you want to do next.",
	}
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

func (h SimpleHarness) GetTokenExceededAnswer() core.AgentResponse {
	return core.AgentResponse{
		Response: "You have exceeded the maximum number of tokens allowed by your plan. Please use the new session.",
	}
}

func (h SimpleHarness) GetConfirmToolMessage(toolCall core.ConfirmToolCall) string {
	return fmt.Sprintf("The tool %s is destructive (tool call id: %s), should make sure if user want to use. Keep running if user responses yes.", toolCall.ToolName, toolCall.ToolCallID)
}

// GenerateToolConfirmResponse generates one confirmation response for a batch of destructive tool
// calls, saves every pending call into the session and returns the response with usage info
// takes the whole batch and keys the pending calls by tool call ID, two destructive
// calls of the same inner tool on the same server used to overwrite each other. Registering the
// pending calls moved here, the reAct loop does not write them anymore.
func (h SimpleHarness) GenerateToolConfirmResponse(
	session core.Session,
	calls []core.ConfirmToolCall,
	usage core.Usage,
) core.Answer {
	items := make([]ToolConfirmItem, 0, len(calls))
	messages := make([]string, 0, len(calls))

	for _, call := range calls {
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
func (h SimpleHarness) HandleUserQuestion(session core.Session, question core.Question, messages *core.Conversation) *core.Diagnostic {
	confirmMsg := core.ReActMessage{Role: core.RoleUser, Content: question.GetQuery()}
	toolCalls := []core.PreparedToolCall{}

	var diag *core.Diagnostic
	if question.GetType() == core.QuestionTypeToolConfirm {
		confirmMsg, toolCalls, diag = h.HandleUserToolConfirmMessage(session, question)
		if diag != nil {
			return diag
		}
	}

	*messages = append(*messages, confirmMsg)
	core.Emit(session, core.CommonEvent[core.ReActMessage]{
		SourceType: core.SessionHistory,
		Data:       confirmMsg,
	})

	toolResultMessages, diag := h.RunToolCalls(session, question, toolCalls)
	if diag != nil {
		return diag
	}

	for _, toolMessage := range toolResultMessages {
		*messages = append(*messages, toolMessage)
		core.Emit(session, core.CommonEvent[core.ReActMessage]{
			SourceType: core.SessionHistory,
			Data:       toolMessage,
		})
	}

	return nil
}

func (h SimpleHarness) GetToolCallRejectedMessage(toolCall core.ConfirmToolCall) string {
	return fmt.Sprintf("the tool call %s was rejected by user, don't need to execute.", toolCall.ToolCallID)
}

func (h SimpleHarness) GetAlreadyConfirmedMessage(toolCallID string) string {
	return fmt.Sprintf("tool call %s was already responded (duplicate request), ignoring.", toolCallID)
}

// HandleUserToolConfirmMessage replays a confirmed batch and returns one tool response message per
// answered call
func (h SimpleHarness) HandleUserToolConfirmMessage(session core.Session, question core.Question) (core.ReActMessage, []core.PreparedToolCall, *core.Diagnostic) {
	confirmQuestion, ok := question.(core.ToolConfirmable)
	if !ok {
		return core.ReActMessage{}, nil, &core.Diagnostic{
			Code:    core.MessageCodeInvalidConfirmAnswer,
			Level:   core.SeverityError,
			Message: "Invalid confirm question, every item needs a toolCallID and Yes or No.",
		}
	}

	answers := confirmQuestion.GetConfirmAnswers()

	toolCalls := []core.PreparedToolCall{}
	userConfirmMessages := []string{}
	hasValidPending := false
	confirmMessages := core.ReActMessage{
		Role: core.RoleUser,
	}

	for _, answer := range answers {
		pending := session.GetPendingToolCall(answer.ToolCallID)
		if pending == nil {
			core.LogStd(core.LogLevelWarn, "[session=%s] HandleUserToolConfirmMessage: no pending tool call found: call=%s", session.GetID(), answer.ToolCallID)
			userConfirmMessages = append(userConfirmMessages, h.GetAlreadyConfirmedMessage(answer.ToolCallID))
			continue
		}

		hasValidPending = true

		confirmMessage := ""
		if answer.ServerName == "" {
			confirmMessage = fmt.Sprintf("The user's answer about the tool call %s (%s) is: %s", answer.ToolCallID, answer.ToolName, answer.Answer)
		} else {
			confirmMessage = fmt.Sprintf("The user's answer about the tool call %s (%s from mcp server %s) is: %s", answer.ToolCallID, answer.ToolName, answer.ServerName, answer.Answer)
		}

		userConfirmMessages = append(userConfirmMessages, confirmMessage)
		arguments, _ := json.Marshal(pending.Args)

		if answer.Answer == "Yes" {
			toolCall := core.PreparedToolCall{
				ToolCall: core.ToolCall{
					ID:   pending.ToolCallID,
					Type: "function",
					Function: core.Function{
						Name:      pending.Tool.GetName(),
						Arguments: string(arguments),
					},
				},
				TargetTool: pending.Tool,
				Args:       pending.Args,
				Runnable:   true,
			}
			toolCalls = append(toolCalls, toolCall)
		} else {
			// when the user says No, the ReAct loop still needs a tool result for this
			// tool_call_id so the LLM knows the call was resolved and does not retry.
			rejectMsg := core.ReActMessage{
				Role:       core.RoleTool,
				Content:    h.GetToolCallRejectedMessage(*pending),
				ToolCallID: pending.ToolCallID,
			}

			messages := session.GetConversation()
			*messages = append(*messages, rejectMsg)
			core.Emit(session, core.CommonEvent[core.ReActMessage]{
				SourceType: core.SessionHistory,
				Data:       rejectMsg,
			})
		}

		session.DeletePendingToolCall(answer.ToolCallID)
		session.ClearToolConfirmed(answer.ToolCallID)
	}

	// when every answer in the batch is a duplicate, return without asking the LLM to
	// prevent it from hallucinating about whether the tool was executed.
	if !hasValidPending {
		return confirmMessages, nil, &core.Diagnostic{
			Code:    core.MessageCodeToolAlreadyConfirmed,
			Level:   core.SeverityInfo,
			Message: strings.Join(userConfirmMessages, "\n"),
		}
	}

	confirmMessages.Content = strings.Join(userConfirmMessages, "\n")
	return confirmMessages, toolCalls, nil
}

func (h SimpleHarness) WatchReActCancel(session core.Session, toolCalls []core.PreparedToolCall) bool {
	ctx := session.GetQueryCtx()
	select {
	case <-ctx.Done():
		core.LogStd(core.LogLevelInfo, "[session=%s] cancelled by user", session.GetID())
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
					Content:    fmt.Sprintf("Tool call id %s cancelled by user", toolCall.ToolCall.ID),
					ToolCallID: toolCall.ToolCall.ID,
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

// given tool calls, run them concurrently
func (h SimpleHarness) RunToolCalls(
	session core.Session,
	question core.Question,
	toolCalls []core.PreparedToolCall,
) ([]core.ReActMessage, *core.Diagnostic) {
	// last round: force model to produce final answer, skip tool execution
	// check if session has been cancelled, once for the whole batch
	if h.WatchReActCancel(session, toolCalls) {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Message: "Operation has been canceled by user",
		}
	}

	toolResultMessages := make([]core.ReActMessage, len(toolCalls))
	succeeded := make([]bool, len(toolCalls))
	var wg sync.WaitGroup

	model := question.GetModelName()

	for idx := range toolCalls {
		toolResultMessages[idx] = core.ReActMessage{
			Role:       core.RoleTool,
			ToolCallID: toolCalls[idx].ToolCall.ID,
		}

		if toolCalls[idx].ToolCall.Function.Name == "" {
			toolResultMessages[idx].Content = "Error: empty tool name"
			continue
		}

		if !toolCalls[idx].Runnable {
			toolResultMessages[idx].Content = toolCalls[idx].Result
			continue
		}

		arguments := make(map[string]any)
		if err := json.Unmarshal([]byte(toolCalls[idx].ToolCall.Function.Arguments), &arguments); err != nil {
			toolResultMessages[idx].Content = "Error: invalid arguments format - " + err.Error()
			continue
		}

		wg.Add(1)
		go func(idx int, arguments map[string]any) {
			defer wg.Done()
			// a panicking tool must not take the process down, and this slot must still
			// carry a tool response so every tool_call keeps its matching response
			defer func() {
				if recovered := recover(); recovered != nil {
					core.LogStd(core.LogLevelWarn, "[session=%s] tool panicked: name=%s, panic=%v", session.GetID(), toolCalls[idx].ToolCall.Function.Name, recovered)
					toolResultMessages[idx].Content = fmt.Sprintf("Error: tool panicked - %v", recovered)
				}
			}()

			item := toolCalls[idx]
			core.LogStd(core.LogLevelDebug, "[session=%s] running tool call currently: name=%s, args=%v", session.GetID(), item.ToolCall.Function.Name, item.ToolCall.Function.Arguments)
			// use result string from RunToolCall, check diagnostic level for error.
			if h.CheckIfHasLoop(question, item.ToolCall) {
				toolResultMessages[idx].Content = h.GetToolCallLoopMessage(item.ToolCall)
				return
			}

			result, diag := h.RunToolCall(session, item.TargetTool, arguments, model)
			if diag != nil && diag.Level == core.SeverityError {
				toolResultMessages[idx].Content = "Error: " + diag.Message
				return
			}

			toolResultMessages[idx].Content = result
			succeeded[idx] = true
		}(idx, arguments)
	}

	wg.Wait()

	for idx := range toolCalls {
		if succeeded[idx] {
			question.SetToolCalls(toolCalls[idx].ToolCall)
		}
	}

	return toolResultMessages, nil
}

func (h SimpleHarness) GetToolCallLoopMessage(toolCall core.ToolCall) string {
	return fmt.Sprintf("Tool call loop detected at func %s with args %s. Please check the tool call history and replan the next step.", toolCall.Function.Name, toolCall.Function.Arguments)
}

// RunToolCall executes one tool call already resolved by the reAct loop and returns the text that
// becomes its tool result.
// implements core.Harness. It touches no shared state and emits no event, because the
// reAct loop runs a whole batch of them concurrently and appends the tool responses itself, in
// tool call order.
func (h SimpleHarness) RunToolCall(
	session core.Session,
	targetTool core.Tool,
	args map[string]any,
	model string,
) (string, *core.Diagnostic) {
	// CreateSubSession is intercepted instead of going through CallTool. The tool only
	// declares the intent, the harness creates and drives the sub-session, because the tool package
	// cannot import the agent package that owns the concrete Session and Question types.
	// Only the model name crosses the boundary, never the caller question.
	if targetTool.GetName() == core.SubSessionToolName {
		return h.RunSubSession(session, args, model)
	}

	// set session on tools that support runtime session injection (e.g., UseSkill needs it for SetLoadTools).
	// tools created during LoadTools are created with agent context but no session.
	if setter, ok := targetTool.(interface{ SetSession(core.Session) }); ok {
		setter.SetSession(session)
	}

	sandbox, diag := core.BuildSandboxForToolCall(session.GetConfigs().RootPath, targetTool, args)
	if diag != nil {
		core.LogStd(core.LogLevelWarn, "[session=%s] RunToolCall: failed to build sandbox, err: %v", session.GetID(), diag.Message)
	}

	return core.CallTool(targetTool, args, sandbox)
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
			tool := core.CreateTool(toolName, session.GetContext())
			// CreateTool returns nil for an unregistered name, a nil tool in the
			// list would break the reAct loop when it looks the tool up by name
			if tool == nil {
				core.LogStd(core.LogLevelWarn, "[session=%s] RunSubSession: tool not registered: name=%s", session.GetID(), toolName)
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

	core.LogStd(core.LogLevelInfo, "[session=%s] RunSubSession: created sub-session=%s", session.GetID(), sub.GetID())

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
		time.Now().Unix(),
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
