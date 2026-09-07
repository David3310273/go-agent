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
	SchemaPath      = "tools/filewriter.schema.json"
	MaxBytesAllowed = 10 * 1024 * 1024 // 10K
)

// FileWriterCall, file writer tool implementing core.Tool interface
type FileWriterCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f FileWriterCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f FileWriterCall) GetSchema() core.ToolSchema {
	cwd, _ := os.Getwd()
	var schema core.ToolSchema

	// hard code here, for simplicity
	content, err := os.ReadFile(path.Join(cwd, "..", "agent/simple/tools", f.Schema))
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
func (f FileWriterCall) GetDescription() string {
	return f.GetSchema().Function.Description
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
func (f FileWriterCall) GetRunner() func(args map[string]any) *core.Diagnostic {
	return func(args map[string]any) *core.Diagnostic {
		filePath, _ := args["path"].(string)
		content, _ := args["content"].(string)
		return WriteToFile(filePath, content)
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
