package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
)

func init() {
	// register UseSkillCall tool factory
	// updated to accept Session instead of Context.
	core.RegisterTool("UseSkill", func(rootPath string, session core.Session) core.Tool {
		config := LoadToolConfig(rootPath, UseSkillSchemaPath, "useskill.config.json")
		return UseSkillCall{
			Name:     config.Name,
			Schema:   config.Schema,
			RootPath: rootPath,
			Session:  session,
		}
	})
}

// UseSkillCall implements core.Tool interface for loading skills dynamically.
// loads skill by name and returns skill info with tool list.
// changed Context to Session for unified access to session and context.
type UseSkillCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	// session for accessing context at runtime.
	Session core.Session
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f UseSkillCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f UseSkillCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	content, err := os.ReadFile(path.Join(f.RootPath, UseSkillSchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "useskill: failed to read schema: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}

	return schema
}

func (f UseSkillCall) IsDestructive() bool {
	return false
}

func (f UseSkillCall) ToShellScript(args map[string]any) string {
	return ""
}

func (f UseSkillCall) GetAvailableSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
}

// GetDescription returns the tool description from schema
func (f UseSkillCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f UseSkillCall) GetContext() core.Context {
	return f.Session.GetContext()
}

// Validate validates the tool configuration
// updated to get skills from Context.
func (f UseSkillCall) Validate(args map[string]any) *core.Diagnostic {
	name, _ := args["name"].(string)
	if name == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: "skill name is required",
		}
	}

	// check if skill exists
	for _, skill := range f.Session.GetContext().GetSkills() {
		if skill.Name == name {
			return nil
		}
	}

	return &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeToolRunError,
		Message: fmt.Sprintf("skill not found: %s", name),
	}
}

// GetRunner returns the runner function for UseSkillCall
// returns skill info with tool list and descriptions for reAct to load tools dynamically.
// updated to get skills from Context.
func (f UseSkillCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		name, _ := args["name"].(string)

		// find skill definition from context
		var skillDef *core.SkillDefinition
		skills := f.Session.GetContext().GetSkills()
		for i := range skills {
			if skills[i].Name == name {
				skillDef = &skills[i]
				break
			}
		}

		if skillDef == nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("skill not found: %s", name),
			}
		}

		// build response with skill info and tool list with descriptions
		var response strings.Builder
		core.LogStd(core.LogLevelInfo, "Skill: %s\n", skillDef.Name)
		core.LogStd(core.LogLevelInfo, "Description: %s\n", skillDef.Description)
		core.LogStd(core.LogLevelInfo, "\nTools to execute (with descriptions):\n")
		for _, toolName := range skillDef.Tools {
			// use GetTool to get tool instance, then call GetDescription.
			tool := core.GetTool(toolName, f.RootPath, f.Session)
			if tool != nil {
				// register tool to session's loaded tools
				f.Session.SetLoadTools(tool)
				fmt.Fprintf(&response, "- **%s**: %s\n", toolName, tool.GetDescription())
				// add sandbox spec in skill
				supportSandbox := tool.GetAvailableSandboxEnv()
				if supportSandbox.Type != "" {
					fmt.Fprintf(&response, "- - **supported runtime environment sandbox**: type: %s, image name: %s, vm name: %s\n", supportSandbox.Type, supportSandbox.ImageName, supportSandbox.VMName)
				}
			}
		}
		fmt.Fprintf(&response, "\nPlease call these tools with appropriate parameters.")

		core.LogStd(core.LogLevelDebug, "skill loaded: name=%s, desc=%s", skillDef.Name, skillDef.Description)

		return response.String(), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
