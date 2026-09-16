package main

import (
	"log"

	mcp "github.com/David3310273/go-agent/mcp/components"
	mcpServer "github.com/David3310273/go-agent/mcp/server"
	_ "github.com/David3310273/go-agent/tools" // import to register tools
)

func main() {
	// load MCP config from config.json
	// reads tools and port configuration from config.json.
	mcpConfig, err := mcp.LoadConfig()
	if err != nil {
		log.Fatalf("[MCP Server] failed to load config: %v", err)
	}

	log.Printf("[MCP Server config] root path: %s, port: %s, tools: %v", mcpConfig.RootPath, mcpConfig.Port, mcpConfig.Tools)

	mcpServer.Start(mcpConfig)
}
