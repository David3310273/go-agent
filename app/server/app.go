package server

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

// compile-time check that SimpleApp implements core.AgentApp
var _ core.AgentApp = (*SimpleApp)(nil)

type SimpleApp struct {
	server core.AgentServer
	config core.AppConfig
	router *gin.Engine
}

func NewSimpleApp(appConfig *core.AppConfig) (*SimpleApp, error) {
	// 1. init logger
	initLogger(appConfig)
	core.LogInfo("", "app starting, config loaded")

	// 2. init agent
	agent, initErr := initAgent(appConfig.RootPath)
	if initErr != nil {
		return nil, fmt.Errorf("failed to init agent: %s", initErr.ToString())
	}

	// 3. create app + server, wire dependencies
	app := &SimpleApp{
		config: *appConfig,
	}
	server := NewSimpleAgentServer(agent)
	app.server = server
	core.InitAgentServer(server, agent, *appConfig)
	app.SetRouter()

	return app, nil
}

func (s *SimpleApp) SetRouter() {
	s.router = SetupRouter(s)
}

func (s *SimpleApp) BeforeStart() []core.Diagnostic {
	return nil
}

func (s *SimpleApp) BeforeStop() []core.Diagnostic {
	return nil
}

// initLogger initialises the global structured logger singleton.
func initLogger(appConfig *core.AppConfig) {
	logPath := fmt.Sprintf(appConfig.LogPath, time.Now().Format(time.RFC3339))
	logFile := path.Join(appConfig.RootPath, logPath)
	logDir := path.Dir(logFile)

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0755)
		if err != nil {
			core.LogError("", "failed to create log directory: %v", err)
			os.Exit(1)
		}
	}
}

// initAgent creates, configures and starts the agent
func initAgent(rootPath string) (core.AgentCore, *core.Diagnostic) {
	agent, diag := simple.NewSimpleAgent(rootPath)
	if diag != nil {
		return nil, diag
	}

	providers := simple.CreateProviders(rootPath, agent.Configs.Agent.QuestionProvider)
	if diag := agent.SetProviders(providers); diag != nil {
		return nil, diag
	}

	return agent, nil
}

// UI interface

func (s *SimpleApp) Render(code core.MessageCode, locale core.LanguageType) string {
	return ""
}

func (s *SimpleApp) RenderSystemMessage(diagnostics *core.DiagnosticList, locale core.LanguageType) string {
	return ""
}

func (s *SimpleApp) Welcome() string {
	return ""
}

// I18n interface

func (s *SimpleApp) Translate(key core.MessageCode, targetLanguage core.LanguageType) string {
	return ""
}

// SingalManager interface (via AgentServer)

func (s *SimpleApp) GracefulQuit() {
	s.server.GracefulQuit()
}

// AgentServer interface

func (s *SimpleApp) SetAgentCore(agent core.AgentCore) {
	s.server.SetAgentCore(agent)
}

func (s *SimpleApp) GetAgentCore() core.AgentCore {
	return s.server.GetAgentCore()
}

func (s *SimpleApp) SetAppConfig(config core.AppConfig) {
	s.config = config
}

func (s *SimpleApp) GetAppConfig() core.AppConfig {
	return s.config
}

func (s *SimpleApp) Start() []core.Diagnostic {
	// start agent core
	s.server.Start()

	// block on HTTP listen
	addr := fmt.Sprintf(":%d", s.config.Service.Port)
	core.LogStd(core.LogLevelInfo, "HTTP server listening on %s", addr)
	if err := s.router.Run(addr); err != nil {
		core.LogStd(core.LogLevelError, "HTTP server error: %v", err)
	}
	return nil
}

func (s *SimpleApp) Stop() []core.Diagnostic {
	s.server.Stop()
	return nil
}

// UserManager interface

func (s *SimpleApp) AuthUser(token string) *core.Diagnostic {
	return nil
}

func (s *SimpleApp) GetUserPlan(userID string) (core.Plan, *core.Diagnostic) {
	return core.Plan{}, nil
}

func (s *SimpleApp) CheckUserPlan(userInfo any, plan core.Plan) *core.Diagnostic {
	return nil
}
