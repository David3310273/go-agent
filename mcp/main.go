package main

import (
	"os"

	"github.com/David3310273/go-agent/core"
	mcp "github.com/David3310273/go-agent/mcp/components"
	mcpServer "github.com/David3310273/go-agent/mcp/server"
	_ "github.com/David3310273/go-agent/tools" // import to register tools
)

func main() {
	// load MCP config from config.json
	// reads tools and port configuration from config.json.
	mcpConfig, err := mcp.LoadConfig()
	if err != nil {
		core.LogStd(core.LogLevelError, "mcp server failed to load config: %v", err)
		os.Exit(1)
	}

	core.LogStd(core.LogLevelInfo, "mcp server config: root_path=%s, port=%s, tools=%v", mcpConfig.RootPath, mcpConfig.Port, mcpConfig.Tools)

	mcpServer.Start(mcpConfig)
}
