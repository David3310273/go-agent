package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/sandbox"
)

const (
	MaxBytesAllowed = 10 * 1024 * 1024 // 10K
)

func init() {
	// register FileWriterCall tool factory
	core.RegisterTool("WriteToFile", func(rootPath string, session core.Session) core.Tool {
		config := LoadToolConfig(rootPath, FileWriterSchemaPath, "filewriter.config.json")
		config.Sandbox.RootPath = rootPath
		return FileWriterCall{
			RootPath:      rootPath,
			Session:       session,
			SandboxConfig: config.Sandbox,
			Name:          config.Name,
			Schema:        config.Schema,
			IsDangerous:   config.IsDestructive,
		}
	})
}

// FileWriterCall, file writer tool implementing core.Tool interface
type FileWriterCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	//  project root path for resolving schema file path
	RootPath string
	// session for accessing runtime resources.
	Session       core.Session
	SandboxConfig *core.SandBoxConfig
	IsDangerous   bool
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f FileWriterCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f FileWriterCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	content, err := os.ReadFile(path.Join(f.RootPath, FileWriterSchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "filewriter: failed to read schema: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}

	return schema
}

// GetDescription returns the tool description from schema
func (f FileWriterCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f FileWriterCall) GetContext() core.Context {
	return f.Session.GetContext()
}

func (f FileWriterCall) IsDestructive() bool {
	return f.IsDangerous
}

func (f FileWriterCall) ToShellScript(args map[string]any) string {
	finalPath := fmt.Sprintf("%s/%s", f.RootPath, args["path"].(string))
	cmd := fmt.Sprintf("echo \"%s\" > %s && echo Success", args["content"].(string), finalPath)
	return cmd
}

func (f FileWriterCall) GetSandbox() core.Sandbox {
	if f.SandboxConfig == nil {
		return nil
	}

	localSandbox := sandbox.NewLocalSandbox(*f.SandboxConfig)
	if err := core.InitSandbox(localSandbox); err != nil {
		core.LogStd(core.LogLevelWarn, "filewriter: failed to init sandbox %#v: %v", localSandbox, err)
		return nil
	}

	return localSandbox
}

// Validate validates the tool configuration
func (f FileWriterCall) Validate(args map[string]any) *core.Diagnostic {
	pathArg, _ := args["path"].(string)
	// check for path traversal
	if strings.Contains(pathArg, "./") {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: fmt.Sprintf("Potential path traversal detected: %s, refuse to execute", pathArg),
		}
	}
	// check content size
	content, _ := args["content"].(string)
	if len(content) > MaxBytesAllowed {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: fmt.Sprintf("Content size %d exceeds the maximum allowed %d", len(content), MaxBytesAllowed),
		}
	}

	return nil
}

// GetRunner returns the runner function for FileWriterCall
func (f FileWriterCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		filePath, _ := args["path"].(string)
		content, _ := args["content"].(string)
		diag := WriteToFile(filePath, content)
		if diag != nil && diag.Level == core.SeverityError {
			return "", diag
		}
		return "Success", &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}

func WriteToFile(filename string, content string) *core.Diagnostic {
	cwd, _ := os.Getwd()

	completePath := path.Join(cwd, filename)
	if err := os.WriteFile(completePath, []byte(content), 0644); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: err.Error(),
		}
	}

	return nil
}
