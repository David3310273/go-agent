package server

import (
	"net/http"
	"os"

	"github.com/David3310273/go-agent/core"
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
	core.LogStd(core.LogLevelInfo, "mcp server starting on %s", addr)
	core.LogStd(core.LogLevelInfo, "mcp server endpoint: http://%s/mcp", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		core.LogStd(core.LogLevelError, "mcp server failed: %v", err)
		os.Exit(1)
	}
}
