package main

import (
	"encoding/json"
	"fmt"
	"log"
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

// initLogger creates and returns a logger instance
func initLogger(appConfig *core.AppConfig) *log.Logger {
	//  use RootPath instead of os.Getwd()
	logPath := fmt.Sprintf(appConfig.LogPath, time.Now().Format(time.RFC3339))
	logFile := path.Join(appConfig.RootPath, logPath)

	return core.NewLogger(logFile)
}

// initAgent creates, configures and starts the agent
// rootPath parameter for resolving all runtime file paths
func initAgent(rootPath string) (*simple.SimpleAgent, *core.Diagnostic) {
	agent := simple.NewSimpleAgent()
	//  set RootPath on agent so GetConfigPath and other methods can use it
	agent.RootPath = rootPath

	// load agent core config
	agentConfig, diag := agent.LoadConfigs("")
	if diag != nil {
		return nil, diag
	}

	agent.Configs = agentConfig
	// set agent ID
	if diag := agent.SetID(); diag != nil {
		return nil, diag
	}

	// create providers from registry and set on agent before start
	providers := simple.CreateProviders(rootPath)
	if diag := agent.SetProviders(providers); diag != nil {
		return nil, diag
	}

	return agent, nil
}

func main() {
	// 1. load app config
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	configPath := path.Join(cwd, "config.json")
	appConfig, err := loadAppConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load app config: %v", err)
	}

	// 2. init logger
	logger := initLogger(appConfig)
	logger.Printf("app starting, config loaded from %s", configPath)

	// 3. init agent
	agent, initErr := initAgent(appConfig.RootPath)
	if initErr != nil {
		log.Fatalf("failed to init agent: %v", initErr.ToString())
	}
	// 4. init server
	server := simpleApp.SimpleAgentServer{}
	server.SetLogger(logger)
	core.InitAgentServer(&server, agent, *appConfig, logger)

	// 5. start server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go core.StartAgentServer(&server)

	// waiting for signal
	sig := <-sigChan
	logger.Printf("\nReceived signal: %v\n", sig)

	// graceful shutdown
	core.StopAgentServer(&server)

	time.Sleep(2 * time.Second)
	logger.Printf("Graceful shutdown completed")
}
