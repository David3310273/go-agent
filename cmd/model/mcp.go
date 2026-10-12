package model

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*MCPCommand)(nil)

type MCPCommand struct {
	SubCommand string
	ServerName string
	FilePath   string
}

const (
	MCPCommandName = "/mcp"

	MCPSubCommandLs  = "ls"
	MCPSubCommandRm  = "rm"
	MCPSubCommandAdd = "add"
)

func init() {
	RegisterCommand(MCPCommandName, func() core.Command {
		return &MCPCommand{}
	})
}

func (c MCPCommand) GetHelp() map[string]any {
	return map[string]any{
		"format": "/mcp add|ls|rm [server_name] [--file file_path]",
		"desc":   "manage MCP server",
		"options": map[string]string{
			"add": "add a MCP server, usage: /mcp add <server_name> <--file file_path>",
			"ls":  "list MCP servers, usage: /mcp ls [server_name]",
			"rm":  "remove a MCP server, usage: /mcp rm <server_name>",
		},
	}
}

func (c *MCPCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "serverName":
		c.ServerName = value
	case "filePath":
		c.FilePath = value
	case "name":
		c.ServerName = value
	default:
		return c
	}
	return c
}

func (c MCPCommand) GetBase() string {
	return MCPCommandName
}

func (c *MCPCommand) Run(app core.CommandApp) *core.Diagnostic {
	switch c.SubCommand {
	case MCPSubCommandAdd:
		return c.handleAdd(app)
	case MCPSubCommandLs:
		return c.handleLs(app)
	case MCPSubCommandRm:
		return c.handleRm(app)
	default:
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidCommand,
			Message: fmt.Sprintf("unknown subcommand %s %s", c.GetBase(), c.SubCommand),
		}
	}
}

func (c *MCPCommand) handleAdd(app core.CommandApp) *core.Diagnostic {
	agent := app.GetAgentCore()
	mcpConfigs := (*agent).GetMCPServerConfigs()

	var newMCPConfigs []core.MCPConfig

	if c.FilePath != "" {
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeDocProcessError,
				Message: fmt.Sprintf("failed to read config file %s: %v", c.FilePath, err),
			}
		}

		if err := json.Unmarshal(data, &newMCPConfigs); err != nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeDocProcessError,
				Message: fmt.Sprintf("failed to parse config file %s: %v", c.FilePath, err),
			}
		}
	}

	mcpConfigMap := make(map[string]core.MCPConfig)
	for _, config := range mcpConfigs {
		mcpConfigMap[config.Name] = config
	}

	// validate and add all configs from file
	for _, config := range newMCPConfigs {
		// apply command-line overrides if provided
		if c.ServerName != "" {
			config.Name = c.ServerName
		}

		if config.Host == "" {
			core.LogStd(core.LogLevelWarn, "MCP server %s host is required", config.Name)
			continue
		}

		if config.Name == "" {
			core.LogStd(core.LogLevelWarn, "MCP server %s name is required", config.Name)
			continue
		}

		if _, ok := mcpConfigMap[config.Name]; ok {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("MCP server %s already exists", config.Name),
			}
		}

		mcpConfigs = append(mcpConfigs, config)
	}

	(*agent).SetMCPClient(mcpConfigs)

	return &core.Diagnostic{
		Level:   core.SeverityInfo,
		Code:    core.MessageCodeSuccess,
		Message: fmt.Sprintf("added %d MCP server(s). Use /new-page to start with the updated MCP servers.", len(newMCPConfigs)),
	}
}

func (c *MCPCommand) handleLs(app core.CommandApp) *core.Diagnostic {
	agent := app.GetAgentCore()
	configs := (*agent).GetMCPServerConfigs()

	if c.ServerName != "" {
		for _, config := range configs {
			if config.Name == c.ServerName {
				return &core.Diagnostic{
					Code:    core.MessageCodeSuccess,
					Message: fmt.Sprintf("MCP server: %s", config.Name),
					Data:    []core.MCPConfig{config},
				}
			}
		}
		return &core.Diagnostic{
			Code:    core.MessageCodeInvalidInput,
			Message: fmt.Sprintf("MCP server %s not found", c.ServerName),
			Data:    nil,
		}
	}

	return &core.Diagnostic{
		Code:    core.MessageCodeSuccess,
		Message: fmt.Sprintf("list %d MCP servers", len(configs)),
		Data:    configs,
	}
}

func (c *MCPCommand) handleRm(app core.CommandApp) *core.Diagnostic {
	agent := app.GetAgentCore()

	mcpConfigs := (*agent).GetMCPServerConfigs()
	newConfigs := []core.MCPConfig{}

	for _, cfg := range mcpConfigs {
		if cfg.Name != c.ServerName {
			newConfigs = append(newConfigs, cfg)
		}
	}

	(*agent).SetMCPClient(newConfigs)

	return &core.Diagnostic{
		Level:   core.SeverityInfo,
		Code:    core.MessageCodeSuccess,
		Message: fmt.Sprintf("removed MCP server %s", c.ServerName),
	}
}

func (c *MCPCommand) GetOutputChans() []core.OutputChannel {
	return nil
}
