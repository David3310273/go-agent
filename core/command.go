package core

// command line interface, take interface param as input
type Command interface {
	// return help message
	GetHelp() map[string]string
	// with option
	WithOption(string, string) Command
	// get base
	GetBase() string
	// run
	Run(*AgentCore, Question) *Diagnostic
}

type CommandParser interface {
	Parse(string) (Command, []Diagnostic)
}

// command app for agent
type CommandApp interface {
	UI
	CommandParser
	UserManager
	I18n
}
