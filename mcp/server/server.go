package server

import (
	"log"
	"net/http"

	mcp "github.com/David3310273/go-agent/mcp/components"
	_ "github.com/David3310273/go-agent/tools" // import to register tools
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func Start(config *mcp.MCPConfig) {
	// create MCP server
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "go-agent-mcp-server",
		Version: "1.0.0",
	}, nil)

	// register tools from config.json
	// context is now managed by each tool internally via GetContext().
	mcp.RegisterToolsToServer(server, config)
	mcp.RegisterPromptsToServer(server, config)
	mcp.RegisterResourcesToServer(server, config)

	// create HTTP handler for MCP
	handler := mcpsdk.NewStreamableHTTPHandler(
		func(req *http.Request) *mcpsdk.Server {
			return server
		},
		&mcpsdk.StreamableHTTPOptions{
			Stateless: true,
		},
	)

	// setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	addr := ":" + config.Port
	log.Printf("[MCP Server] starting on %s", addr)
	log.Printf("[MCP Server] endpoint: http://%s/mcp", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[MCP Server] failed: %v", err)
	}
}
