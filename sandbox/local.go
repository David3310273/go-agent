package sandbox

import (
	exec "os/exec"
	"strings"

	"github.com/David3310273/go-agent/core"
)

type LocalSandbox struct {
	RootPath    string
	ConnectInfo map[string]any
	stub        []string
}

var _ core.Sandbox = (*LocalSandbox)(nil)

func NewLocalSandbox(config core.SandBoxConfig) *LocalSandbox {
	return &LocalSandbox{
		RootPath:    config.RootPath,
		ConnectInfo: config.Config,
		stub:        []string{},
	}
}

// For local sandbox, connect doesn't connect to anything. Like mock sandbox.
func (s *LocalSandbox) Connect() error {
	s.WithStub([]string{"bash", "-c"})
	return nil
}

// Close releases sandbox resources
func (s *LocalSandbox) Close() error {
	return nil
}

// Stub returns the sandbox stub instance for executing tools
func (s *LocalSandbox) WithStub(stub []string) core.Sandbox {
	s.stub = stub
	return s
}

func (s *LocalSandbox) DryRun(tool core.Tool, args map[string]any) string {
	cmd := tool.ToShellScript(args)
	base := s.stub

	finalCmd := append(base, cmd)
	return strings.Join(finalCmd, " ")
}

// Run executes the tool in the sandbox
func (s *LocalSandbox) Run(tool core.Tool, args map[string]any) (string, *core.Diagnostic) {
	cmd := tool.ToShellScript(args)
	base := s.stub

	finalCmd := append(base, cmd)

	result, err := exec.Command(finalCmd[0], finalCmd[1:]...).CombinedOutput()
	if err != nil {
		return "", &core.Diagnostic{
			Level:   core.SeverityError,
			Message: err.Error(),
		}
	}

	return string(result), nil
}
