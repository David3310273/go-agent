package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/David3310273/go-agent/app/server"
	"github.com/David3310273/go-agent/core"
	_ "github.com/David3310273/go-agent/providers/qwen"
	_ "github.com/David3310273/go-agent/sandbox"
)

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

func main() {
	// 1. load config
	cwd, err := os.Getwd()
	if err != nil {
		core.LogError("", "%v", err)
		os.Exit(1)
	}

	configPath := path.Join(cwd, "config.json")
	appConfig, err := loadAppConfig(configPath)
	if err != nil {
		core.LogError("", "%v", err)
		os.Exit(1)
	}

	// 2. init app (init logger → init agent → wire server)
	app, err := server.NewSimpleApp(appConfig)
	if err != nil {
		core.LogError("", "%v", err)
		os.Exit(1)
	}

	// 3. start server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go app.Start()

	// waiting for signal
	sig := <-sigChan
	core.LogInfo("", "received signal: %v", sig)

	// graceful shutdown
	app.Stop()

	time.Sleep(2 * time.Second)
	core.LogInfo("", "graceful shutdown completed")
}
