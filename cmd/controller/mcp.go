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

	name := fs.String("name", "", "MCP server name")
	filePath := fs.String("file", "", "load MCP config from JSON file")

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

	switch subCommand {
	case model.MCPSubCommandAdd:
		isFilePath := false
		for _, arg := range argList {
			if arg == "--file" {
				isFilePath = true
			} else if isFilePath {
				*filePath = arg
				isFilePath = false
			} else {
				if *name == "" {
					*name = arg
				}
			}
		}
	case model.MCPSubCommandLs:
		if len(argList) > 0 {
			*name = argList[0]
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
		if *name == "" {
			*name = argList[0]
		}
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("unknown MCP subcommand: %s, expected add|ls|rm", subCommand),
			},
		}
	}

	cmd := &model.MCPCommand{
		SubCommand: subCommand,
		ServerName: *name,
		FilePath:   *filePath,
	}

	return cmd, nil
}
