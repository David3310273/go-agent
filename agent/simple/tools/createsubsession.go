package tools

import (
	"encoding/json"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

func init() {
	// register CreateSubSession tool factory
	core.RegisterTool(core.SubSessionToolName, func(rootPath string, session core.Session) core.Tool {
		return CreateSubSessionCall{
			RootPath: rootPath,
			Session:  session,
			Schema:   "createsubsession.schema.json",
		}
	})
}

// CreateSubSessionCall implements core.Tool interface for requesting a sub-session.
// this tool only declares the intent. The sub-session is created and driven by
// SimpleHarness.RunSubSession, which core.ProcessQuestion calls when it intercepts this tool
// by name, because this package cannot import agent/simple.
type CreateSubSessionCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	// project root path for resolving schema file path
	RootPath string
	// session for accessing context at runtime.
	Session core.Session
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f CreateSubSessionCall) GetName() string {
	return f.GetSchema().Function.Name
}

// GetSchema implements core.Tool interface, returns provider-agnostic ToolSchema
func (f CreateSubSessionCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	// use RootPath instead of hardcoded relative path
	content, err := os.ReadFile(path.Join(f.RootPath, CreateSubsessionSchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "createsubsession: failed to read schema: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		core.LogStd(core.LogLevelWarn, "createsubsession: failed to unmarshal schema: %v", err)
		return schema
	}

	return schema
}

// GetDescription returns the tool description from schema
func (f CreateSubSessionCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f CreateSubSessionCall) GetContext() core.Context {
	return f.Session.GetContext()
}

// Validate validates the tool arguments.
// implements core.Tool interface.
// checks the query the sub-session has to work on. core.ProcessQuestion intercepts
// this tool and calls the harness instead of CallTool, so this only runs on a direct call.
func (f CreateSubSessionCall) Validate(args map[string]any) *core.Diagnostic {
	query, _ := args["query"].(string)
	if query == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: "query is required for a sub-session",
		}
	}

	return nil
}

// IsDestructive implements core.Tool interface.
// requesting a sub-session changes nothing on its own.
func (f CreateSubSessionCall) IsDestructive() bool {
	return false
}

func (f CreateSubSessionCall) ToShellScript(args map[string]any) string {
	return ""
}

func (f CreateSubSessionCall) GetAvailableSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
}

// GetRunner implements core.Tool interface.
// the sub-session is created and driven by SimpleHarness.RunSubSession, so this
// runner is never reached through the reAct loop. It reports that instead of pretending a
// sub-session was created.
func (f CreateSubSessionCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		return "", &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: "CreateSubSession is handled by the harness, call it through the reAct loop",
		}
	}
}
