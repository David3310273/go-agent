package model

import (
	"github.com/David3310273/go-agent/core"
)

var commandRegistry = make(map[string]func() core.Command)

func RegisterCommand(name string, factory func() core.Command) {
	commandRegistry[name] = factory
}

func GetAllCommands() map[string]core.Command {
	commands := make(map[string]core.Command)
	for name, factory := range commandRegistry {
		commands[name] = factory()
	}
	return commands
}
