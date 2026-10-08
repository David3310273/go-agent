package app

import (
	"sync"

	"github.com/David3310273/go-agent/core"
)

var (
	instance *SimpleCommandApp
	once     sync.Once
)

func GetInstance() *SimpleCommandApp {
	once.Do(func() {
		panic("SimpleCommandApp not initialized, call InitInstance first")
	})
	return instance
}

func InitInstance(agent core.AgentCore, cfg core.CommandAppConfig, parser core.CommandParser) *SimpleCommandApp {
	once.Do(func() {
		instance = NewSimpleCommandApp(agent, cfg, parser)
	})
	return instance
}
