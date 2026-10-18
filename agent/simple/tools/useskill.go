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
	// accepts Context for init-time creation; Session is set at runtime via SetSession.
	core.RegisterTool("UseSkill", func(ctx core.Context) core.Tool {
		config := LoadToolConfig(ctx.GetRootPath(), UseSkillSchemaPath, "useskill.config.json")
		return &UseSkillCall{
			Name:     config.Name,
			Schema:   config.Schema,
			RootPath: ctx.GetRootPath(),
			Context:  ctx,
		}
	})
}

// UseSkillCall implements core.Tool interface for loading skills dynamically.
// loads skill by name and returns skill info with tool list.
// Context is set at init time; Session is set at runtime via SetSession for SetLoadTools.
type UseSkillCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	// session for dynamic tool registration (SetLoadTools), set at runtime
	Session core.Session
	// context for accessing runtime resources
	Context core.Context
}

// SetSession sets the session at runtime, required for SetLoadTools.
func (f *UseSkillCall) SetSession(session core.Session) {
	f.Session = session
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

func (f UseSkillCall) GetSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
}

// GetDescription returns the tool description from schema
func (f UseSkillCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f UseSkillCall) GetContext() core.Context {
	return f.Context
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
	for _, skill := range *f.Context.GetSkills() {
		if skill.GetName() == name {
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
		var skillDef core.Skill
		skills := f.Context.GetSkills()
		for i := range *skills {
			if (*skills)[i].GetName() == name {
				skillDef = (*skills)[i]
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
		core.LogStd(core.LogLevelInfo, "Skill: %s\n", skillDef.GetName())
		core.LogStd(core.LogLevelInfo, "Description: %s\n", skillDef.GetDescription())
		core.LogStd(core.LogLevelInfo, "\nTools to execute (with descriptions):\n")
		for _, tool := range skillDef.GetTools() {
			// use GetTool to get tool instance, then call GetDescription.
			tool := core.GetTool(tool.GetName(), f.Context)
			if tool != nil {
				// register tool to session's loaded tools
				if f.Session != nil {
					f.Session.SetLoadTools(tool)
				}
				fmt.Fprintf(&response, "- **%s**: %s\n", tool.GetName(), tool.GetDescription())
				// add sandbox spec in skill
				supportSandbox := tool.GetSandboxEnv()
				if supportSandbox.Type != "" {
					fmt.Fprintf(&response, "- - **supported runtime environment sandbox**: type: %s, image name: %s, vm name: %s\n", supportSandbox.Type, supportSandbox.ImageName, supportSandbox.VMName)
				}
			}
		}
		fmt.Fprintf(&response, "\nPlease call these tools with appropriate parameters.")

		core.LogStd(core.LogLevelDebug, "skill loaded: name=%s, desc=%s", skillDef.GetName(), skillDef.GetDescription())

		return response.String(), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
