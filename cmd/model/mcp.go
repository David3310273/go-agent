package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*MCPCommand)(nil)

type MCPCommand struct {
	SubCommand      string
	ServerName      string
	FilePath        string
	Host            string
	Port            int
	BaseUrl         string
	ProtocolVersion string
	JSONRPCVersion  string
	Name            string
	Description     string
	ToolName        string
	Args            string
	ResponseChan    chan core.Diagnostic
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
		"format": "/mcp add|ls|rm [server_name] [--host host] [--port port] [--baseUrl baseUrl] [--protocolVersion version] [--jsonrpcVersion version] [--name name] [--description desc] [--file file_path] [--tool tool_name] [--args args]",
		"desc":   "manage MCP server",
		"options": map[string]string{
			"add": "add a MCP server, usage: /mcp add <server_name> <--host host> <--port port> <--name name> <--description desc> [--file file_path] [--baseUrl baseUrl] [--protocolVersion version] [--jsonrpcVersion version]",
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
	case "host":
		c.Host = value
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			core.LogStd(core.LogLevelWarn, "invalid port %v", value)
			return c
		}
		c.Port = port
	case "baseUrl":
		c.BaseUrl = value
	case "protocolVersion":
		c.ProtocolVersion = value
	case "jsonrpcVersion":
		c.JSONRPCVersion = value
	case "name":
		c.Name = value
	case "description":
		c.Description = value
	case "toolName":
		c.ToolName = value
	case "args":
		c.Args = value
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
	configs := (*agent).GetMCPServerConfigs()

	var config core.MCPConfig

	if c.FilePath != "" {
		data, err := os.ReadFile(c.FilePath)
		if err != nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeDocProcessError,
				Message: fmt.Sprintf("failed to read config file %s: %v", c.FilePath, err),
			}
		}

		if err := json.Unmarshal(data, &config); err != nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeDocProcessError,
				Message: fmt.Sprintf("failed to parse config file %s: %v", c.FilePath, err),
			}
		}
	}

	if c.Host != "" {
		config.Host = c.Host
	}
	if c.Port != 0 {
		config.Port = c.Port
	}
	if c.BaseUrl != "" {
		config.BaseUrl = c.BaseUrl
	}
	if c.ProtocolVersion != "" {
		config.ProtocolVersion = c.ProtocolVersion
	}
	if c.JSONRPCVersion != "" {
		config.JSONRPCVersion = c.JSONRPCVersion
	}
	if c.Name != "" {
		config.Name = c.Name
	}
	if c.Description != "" {
		config.Description = c.Description
	}

	if config.Host == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidInput,
			Message: "MCP server host is required",
		}
	}
	if config.Port == 0 {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidInput,
			Message: "MCP server port is required",
		}
	}
	if config.Name == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidInput,
			Message: "MCP server name is required",
		}
	}

	if config.ProtocolVersion == "" {
		config.ProtocolVersion = "2026-07-28"
	}
	if config.JSONRPCVersion == "" {
		config.JSONRPCVersion = "2.0"
	}

	for _, cfg := range configs {
		if cfg.Name == config.Name {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("MCP server %s already exists", config.Name),
			}
		}
	}

	newConfigs := append(configs, config)
	diag := (*agent).SetMCPClient(newConfigs)
	if diag != nil {
		return diag
	}

	return &core.Diagnostic{
		Level:   core.SeverityInfo,
		Code:    core.MessageCodeSuccess,
		Message: fmt.Sprintf("MCP server %s added", config.Name),
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
	clients := (*agent).GetMCPClients()

	client, exists := clients[c.ServerName]
	if !exists {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidInput,
			Message: fmt.Sprintf("MCP server %s not found", c.ServerName),
		}
	}

	config := client.GetConfig()
	configs := (*agent).GetMCPServerConfigs()

	newConfigs := []core.MCPConfig{}
	for _, cfg := range configs {
		if cfg.Name != c.ServerName {
			newConfigs = append(newConfigs, cfg)
		}
	}

	diag := (*agent).SetMCPClient(newConfigs)
	if diag != nil {
		return diag
	}

	return &core.Diagnostic{
		Level:   core.SeverityInfo,
		Code:    core.MessageCodeSuccess,
		Message: fmt.Sprintf("removed MCP server %s", config.Name),
	}
}

func (c *MCPCommand) GetOutputChans() []core.OutputChannel {
	return nil
}
