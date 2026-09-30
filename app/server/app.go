package server

import (
	"fmt"

	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

type SimpleAgentServer struct {
	agent  core.AgentCore
	Config core.AppConfig
	Router *gin.Engine
}

// inject agent to app layer
func NewSimpleAgentServer(agent core.AgentCore) *SimpleAgentServer {
	return &SimpleAgentServer{
		agent: agent,
	}
}

func (s *SimpleAgentServer) SetAgentCore(agent core.AgentCore) {
	s.agent = agent
}

func (s SimpleAgentServer) GetAgentCore() core.AgentCore {
	return s.agent
}

func (s *SimpleAgentServer) SetAppConfig(config core.AppConfig) {
	s.Config = config
}

func (s SimpleAgentServer) GetAppConfig() core.AppConfig {
	return s.Config
}

// SetRouter initializes the gin router
func (s *SimpleAgentServer) SetRouter() {
	s.Router = SetupRouter(&s.Config, s.agent)
}

// starts the agent core and listens on the configured HTTP port
func (s *SimpleAgentServer) Start() {
	go func() {
		diagnostics := core.StartAgentCore(s.agent, s.Config)
		if len(diagnostics) > 0 {
			for _, d := range diagnostics {
				if d.Level >= core.SeverityError {
					core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
				}
			}
		}
	}()

	addr := fmt.Sprintf(":%d", s.Config.Service.Port)
	core.LogStd(core.LogLevelInfo, "HTTP server listening on %s", addr)

	if err := s.Router.Run(addr); err != nil {
		core.LogStd(core.LogLevelError, "HTTP server error: %v", err)
	}
}

// stop in server layer
func (s SimpleAgentServer) Stop() {
	s.GracefulQuit()
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
