package model

import (
	"fmt"

	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*QuitCommand)(nil)

type QuitCommand struct {
}

const (
	QuitCommandName = "/quit"
)

func (c QuitCommand) GetHelp() map[string]string {
	return map[string]string{
		"format": "/quit",
		"desc":   "exit the command line application",
	}
}

func (c QuitCommand) WithOption(key string, value string) core.Command {
	return c
}

func (c QuitCommand) GetBase() string {
	return QuitCommandName
}

func (c QuitCommand) Run(agent *core.AgentCore, question core.Question) *core.Diagnostic {
	fmt.Println("Exiting...")
	err := core.StopAgentCore(*agent)
	if len(err) > 0 {
		core.LogStd(core.LogLevelError, "agent %s stopped error: %v", (*agent).GetID(), core.DiagnosticList(err).ToString())
	} else {
		core.LogStd(core.LogLevelInfo, "agent %s stopped", (*agent).GetID())
	}
	return nil
}
