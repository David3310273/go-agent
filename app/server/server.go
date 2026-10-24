package server

import (
	"context"

	"github.com/David3310273/go-agent/core"
)

type SimpleAgentServer struct {
	agent  core.AgentCore
	config core.AppConfig
	ctx    context.Context
	cancel context.CancelFunc
}

func NewSimpleAgentServer(agent core.AgentCore) *SimpleAgentServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SimpleAgentServer{
		agent:  agent,
		ctx:    ctx,
		cancel: cancel,
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

func (s *SimpleAgentServer) dispatch() {
	for {
		select {
		case query, ok := <-s.agent.GetQuestionChan():
			if !ok {
				return
			}
			sessionID := query.GetSessionID()
			session, err := s.agent.GetSessionOnCreate(sessionID, true)
			if err != nil {
				core.LogStd(core.LogLevelError, "failed to create session: %s", err.Message)
				select {
				case query.GetResponseChan() <- err:
				default:
				}
			} else {
				session.GetQuestionChan() <- query
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *SimpleAgentServer) Start() []core.Diagnostic {
	go s.dispatch()

	// determine worker count: config value or default to 1
	workerCount := s.config.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	core.LogStd(core.LogLevelInfo, "server starting with %d agent core workers, agentID=%s", workerCount, s.agent.GetID())

	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			core.LogStd(core.LogLevelDebug, "agent core worker %d started, agentID=%s", workerID, s.agent.GetID())
			diagnostics := core.StartAgentCore(s.agent, s.config)
			if len(diagnostics) > 0 {
				for _, d := range diagnostics {
					if d.Level >= core.SeverityError {
						core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
					}
				}
			}
		}(i)
	}

	return nil
}

// stop in server layer
func (s SimpleAgentServer) Stop() []core.Diagnostic {
	s.GracefulQuit()
	return nil
}

// graceful quit, in agent layer
func (s SimpleAgentServer) GracefulQuit() {
	s.cancel()

	err := core.StopAgentCore(s.agent)
	if len(err) > 0 {
		core.LogStd(core.LogLevelError, "agent %s stopped error: %v", s.GetAgentCore().GetID(), core.DiagnosticList(err).ToString())
	} else {
		core.LogStd(core.LogLevelInfo, "agent %s stopped", s.GetAgentCore().GetID())
	}
}
