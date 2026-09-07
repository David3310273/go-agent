package core

// ToolSchema represents a provider-agnostic tool schema
type ToolSchema struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

// FunctionSchema represents the function definition within a tool schema
type FunctionSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// TODO: publish tool calls as mcp service
// tool should implement this if needed
type MCPAvailable[T any] interface {
	// transform data into mcp format
	Transform(result any) T
}

type Tool interface {
	// get tool schema in provider-agnostic format
	GetSchema() ToolSchema
	// get name
	GetName() string
	// get tool description
	GetDescription() string
	// validate
	Validate(args map[string]any) *Diagnostic
	// get runner function that accepts args map
	GetRunner() func(args map[string]any) *Diagnostic
}

// CallTool validates and executes a tool with given args
func CallTool(tool Tool, args map[string]any) *Diagnostic {
	if err := tool.Validate(args); err != nil {
		return err
	}
	runner := tool.GetRunner()
	return runner(args)
}
