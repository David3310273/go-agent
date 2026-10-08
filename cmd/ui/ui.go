package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/David3310273/go-agent/cmd/app"
	"github.com/David3310273/go-agent/core"
	tea "github.com/charmbracelet/bubbletea"
	uuid "github.com/gofrs/uuid/v5"
)

func StartUI(app *app.SimpleCommandApp) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	id, _ := uuid.NewV4()
	sessionID := id.String()

	model := NewModel(app, sessionID)

	p := tea.NewProgram(&model)

	go func() {
		sig := <-sigChan
		fmt.Printf("received signal: %v, shutting down...\n", sig)

		var diag core.DiagnosticList = core.StopCommandApp(app)
		if len(diag) > 0 {
			core.LogStd(core.LogLevelError, "error shutting down: %v\n", diag.ToString())
		}

		p.Quit()
		os.Exit(0)
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
