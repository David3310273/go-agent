package controller

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

var _ core.CommandParser = (*SimpleCommandParser)(nil)

type SimpleCommandParser struct {
}

func (p SimpleCommandParser) Parse(cmd string) (core.Command, []core.Diagnostic) {
	cmdSlices := strings.Split(cmd, " ")
	if len(cmdSlices) == 0 {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("invalid command: %s", cmd),
			},
		}
	}
	switch cmdSlices[0] {
	case model.AskCommandName:
		command, diagnostics := parseAskCommand(cmdSlices)
		if len(diagnostics) > 0 {
			return nil, diagnostics
		}
		return command, nil
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("invalid command: %s", cmd),
			},
		}
	}
}
