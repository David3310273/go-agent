package core

// output channel descriptor for UI to listen on
type OutputChannel struct {
	Chan chan Answer
	Type string // "response", "hint", or any custom type
}

// command line interface, take interface param as input
type Command interface {
	// return help message
	GetHelp() map[string]any
	// with option
	WithOption(string, string) Command
	// get base
	GetBase() string
	// run
	Run(app CommandApp) *Diagnostic
	// get output channels for UI to listen on
	GetOutputChans() []OutputChannel
}

type CommandParser interface {
	Parse(string) (Command, []Diagnostic)
}

type CommandAppConfig struct {
	AppConfig
	I18nConfig
}

// command app for agent
type CommandApp interface {
	UI
	WorkFlow
	CommandParser
	UserManager
	I18n
	SingalManager
	// get user input
	GetUserInput() chan Command
	// get agent core
	GetAgentCore() *AgentCore
	// get config
	GetConfig() CommandAppConfig
	// get help
	GetHelpDoc() string
}

func StartCommandApp(app CommandApp) DiagnosticList {
	diag := app.BeforeStart()
	if diag != nil {
		return diag
	}

	diag = app.Start()
	if diag != nil {
		return diag
	}
	return nil
}

func StopCommandApp(app CommandApp) DiagnosticList {
	diag := app.BeforeStop()
	if diag != nil {
		return diag
	}
	return app.Stop()
}
