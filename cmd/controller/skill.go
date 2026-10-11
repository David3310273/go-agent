package controller

import (
	"flag"
	"fmt"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

func parseSkillCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	if len(cmdSlices) < 2 {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "skill subcommand is required, usage: /skill ls [--name skill_name]",
			},
		}
	}

	subCommand := cmdSlices[1]

	fs := flag.NewFlagSet("skill", flag.ContinueOnError)
	fs.SetOutput(nil)

	name := fs.String("name", "", "skill name filter")

	err := fs.Parse(cmdSlices[2:])
	if err != nil {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("failed to parse skill command: %v", err),
			},
		}
	}

	switch subCommand {
	case model.SkillSubCommandLs:
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("unknown skill subcommand: %s, expected: ls", subCommand),
			},
		}
	}

	cmd := &model.SkillCommand{
		SubCommand:   subCommand,
		Name:         *name,
		ResponseChan: make(chan core.Diagnostic, 1),
	}

	return cmd, nil
}
