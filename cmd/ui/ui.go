package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/David3310273/go-agent/cmd/config"
	"github.com/David3310273/go-agent/core"
	tea "github.com/charmbracelet/bubbletea"
)

func StartUI(agent core.AgentCore, config config.CommandAppConfig) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	adapter := NewAgentAdapter(agent, config)

	p := tea.NewProgram(NewModel(adapter))

	go func() {
		sig := <-sigChan
		core.LogStd(core.LogLevelInfo, "received signal: %v, shutting down...", sig)
		adapter.Stop()
		p.Quit()
		os.Exit(0)
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
