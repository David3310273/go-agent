package core

type WorkFlow interface {
	// before start
	BeforeStart() []Diagnostic
	// start the agent
	Start() []Diagnostic

	// before stop
	BeforeStop() []Diagnostic
	// stop the agent
	Stop() []Diagnostic
}

func StartWorkFlow(w WorkFlow) []Diagnostic {
	diagnostics := w.BeforeStart()
	if len(diagnostics) > 0 {
		return diagnostics
	}

	diagnostics = w.Start()
	if len(diagnostics) > 0 {
		return diagnostics
	}

	return diagnostics
}

func StopWorkFlow(w WorkFlow) []Diagnostic {
	diagnostics := w.BeforeStop()
	if len(diagnostics) > 0 {
		return diagnostics
	}

	diagnostics = w.Stop()
	if len(diagnostics) > 0 {
		return diagnostics
	}

	return diagnostics
}
