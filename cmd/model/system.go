package model

import (
	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*SystemCommand)(nil)

type SystemCommand struct {
	SubCommand   string
	Language     string
	ResponseChan chan core.Diagnostic
}

const (
	HelpCommandName    = "/help"
	NewPageCommandName = "/new-page"
	QuitCommandName    = "/quit"
	ClearCommandName   = "/clear"
	I18nCommandName    = "/i18n"

	SystemCommandName = "system"

	SystemSubCommandHelp    = "help"
	SystemSubCommandNewPage = "new-page"
	SystemSubCommandQuit    = "quit"
	SystemSubCommandClear   = "clear"
	SystemSubCommandI18n    = "i18n"
)

func init() {
	// Fake command just for uniform help message.
	RegisterCommand(HelpCommandName, func() core.Command {
		return &SystemCommand{}
	})
}

func (c SystemCommand) GetHelp() map[string]any {
	return map[string]any{
		"/help": map[string]any{
			"format":  "/help | /h",
			"desc":    "show help page",
			"options": map[string]string{},
		},
		"/new-page": map[string]any{
			"format":  "/new-page",
			"desc":    "create a new page",
			"options": map[string]string{},
		},
		"/quit": map[string]any{
			"format":  "/quit | /q",
			"desc":    "quit the agent",
			"options": map[string]string{},
		},
		"/clear": map[string]any{
			"format":  "/clear | /clr",
			"desc":    "clear the knowledge base",
			"options": map[string]string{},
		},
		"/i18n": map[string]any{
			"format": "/i18n <language type, en_US|zh_CN>",
			"desc":   "set the language of the agent",
			"options": map[string]string{
				"language": "set the language of the agent",
			},
		},
	}
}

func (c *SystemCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "language":
		c.Language = value
	default:
		return c
	}
	return c
}

func (c SystemCommand) GetBase() string {
	return SystemCommandName
}

func (c *SystemCommand) Run(app core.CommandApp) *core.Diagnostic {
	return nil
}

func (c *SystemCommand) GetOutputChans() []core.OutputChannel {
	return nil
}
