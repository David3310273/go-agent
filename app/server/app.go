package server

import (
	"fmt"
	"log"

	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

type SimpleAgentServer struct {
	agent  core.AgentCore
	Config core.AppConfig
	Logger *log.Logger
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

func (s *SimpleAgentServer) SetLogger(logger *log.Logger) {
	s.Logger = logger
}

func (s SimpleAgentServer) GetLogger() *log.Logger {
	return s.Logger

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
				// auto-added: only log errors, not warnings like system closed
				if d.Level >= core.SeverityError {
					s.Logger.Printf("StartAgentCore error: %s", d.ToString())
				}
			}
		}
	}()

	addr := fmt.Sprintf(":%d", s.Config.Service.Port)
	s.Logger.Printf("HTTP server listening on %s", addr)

	if err := s.Router.Run(addr); err != nil {
		s.Logger.Printf("HTTP server error: %v", err)
	}
}

// stop in server layer
func (s SimpleAgentServer) Stop() {
	s.GracefulQuit()
}

// graceful quit, in agent layer
func (s SimpleAgentServer) GracefulQuit() {
	err := core.StopAgentCore(s.agent)
	s.Logger.Printf("agent stopped error: %v", core.DiagnosticList(err).ToString())
}
