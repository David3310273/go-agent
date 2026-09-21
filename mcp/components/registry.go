package components

import (
	"context"
	"log"
	"os"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterToolsToServer registers all tools from config.json to MCP server.
// loads tool names from config, creates tool instances, and registers them to MCP server.
// context is managed by each tool internally via GetContext().
func RegisterToolsToServer(server *mcp.Server, config *MCPConfig) {
	for _, toolName := range config.Tools {
		// general tool, agent context can be nil
		tool := core.GetTool(toolName, config.RootPath, nil)
		if tool == nil {
			log.Printf("[MCP] tool not found: %s", toolName)
			continue
		}

		// get tool schema for MCP registration
		schema := tool.GetSchema()
		mcpToolName := schema.Function.Name
		toolDesc := schema.Function.Description
		isDestructive := tool.IsDestructive()
		// create a copy to avoid pointer issues
		destructiveHint := isDestructive

		// create MCP tool definition
		mcpTool := &mcp.Tool{
			Name:        mcpToolName,
			Description: toolDesc,
			InputSchema: schema.Function.Parameters,
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructiveHint,
			},
		}

		// register tool with adapter
		handler := MCPAdapter(tool)
		mcp.AddTool(server, mcpTool, handler)

		log.Printf("[MCP] registered tool: %s", mcpToolName)
	}
}

// RegisterPromptsToServer registers prompts from config.json to MCP server.
// reads prompt files and registers them as MCP prompts.
// Supports template variables in format {{variableName}} that can be replaced via arguments.
func RegisterPromptsToServer(server *mcp.Server, config *MCPConfig) {
	for _, promptPath := range config.Prompts {
		// resolve full path
		fullPath := path.Join(config.RootPath, promptPath)

		// read prompt file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[MCP] failed to read prompt file %s: %v", fullPath, err)
			continue
		}

		// extract prompt name from file path
		promptName := path.Base(promptPath)
		templateContent := string(content)

		// create MCP prompt
		prompt := &mcp.Prompt{
			Name:        promptName,
			Description: "Prompt from " + promptPath,
		}

		// register prompt handler
		// supports template variable replacement using {{key}} syntax.
		server.AddPrompt(prompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			result := templateContent

			// replace template variables with arguments
			if req.Params != nil && req.Params.Arguments != nil {
				for key, value := range req.Params.Arguments {
					result = strings.ReplaceAll(result, "{{"+key+"}}", value)
				}
			}

			return &mcp.GetPromptResult{
				Description: "Prompt from " + promptPath,
				Messages: []*mcp.PromptMessage{
					{
						Role: mcp.Role("user"),
						Content: &mcp.TextContent{
							Text: result,
						},
					},
				},
			}, nil
		})

		log.Printf("[MCP] registered prompt: %s", promptName)
	}
}

// RegisterResourcesToServer registers resources from config.json to MCP server.
// reads resource files and registers them as MCP resources.
func RegisterResourcesToServer(server *mcp.Server, config *MCPConfig) {
	for _, resourcePath := range config.Resources {
		// resolve full path
		fullPath := path.Join(config.RootPath, resourcePath)

		// read resource file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[MCP] failed to read resource file %s: %v", fullPath, err)
			continue
		}

		// extract resource name from file path
		resourceName := path.Base(resourcePath)

		// create resource URI depend on your storage, mock here
		resourceURI := "file://" + resourcePath

		// create MCP resource
		resource := &mcp.Resource{
			URI:         resourceURI,
			Name:        resourceName,
			Description: "Resource from " + resourcePath,
		}

		// register resource handler
		server.AddResource(resource, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:  resourceURI,
						Text: string(content),
					},
				},
			}, nil
		})

		log.Printf("[MCP] registered resource: %s (%s)", resourceName, resourceURI)
	}
}
