package tools

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/David3310273/go-agent/core"
)

func init() {
	// register GetDateCall tool factory
	core.RegisterTool("GetDate", func(rootPath string, session core.Session) core.Tool {
		return GetDateCall{
			Name:     "getdate",
			Schema:   "getdate.schema.json",
			RootPath: rootPath,
			Session:  session,
		}
	})
}

// GetDateCall, date tool implementing core.Tool interface
type GetDateCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	//  project root path for resolving schema file path
	RootPath string
	// session for accessing runtime resources.
	Session core.Session
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f GetDateCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f GetDateCall) IsDestructive() bool {
	return false
}

func (f GetDateCall) ToShellScript(args map[string]any) string {
	return ""
}

func (f GetDateCall) GetAvailableSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f GetDateCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	//  use RootPath instead of hardcoded relative path
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "getdate: failed to read schema: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}

	return schema
}

// GetDescription returns the tool description from schema
func (f GetDateCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f GetDateCall) GetContext() core.Context {
	return f.Session.GetContext()
}

// Validate validates the tool configuration
func (f GetDateCall) Validate(args map[string]any) *core.Diagnostic {
	return nil
}

// GetRunner returns the runner function for GetDateCall
// returns current system time to model.
func (f GetDateCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		return GetDate(), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}

func GetDate() string {
	return time.Now().Format(time.DateTime)
}
