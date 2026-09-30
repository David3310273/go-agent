package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	simpleApp "github.com/David3310273/go-agent/app/server"
	"github.com/David3310273/go-agent/core"
	_ "github.com/David3310273/go-agent/providers/qwen"
)

// AppConfig represents the app-level configuration from config.json

// ServiceConfig represents the HTTP service configuration

// loadAppConfig reads and parses the app config.json file
func loadAppConfig(configPath string) (*core.AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config core.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
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
// rootPath parameter for resolving all runtime file paths
func initAgent(rootPath string) (*simple.SimpleAgent, *core.Diagnostic) {
	agent, diag := simple.NewSimpleAgent(rootPath)
	if diag != nil {
		return nil, diag
	}

	// set agent ID
	if diag := agent.SetID(); diag != nil {
		return nil, diag
	}

	// create providers from registry and set on agent before start
	providers := simple.CreateProviders(rootPath, agent.Configs.Agent)
	if diag := agent.SetProviders(providers); diag != nil {
		return nil, diag
	}

	return agent, nil
}

func main() {
	// 1. load app config
	cwd, err := os.Getwd()
	if err != nil {
		core.LogError("", "failed to get working directory: %v", err)
		os.Exit(1)
	}

	configPath := path.Join(cwd, "config.json")
	appConfig, err := loadAppConfig(configPath)
	if err != nil {
		core.LogError("", "failed to load app config: %v", err)
		os.Exit(1)
	}

	// 2. init logger
	initLogger(appConfig)
	core.LogInfo("", "app starting, config loaded from %s", configPath)

	// 3. init agent
	agent, initErr := initAgent(appConfig.RootPath)
	if initErr != nil {
		core.LogError("", "failed to init agent: %s", initErr.ToString())
		os.Exit(1)
	}
	// 4. init server
	server := simpleApp.SimpleAgentServer{}
	core.InitAgentServer(&server, agent, *appConfig)

	// 5. start server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go core.StartAgentServer(&server)

	// waiting for signal
	sig := <-sigChan
	core.LogInfo("", "received signal: %v", sig)

	// graceful shutdown
	core.StopAgentServer(&server)

	time.Sleep(2 * time.Second)
	core.LogInfo("", "graceful shutdown completed")
}
