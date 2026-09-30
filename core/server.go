package core

import (
	"context"
	"time"
)

// pure server, without UI
type AgentServer interface {
	SingalManager
	ServiceProvider
	// set agent core
	SetAgentCore(AgentCore)
	// get agent core
	GetAgentCore() AgentCore
	// set config
	SetAppConfig(AppConfig)
	// get config
	GetAppConfig() AppConfig
	// set router
	SetRouter()
	// start
	Start()
	// stop
	Stop()
}

func InitAgentServer(server AgentServer, agent AgentCore, appConfig AppConfig) AgentServer {
	server.SetAppConfig(appConfig)
	server.SetAgentCore(agent)
	server.SetRouter()

	return server
}

// using go routine outside
func StartAgentServer(server AgentServer) {
	server.Start()
}

// stop with graceful shutdown before timeout
func StopAgentServer(server AgentServer) {
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(done)
		server.Stop()
	}()

	select {
	case <-done:
		LogStd(LogLevelInfo, "graceful shutdown completed successfully")
	case <-ctx.Done():
		LogStd(LogLevelWarn, "graceful shutdown timeout after 30s, forcing exit...")
	}
}
