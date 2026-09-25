package simple

import (
	"fmt"
	"log"
	"strings"

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

func (h SimpleHarness) SetFinalQuery(question *core.Question, knowledge string, splitter string) {
	query := fmt.Sprintf("[Question]\n: %s", (*question).GetQuery())
	kb := fmt.Sprintf("[Knowledge]\n: %s", knowledge)
	finalQuery := fmt.Sprintf("%s%s%s", kb, splitter, query)

	(*question).SetQuery(finalQuery)
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

func (h SimpleHarness) GetConfirmDestructiveToolResult(toolName string) string {
	return fmt.Sprintf("The tool %s is destructive, should make sure if user want to use. Keep running if user responses yes.", toolName)
}

// GenerateToolConfirmResponse generates the confirmation response for destructive tools
// saves pending MCP tool call info to session and returns confirmation response with usage info
func (h SimpleHarness) GenerateToolConfirmResponse(
	session core.Session,
	tool core.Tool,
	args map[string]any,
	usage core.Usage,
) core.Answer {
	// extract serverName and inner tool info from args
	serverName, _ := args["serverName"].(string)
	innerToolName, _ := args["toolName"].(string)
	innerArgs, _ := args["arguments"].(map[string]any)

	// save pending MCP tool call info to session for later reference
	session.SetPendingMCPToolCall(serverName, innerToolName, &core.PendingMCPToolCall{
		Args: innerArgs,
		Tool: tool,
	})
	// return confirmation response
	return SimpleToolConfirmResponse{
		Response: core.AgentResponse{
			Response: h.GetUserToolConfirmMessage(fmt.Sprintf("%s:%s", serverName, innerToolName)),
			Usage:    usage,
		},
		ToolName:   tool.GetName(),
		ServerName: serverName,
		MCPTool:    innerToolName,
		SessionID:  session.GetID(),
	}
}

// HandleUserQuestion handles question types and returns the user message to append
// for normal questions: constructs message from query
// for confirm questions: records answer and constructs simple confirmation message
func (h SimpleHarness) HandleUserQuestion(session core.Session, question core.Question) (*core.ReActMessage, core.Diagnostic) {
	if question.GetType() == core.QuestionTypeToolConfirm {
		confirmQuestion, ok := question.(core.ToolConfirmable)
		// not a tool confirm question, treat it as normal question
		if !ok || !confirmQuestion.ValiateConfirmAnswer() {
			return nil, core.Diagnostic{
				Code:    core.MessageCodeInvalidConfirmAnswer,
				Level:   core.SeverityError,
				Message: "Invalid confirm answer, please use Yes or No.",
			}
		}

		toolName := confirmQuestion.GetConfirmToolName()
		serverName := confirmQuestion.GetConfirmMCPServerName()
		confirmAnswer := confirmQuestion.GetConfirmAnswer()

		if toolCall := session.GetPendingMCPToolCall(serverName, toolName); toolCall == nil {
			// auto-add: tool already confirmed, return diagnostic to skip LLM
			diag := core.Diagnostic{
				Code:    core.MessageCodeToolAlreadyConfirmed,
				Level:   core.SeverityInfo,
				Message: fmt.Sprintf("The tool %s from mcp server %s has already been confirmed and executed.", toolName, serverName),
			}
			return nil, diag
		}

		// record user's answer (Yes or No) - either way counts as confirmed
		session.SetToolConfirmed(serverName, toolName, confirmAnswer)

		// simple confirmation message - tool will be executed directly by ProcessQuestion
		confirmMessage := fmt.Sprintf("The user's answer about using tool %s from mcp server %s is: %s", toolName, serverName, confirmAnswer)
		return &core.ReActMessage{Role: core.RoleUser, Content: confirmMessage}, core.Diagnostic{}
	}

	// normal question: construct message from query
	return &core.ReActMessage{Role: core.RoleUser, Content: question.GetQuery()}, core.Diagnostic{}
}

func (h SimpleHarness) HandleUserToolConfirm(session core.Session, question core.Question) *core.ReActMessage {
	if confirmQuestion, ok := question.(core.ToolConfirmable); ok {
		serverName := confirmQuestion.GetConfirmMCPServerName()
		toolName := confirmQuestion.GetConfirmToolName()

		toolCall := session.GetPendingMCPToolCall(serverName, toolName)
		if toolCall == nil {
			log.Printf("HandleUserToolConfirm: no pending tool call found for %s:%s", serverName, toolName)
			return nil
		}

		if question.GetQuery() == "No" {
			// user declined - return cancellation message
			session.DeletePendingMCPToolCall(serverName, toolName)
			session.ClearToolConfirmed(serverName, toolName)
			return &core.ReActMessage{
				Role:       core.RoleTool,
				Content:    "Tool call has been declined by user",
				ToolCallID: toolCall.ToolCallID,
			}
		}

		// user confirmed - execute the tool
		tool := toolCall.Tool
		args := toolCall.Args
		result, diag := core.CallTool(tool, args)

		// clear pending info in session
		session.DeletePendingMCPToolCall(serverName, toolName)
		session.ClearToolConfirmed(serverName, toolName)

		if diag != nil && diag.Level == core.SeverityError {
			return &core.ReActMessage{
				Role:       core.RoleTool,
				Content:    diag.Message,
				ToolCallID: toolCall.ToolCallID,
			}
		} else {
			return &core.ReActMessage{
				Role:       core.RoleTool,
				Content:    result,
				ToolCallID: toolCall.ToolCallID,
			}
		}
	}

	return nil
}

// RunSubSession creates a one-shot sub-session and drives it through the sub-session's own
// RunQuery, then returns the text that becomes the tool result in the parent conversation.
// auto-add: implements core.Harness. The CreateSubSession tool only declares the intent, the
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

	// auto-add: the schema declares tools as a JSON array, so json.Unmarshal gives []any.
	// The previous map[string]any assertion never matched, and the sub-session was created
	// without any tool, which made the model write tool calls as plain text.
	tools := []core.Tool{}
	if toolList, ok := args["tools"].([]any); ok {
		for _, toolItem := range toolList {
			toolObject, ok := toolItem.(map[string]any)
			if !ok {
				continue
			}
			// auto-add: checked assertion, a missing or non-string name used to panic here
			toolName, ok := toolObject["name"].(string)
			if !ok || toolName == "" {
				continue
			}
			tool := core.CreateTool(toolName, session.GetConfigs().RootPath, session)
			// auto-add: CreateTool returns nil for an unregistered name, a nil tool in the
			// list would break the reAct loop when it looks the tool up by name
			if tool == nil {
				log.Printf("RunSubSession: tool %s is not registered, skipped", toolName)
				continue
			}
			tools = append(tools, tool)
		}
	}

	// auto-add: a sub-session runs only with the tools the caller specified, it never inherits
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

	// auto-add: the sub-session thinking goes to the standard output. core.ProcessQuestion pushes
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

	// auto-add: RunQuery has returned, so nothing sends to hintChan anymore. Close it to let the
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
