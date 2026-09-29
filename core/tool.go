package core

// name of the tool that requests a sub-session. ProcessQuestion intercepts this
// tool by name and delegates to Harness.RunSubSession instead of calling the tool runner,
// because the tool package cannot import the agent package that drives the sub-session.
const SubSessionToolName = "CreateSubSession"

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
	// get agent session runtime context
	GetContext() Context
	// is destructive
	IsDestructive() bool
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
// accepts rootPath for resolving schema files, and session for accessing context, skills, and knowledge bases at runtime.
type ToolFactory func(rootPath string, session Session) Tool

// toolRegistry stores registered tool factories.
// populated during init() which is single-threaded.
var toolRegistry = make(map[string]ToolFactory)

// RegisterTool registers a tool factory with the given name.
// No lock needed - init() is single-threaded.
func RegisterTool(name string, factory ToolFactory) {
	toolRegistry[name] = factory
}

// CreateTool creates a tool instance by name.
// used by harness to dynamically create tools with session.
func CreateTool(name string, rootPath string, session Session) Tool {
	if factory, ok := toolRegistry[name]; ok {
		return factory(rootPath, session)
	}
	return nil
}

// GetTool returns the tool instance by name.
// changed from GetToolDescription to return Tool instead of string.
func GetTool(name string, rootPath string, session Session) Tool {
	return CreateTool(name, rootPath, session)
}
