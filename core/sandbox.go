package core

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
