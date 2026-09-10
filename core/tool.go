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
	// get runner function that accepts args map and returns result string + diagnostic
	// auto-added: changed return type to support returning data to model.
	GetRunner() func(args map[string]any) (string, *Diagnostic)
}

// CallTool validates and executes a tool with given args.
// auto-added: changed to return result string + diagnostic.
func CallTool(tool Tool, args map[string]any) (string, *Diagnostic) {
	if diag := tool.Validate(args); diag != nil {
		return "", diag
	}
	runner := tool.GetRunner()
	return runner(args)
}
