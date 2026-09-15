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

type Skill interface {
	// get skill schema in provider-agnostic format
	GetName() string
	// get skill description
	GetDescription() string
	// get tools
	GetTools() []Tool
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
	// changed return type to support returning data to model.
	GetRunner() func(args map[string]any) (string, *Diagnostic)
}

// CallTool validates and executes a tool with given args.
// changed to return result string + diagnostic.
func CallTool(tool Tool, args map[string]any) (string, *Diagnostic) {
	if diag := tool.Validate(args); diag != nil {
		return "", diag
	}
	runner := tool.GetRunner()
	return runner(args)
}

// ToolFactory is a function that creates a Tool instance.
// factory function type for tool registration.
// accepts Context for accessing skills and knowledge bases at runtime.
type ToolFactory func(rootPath string, context Context) Tool

// toolRegistry stores registered tool factories.
// populated during init() which is single-threaded.
var toolRegistry = make(map[string]ToolFactory)

// RegisterTool registers a tool factory with the given name.
// No lock needed - init() is single-threaded.
func RegisterTool(name string, factory ToolFactory) {
	toolRegistry[name] = factory
}

// CreateTool creates a tool instance by name.
// used by harness to dynamically create tools with context.
func CreateTool(name string, rootPath string, context Context) Tool {
	if factory, ok := toolRegistry[name]; ok {
		return factory(rootPath, context)
	}
	return nil
}

// GetToolDescription returns the tool description by name.
// used to get tool description for skill info.
// updated to accept Context instead of skillDefinitions.
func GetToolDescription(name string, rootPath string, context Context) string {
	tool := CreateTool(name, rootPath, context)
	if tool == nil {
		return ""
	}
	return tool.GetDescription()
}
