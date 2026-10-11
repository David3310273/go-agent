package controller

import (
	"flag"
	"fmt"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

func parseMCPCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	if len(cmdSlices) < 2 {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "MCP subcommand is required, usage: /mcp add|ls|rm [server_name] [--host host] [--port port] [--baseUrl baseUrl] [--protocolVersion version] [--jsonrpcVersion version] [--name name] [--description desc] [--file file_path]",
			},
		}
	}

	subCommand := cmdSlices[1]

	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(nil)

	host := fs.String("host", "", "MCP server host")
	port := fs.Int("port", 0, "MCP server port")
	baseUrl := fs.String("baseUrl", "", "MCP server base URL")
	protocolVersion := fs.String("protocolVersion", "", "MCP protocol version")
	jsonrpcVersion := fs.String("jsonrpcVersion", "", "JSON-RPC version")
	name := fs.String("name", "", "MCP server name")
	description := fs.String("description", "", "MCP server description")
	filePath := fs.String("file", "", "load MCP config from JSON file")
	toolName := fs.String("tool", "", "tool name for listing tools")
	args := fs.String("args", "", "arguments for calling tool")

	err := fs.Parse(cmdSlices[2:])
	if err != nil {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("failed to parse MCP command: %v", err),
			},
		}
	}

	argList := fs.Args()
	var serverName string

	switch subCommand {
	case model.MCPSubCommandAdd:
		if len(argList) < 1 {
			return nil, []core.Diagnostic{
				{
					Level:   core.SeverityError,
					Code:    core.MessageCodeInvalidCommand,
					Message: "server name is required for /mcp add",
				},
			}
		}
		serverName = argList[0]
	case model.MCPSubCommandLs:
		if len(argList) > 0 {
			serverName = argList[0]
		}
	case model.MCPSubCommandRm:
		if len(argList) < 1 {
			return nil, []core.Diagnostic{
				{
					Level:   core.SeverityError,
					Code:    core.MessageCodeInvalidCommand,
					Message: "server name is required for /mcp rm",
				},
			}
		}
		serverName = argList[0]
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("unknown MCP subcommand: %s, expected add|ls|rm", subCommand),
			},
		}
	}

	portValue := 0
	if port != nil {
		portValue = *port
	}

	cmd := &model.MCPCommand{
		SubCommand:      subCommand,
		ServerName:      serverName,
		Host:            *host,
		Port:            portValue,
		BaseUrl:         *baseUrl,
		ProtocolVersion: *protocolVersion,
		JSONRPCVersion:  *jsonrpcVersion,
		Name:            *name,
		Description:     *description,
		FilePath:        *filePath,
		ToolName:        *toolName,
		Args:            *args,
		ResponseChan:    make(chan core.Diagnostic, 1),
	}

	return cmd, nil
}
