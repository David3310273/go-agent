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

	// auto-add: append MCP server definitions to prompt
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

// LoadTools loads tools based on skill name. If skillName is empty, loads default tools.
// unified method for loading initial tools and skill-based tools.
// Uses tool registry to create tools dynamically, no switch needed.
// updated to pass context to CreateTool for accessing skills and knowledge bases.
func (h SimpleHarness) LoadTools(skillName string, context core.Context, rootPath string) []core.Tool {
	// if skillName is empty, load default tools directly
	if skillName == "" {
		var result []core.Tool
		for _, cfg := range context.GetToolsConfig() {
			if tool := core.CreateTool(cfg.Name, rootPath, context); tool != nil {
				result = append(result, tool)
			}
		}
		return result
	}

	// load tools from skill definition
	skillDef := context.GetSkill(skillName)
	if skillDef == nil {
		return nil
	}

	var result []core.Tool
	for _, toolName := range skillDef.Tools {
		if tool := core.CreateTool(toolName, rootPath, context); tool != nil {
			result = append(result, tool)
		}
	}
	return result
}

func (h SimpleHarness) GetCurrRoundKnowledges(question core.Question) string {
	return ""
}

func (h SimpleHarness) SetNextRoundMessages(question *core.Question, messages *core.Conversation) {
}
