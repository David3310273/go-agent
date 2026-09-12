package tools

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"time"

	"github.com/David3310273/go-agent/core"
)

// FileWriterCall, file writer tool implementing core.Tool interface
type GetDateCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	//  project root path for resolving schema file path
	RootPath string
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f GetDateCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f GetDateCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	//  use RootPath instead of hardcoded relative path
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
func (f GetDateCall) GetDescription() string {
	return f.GetSchema().Function.Description
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
