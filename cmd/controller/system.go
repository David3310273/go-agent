package controller

import (
	"flag"
	"fmt"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

func parseSystemCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	fs := flag.NewFlagSet("system", flag.ContinueOnError)
	fs.SetOutput(nil)

	language := fs.String("language", "", "language type")

	err := fs.Parse(cmdSlices[1:])
	if err != nil {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("failed to parse system command: %v", err),
			},
		}
	}

	command := &model.SystemCommand{
		Language: *language,
	}

	switch cmdSlices[0] {
	case model.HelpCommandName, "/h":
		command.SubCommand = model.SystemSubCommandHelp
	case model.NewPageCommandName:
		command.SubCommand = model.SystemSubCommandNewPage
	case model.QuitCommandName, "/q":
		command.SubCommand = model.SystemSubCommandQuit
	case model.ClearCommandName, "/clr":
		command.SubCommand = model.SystemSubCommandClear
	case model.I18nCommandName:
		command.SubCommand = model.SystemSubCommandI18n
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("unknown system command: %s", cmdSlices[0]),
			},
		}
	}

	return command, nil
}
