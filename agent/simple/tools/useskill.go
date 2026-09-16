package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
)

const (
	SchemaPath = "agent/simple/tools"
)

func init() {
	// register UseSkillCall tool factory
	// updated to accept Context instead of skillDefinitions.
	core.RegisterTool("UseSkill", func(rootPath string, context core.Context) core.Tool {
		return UseSkillCall{
			Name:     "UseSkill",
			Schema:   "useskill.schema.json",
			RootPath: rootPath,
			Context:  context,
		}
	})
}

// UseSkillCall implements core.Tool interface for loading skills dynamically.
// loads skill by name and returns skill info with tool list.
// changed SkillDefinitions to Context for unified access to skills and knowledge bases.
type UseSkillCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	// project root path for resolving schema file path
	RootPath string
	// context for accessing skill definitions at runtime.
	Context core.Context
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f UseSkillCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f UseSkillCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	// use RootPath instead of hardcoded relative path
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}

	return schema
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
	for _, skill := range f.Context.GetSkills() {
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
		for i := range f.Context.GetSkills() {
			if f.Context.GetSkills()[i].Name == name {
				skillDef = &f.Context.GetSkills()[i]
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
		fmt.Fprintf(&response, "Skill: %s\n", skillDef.Name)
		fmt.Fprintf(&response, "Description: %s\n", skillDef.Description)
		fmt.Fprintf(&response, "\nTools to execute (with descriptions):\n")
		for _, toolName := range skillDef.Tools {
			// use GetTool to get tool instance, then call GetDescription.
			tool := core.GetTool(toolName, f.RootPath, f.Context)
			if tool != nil {
				fmt.Fprintf(&response, "- **%s**: %s\n", toolName, tool.GetDescription())
			}
		}
		fmt.Fprintf(&response, "\nPlease call these tools with appropriate parameters.")

		return response.String(), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
