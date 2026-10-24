package server

import (
	"github.com/David3310273/go-agent/core"
)

type SimpleAgentServer struct {
	agent  core.AgentCore
	config core.AppConfig
}

func NewSimpleAgentServer(agent core.AgentCore) *SimpleAgentServer {
	return &SimpleAgentServer{
		agent: agent,
	}
}

var _ core.AgentServer = (*SimpleAgentServer)(nil)

func (s *SimpleAgentServer) SetAgentCore(agent core.AgentCore) {
	s.agent = agent
}

func (s SimpleAgentServer) GetAgentCore() core.AgentCore {
	return s.agent
}

func (s *SimpleAgentServer) SetAppConfig(config core.AppConfig) {
	s.config = config
}

func (s SimpleAgentServer) GetAppConfig() core.AppConfig {
	return s.config
}

func (s SimpleAgentServer) BeforeStart() []core.Diagnostic {
	return nil
}

func (s SimpleAgentServer) BeforeStop() []core.Diagnostic {
	return nil
}

func (s *SimpleAgentServer) Start() []core.Diagnostic {
	initDiagnostics := core.InitAgentCore(s.agent, s.config)
	for _, d := range initDiagnostics {
		if d.Level >= core.SeverityError {
			core.LogStd(core.LogLevelError, "InitAgentCore error: %s", d.ToString())
		}
	}

	workerCount := s.config.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	core.LogStd(core.LogLevelInfo, "server starting with %d agent core workers, agentID=%s", workerCount, s.agent.GetID())

	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			core.LogStd(core.LogLevelDebug, "agent core worker %d started, agentID=%s", workerID, s.agent.GetID())
			diagnostics := core.StartAgentCore(s.agent)
			if len(diagnostics) > 0 {
				for _, d := range diagnostics {
					if d.Level >= core.SeverityError {
						core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
					}
				}
			}
		}(i)
	}

	return initDiagnostics
}

// stop in server layer
func (s SimpleAgentServer) Stop() []core.Diagnostic {
	s.GracefulQuit()
	return nil
}

// graceful quit, in agent layer
func (s SimpleAgentServer) GracefulQuit() {
	err := core.StopAgentCore(s.agent)
	if len(err) > 0 {
		core.LogStd(core.LogLevelError, "agent %s stopped error: %v", s.GetAgentCore().GetID(), core.DiagnosticList(err).ToString())
	} else {
		core.LogStd(core.LogLevelInfo, "agent %s stopped", s.GetAgentCore().GetID())
	}
}
