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
func (h SimpleHarness) LoadTools(skillName string, session core.Session, rootPath string) {
	context := session.GetContext()
	// if skillName is empty, load default tools directly
	if skillName == "" {
		for _, cfg := range context.GetToolsConfig() {
			if tool := core.CreateTool(cfg.Name, rootPath, context); tool != nil {
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
		if tool := core.CreateTool(toolName, rootPath, context); tool != nil {
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
	toolName string,
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
		ToolName:   toolName,
		ServerName: serverName,
		MCPTool:    innerToolName,
		SessionID:  session.GetID(),
	}
}

// HandleUserQuestion handles question types and returns the user message to append
// for normal questions: constructs message from query
// for confirm questions: records answer and constructs simple confirmation message
func (h SimpleHarness) HandleUserQuestion(session core.Session, question core.Question) *core.ReActMessage {
	if question.GetType() == core.QuestionTypeToolConfirm {
		confirmQuestion, ok := question.(core.ToolConfirmable)
		// not a tool confirm question, treat it as normal question
		if !ok || !confirmQuestion.ValiateConfirmAnswer() {
			return nil
		}

		toolName := confirmQuestion.GetConfirmToolName()
		serverName := confirmQuestion.GetConfirmMCPServerName()
		confirmAnswer := confirmQuestion.GetConfirmAnswer()

		if toolCall := session.GetPendingMCPToolCall(serverName, toolName); toolCall == nil {
			alreadyConfirmedMessage := fmt.Sprintf("The tool %s from mcp server %s has been confirmed before, ignore this tool confirm operation and do nothing.", toolName, serverName)
			return &core.ReActMessage{Role: core.RoleUser, Content: alreadyConfirmedMessage}
		}

		// record user's answer (Yes or No) - either way counts as confirmed
		session.SetToolConfirmed(serverName, toolName, confirmAnswer)

		// simple confirmation message - tool will be executed directly by ProcessQuestion
		confirmMessage := fmt.Sprintf("The user's answer about using tool %s from mcp server %s is: %s", toolName, serverName, confirmAnswer)
		return &core.ReActMessage{Role: core.RoleUser, Content: confirmMessage}
	}

	// normal question: construct message from query
	return &core.ReActMessage{Role: core.RoleUser, Content: question.GetQuery()}
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
