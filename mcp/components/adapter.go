package components

import (
	"context"
	"fmt"

	"github.com/David3310273/go-agent/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPAdapter adapts core.Tool to MCP handler function.
// returns ToolHandlerFor[map[string]any, any] for MCP registration.
func MCPAdapter(tool core.Tool) func(context.Context, *mcp.CallToolRequest, map[string]any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		result, diag := core.CallTool(tool, args)
		if diag != nil && diag.Level == core.SeverityError {
			return nil, diag, fmt.Errorf("%s", diag.Message)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: result,
				},
			},
		}, nil, nil
	}
}
