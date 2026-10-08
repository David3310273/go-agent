package main

import (
	"fmt"
	"os"
	"path"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/cmd/config"
	"github.com/David3310273/go-agent/cmd/ui"
	"github.com/David3310273/go-agent/core"
	_ "github.com/David3310273/go-agent/providers/qwen"
)

func main() {
	// get working directory
	cwd, err := os.Getwd()
	if err != nil {
		core.LogStd(core.LogLevelError, "failed to get working directory: %v", err.Error())
		os.Exit(1)
	}

	// load config first
	configPath := path.Join(cwd, "config.json")
	config, configErr := config.LoadAppConfig(configPath)
	if configErr != nil {
		core.LogStd(core.LogLevelError, "failed to load app config: %v", configErr.Message)
		os.Exit(1)
	}

	agent, diag := simple.NewSimpleAgent(config.RootPath)
	if diag != nil {
		core.LogStd(core.LogLevelError, "failed to create agent: %v", diag.ToString())
		os.Exit(1)
	}

	// redirect internal logs to file, keep stdout clean for CLI interaction
	logPath := path.Join(cwd, "logs", fmt.Sprintf("agent-%s.log", agent.GetID()))
	core.SetDefaultLogPath(logPath)
	core.LogStd(core.LogLevelInfo, "agent logs written to %s", logPath)
	core.LogStd(core.LogLevelInfo, "cmd app starting, config %#v loaded from %s", config, configPath)

	providers := simple.CreateProviders(config.RootPath, agent.Configs.Agent)
	if diag := agent.SetProviders(providers); diag != nil {
		core.LogStd(core.LogLevelError, "failed to set providers: %v", diag.ToString())
		os.Exit(1)
	}

	go func() {
		diagnostics := core.StartAgentCore(agent, config.AppConfig)
		if len(diagnostics) > 0 {
			for _, d := range diagnostics {
				if d.Level >= core.SeverityError {
					core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
				}
			}
		}
	}()
	// start command app with bubbletea UI
	if err := ui.StartUI(agent, *config); err != nil {
		core.LogStd(core.LogLevelError, "Command app error: %v", err)
		os.Exit(1)
	}
}
