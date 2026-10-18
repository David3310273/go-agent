package model

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*SkillCommand)(nil)

type SkillCommand struct {
	SubCommand   string
	Name         string
	ResponseChan chan core.Diagnostic
}

const (
	SkillCommandName = "/skill"

	SkillSubCommandLs = "ls"
)

func init() {
	RegisterCommand(SkillCommandName, func() core.Command {
		return &SkillCommand{}
	})
}

func (c SkillCommand) GetHelp() map[string]any {
	return map[string]any{
		"format": "/skill ls [--name skill_name]",
		"desc":   "manage skills",
		"options": map[string]string{
			"ls":   "list all available skills, usage: /skill ls [--name skill_name]",
			"name": "filter by skill name (optional)",
		},
	}
}

func (c *SkillCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "name":
		c.Name = value
	default:
		return c
	}
	return c
}

func (c SkillCommand) GetBase() string {
	return SkillCommandName
}

func (c *SkillCommand) Run(app core.CommandApp) *core.Diagnostic {
	switch c.SubCommand {
	case SkillSubCommandLs:
		agent := app.GetAgentCore()
		if agent == nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSystemError,
				Message: "agent not initialized",
			}
		}

		skills := (*agent).GetSkillConfigs()

		if c.Name != "" {
			filtered := []core.SkillDefinition{}
			for _, s := range skills {
				if s.Name == c.Name {
					filtered = append(filtered, s)
					break
				}
			}
			skills = filtered
		}

		return &core.Diagnostic{
			Code:    core.MessageCodeSuccess,
			Message: fmt.Sprintf("list %d skills", len(skills)),
			Data:    skills,
		}
	}

	return nil
}

func (c *SkillCommand) GetOutputChans() []core.OutputChannel {
	return []core.OutputChannel{
		{Chan: c.ResponseChan, Type: "response"},
	}
}

func (c *SkillCommand) ToString() string {
	parts := []string{SkillCommandName, c.SubCommand}
	if c.Name != "" {
		parts = append(parts, "--name", c.Name)
	}
	return strings.Join(parts, " ")
}
