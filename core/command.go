package core

// TODO: support command line in the future
type CommandAccessible interface {
	// command parser
	Parse(string) (any, []Diagnostic)
	// return help message
	GetHelp() string
}
