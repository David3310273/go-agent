package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/David3310273/go-agent/benchmarker/simple"
	"github.com/David3310273/go-agent/core"
)

func main() {
	// 1. load config
	cwd, err := os.Getwd()
	if err != nil {
		core.LogError("metrics", "failed to get working directory: %v", err)
		os.Exit(1)
	}

	configPath := path.Join(cwd, "benchmarker", "config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		core.LogError("metrics", "failed to load config: %v", err)
		os.Exit(1)
	}

	// 2. create benchmarker (implements core.WorkFlow lifecycle)
	benchmarker := simple.NewBenchmarker(*config)

	// 3. start benchmarker
	if diag := core.StartBenchmark(benchmarker); diag != nil {
		core.LogError("metrics", "start benchmarker failed: %v", diag)
		os.Exit(1)
	}

	// 4. wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	core.LogInfo("metrics", "Received signal %v, shutting down...", sig)

	// 5. stop benchmarker
	if diag := core.StopBenchmark(benchmarker); diag != nil {
		core.LogError("metrics", "stop benchmarker failed: %v", diag)
	}
}

func loadConfig(configPath string) (*core.BenchmarkerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config core.BenchmarkerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
