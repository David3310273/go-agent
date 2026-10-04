package core

import json "encoding/json"

// sandbox is the decorator of tool calls
// start from the root path of project

type Sandbox interface {
	// execute tool in sandbox
	Connect() error
	// Stub
	WithStub(stub []string) Sandbox
	// execute tool in sandbox
	Run(tool Tool, args map[string]any) (string, *Diagnostic)
	// dryrun
	DryRun(tool Tool, args map[string]any) string
	// to description
	ToDescription() SandboxSpec
}

type SandboxSpec struct {
	Type      SandBoxType `json:"type"`
	BaseCmd   string      `json:"baseCmd"`
	ImageName string      `json:"imageName"`
	VMName    string      `json:"vmName"`
}

// SandboxFactory creates a Sandbox instance from a spec
// rootPath parameter for resolving sandbox config file paths
type SandboxFactory func(rootPath string, spec SandboxSpec) (Sandbox, *Diagnostic)

// sandboxFactories registry for independent sandbox components
var sandboxFactories = map[SandBoxType]SandboxFactory{}

// RegisterSandboxFactory registers a sandbox factory by type name
func RegisterSandboxFactory(name SandBoxType, factory SandboxFactory) {
	sandboxFactories[name] = factory
}

// GetSandboxFactory returns a registered sandbox factory by type name
func GetSandboxFactory(name SandBoxType) (SandboxFactory, bool) {
	factory, ok := sandboxFactories[name]
	return factory, ok
}

// CreateSandbox creates a sandbox instance by type and spec
func CreateSandbox(rootPath string, spec SandboxSpec) (Sandbox, *Diagnostic) {
	factory, ok := GetSandboxFactory(spec.Type)
	if !ok {
		return nil, &Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeInitSandboxError,
			Message: "unknown sandbox type: " + string(spec.Type),
		}
	}
	return factory(rootPath, spec)
}

func InitSandbox(sandbox Sandbox) *Diagnostic {
	if sandbox != nil {
		err := sandbox.Connect()
		if err != nil {
			return &Diagnostic{
				Level:   SeverityError,
				Code:    MessageCodeInitSandboxError,
				Message: err.Error(),
			}
		} else {
			return nil
		}
	}

	return &Diagnostic{
		Level:   SeverityWarn,
		Code:    MessageCodeNoSandboxInited,
		Message: "no sandbox initialized",
	}
}

// BuildSandboxForToolCall creates and connects a sandbox for a tool call.
// It compares the tool's available sandbox env with the sandbox spec from args.
// Only creates sandbox when both types match.
func BuildSandboxForToolCall(rootPath string, tool Tool, args map[string]any) (Sandbox, *Diagnostic) {
	toolSandboxSpec := tool.GetAvailableSandboxEnv()
	if toolSandboxSpec.Type == "" {
		return nil, nil
	}

	argSandbox, ok := args["sandbox"]
	if !ok {
		return nil, nil
	}

	argSandboxDesc, ok := argSandbox.(map[string]any)
	if !ok {
		return nil, nil
	}

	argSandboxData, err := json.Marshal(argSandboxDesc)
	if err != nil {
		return nil, &Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeInitSandboxError,
			Message: "failed to marshal sandbox spec from args: " + err.Error(),
		}
	}

	var argSandboxSpec SandboxSpec
	if err := json.Unmarshal(argSandboxData, &argSandboxSpec); err != nil {
		return nil, &Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeInitSandboxError,
			Message: "failed to unmarshal sandbox spec from args: " + err.Error(),
		}
	}

	if toolSandboxSpec.Type != argSandboxSpec.Type {
		return nil, nil
	}

	sandbox, diag := CreateSandbox(rootPath, argSandboxSpec)
	if diag != nil {
		return nil, diag
	}

	if err := sandbox.Connect(); err != nil {
		return nil, &Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeInitSandboxError,
			Message: err.Error(),
		}
	}

	return sandbox, nil
}
