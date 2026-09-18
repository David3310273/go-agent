package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

// auto-add: MCPResource represents a single resource from MCP server
type MCPResource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
}

// auto-add: MCPListResourcesResult represents the result of resources/list
type MCPListResourcesResult struct {
	TTLMs      int           `json:"ttlMs,omitempty"`
	CacheScope string        `json:"cacheScope,omitempty"`
	Resources  []MCPResource `json:"resources"`
}

func init() {
	// register UseMCPServerTools tool factory
	core.RegisterTool("UseMCPServerTools", func(rootPath string, context core.Context) core.Tool {
		return UseMCPServerToolsCall{
			Name:     "UseMCPServerTools",
			Schema:   "usemcp_tools.schema.json",
			RootPath: rootPath,
			Context:  context,
		}
	})

	// register SearchMCPResources tool factory
	core.RegisterTool("UseMCPResources", func(rootPath string, context core.Context) core.Tool {
		return SearchMCPResourcesCall{
			Name:     "SearchMCPResources",
			Schema:   "usemcp_resources.schema.json",
			RootPath: rootPath,
			Context:  context,
		}
	})

	// register SearchMCPPrompts tool factory
	core.RegisterTool("UseMCPPrompts", func(rootPath string, context core.Context) core.Tool {
		return SearchMCPPromptsCall{
			Name:     "SearchMCPPrompts",
			Schema:   "usemcp_prompts.schema.json",
			RootPath: rootPath,
			Context:  context,
		}
	})

	// auto-add: register DiscoverMCPServer tool factory
	core.RegisterTool("DiscoverMCPServer", func(rootPath string, context core.Context) core.Tool {
		return DiscoverMCPServerCall{
			Name:     "DiscoverMCPServer",
			Schema:   "discovermcp.schema.json",
			RootPath: rootPath,
			Context:  context,
		}
	})
}

// =============================================================================
// UseMCPServerTools
// =============================================================================

// auto-add: interface assertion
var _ core.Tool = (*UseMCPServerToolsCall)(nil)

// UseMCPServerToolsCall implements core.Tool interface for executing MCP server tools.
type UseMCPServerToolsCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	Context  core.Context
}

func (f UseMCPServerToolsCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f UseMCPServerToolsCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}
	return schema
}

func (f UseMCPServerToolsCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

func (f UseMCPServerToolsCall) GetContext() core.Context {
	return f.Context
}

func (f UseMCPServerToolsCall) Validate(args map[string]any) *core.Diagnostic {
	// auto-add: validate serverName and toolName are present
	if serverName, ok := args["serverName"].(string); !ok || serverName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "serverName is required",
		}
	}
	if toolName, ok := args["toolName"].(string); !ok || toolName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "toolName is required",
		}
	}
	return nil
}

// MCPToolCallResponse represents the response from executing an MCP tool
type MCPToolCallResponse struct {
	ServerName string `json:"serverName"`
	ToolName   string `json:"toolName"`
	Result     string `json:"result"`
}

func (f UseMCPServerToolsCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		serverName, _ := args["serverName"].(string)
		toolName, _ := args["toolName"].(string)
		arguments, _ := args["arguments"].(map[string]any)

		// get MCP clients from context
		mcpClients := f.Context.GetMCPClients()
		client, exists := mcpClients[serverName]
		if !exists {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("MCP server not found: %s", serverName),
			}
		}

		// auto-add: call the specific tool on the MCP server
		resultStr, _, err := client.CallTool(toolName, arguments)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("tool %s call failed: %v", toolName, err),
			}
		}

		// build response
		response := MCPToolCallResponse{
			ServerName: serverName,
			ToolName:   toolName,
			Result:     resultStr,
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}

// =============================================================================
// SearchMCPResources
// =============================================================================

// auto-add: interface assertion
var _ core.Tool = (*SearchMCPResourcesCall)(nil)

// SearchMCPResourcesCall implements core.Tool interface for searching MCP resources.
type SearchMCPResourcesCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	Context  core.Context
}

func (f SearchMCPResourcesCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f SearchMCPResourcesCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}
	return schema
}

func (f SearchMCPResourcesCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

func (f SearchMCPResourcesCall) GetContext() core.Context {
	return f.Context
}

func (f SearchMCPResourcesCall) Validate(args map[string]any) *core.Diagnostic {
	// auto-add: validate serverName and uri are present
	if serverName, ok := args["serverName"].(string); !ok || serverName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "serverName is required",
		}
	}
	if uri, ok := args["uri"].(string); !ok || uri == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "uri is required",
		}
	}
	return nil
}

// MCPResourceReadResponse represents the response from reading an MCP resource
type MCPResourceReadResponse struct {
	ServerName string `json:"serverName"`
	URI        string `json:"uri"`
	Content    string `json:"content"`
}

func (f SearchMCPResourcesCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		serverName, _ := args["serverName"].(string)
		uri, _ := args["uri"].(string)

		// get MCP clients from context
		mcpClients := f.Context.GetMCPClients()
		client, exists := mcpClients[serverName]
		if !exists {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("MCP server not found: %s", serverName),
			}
		}

		// auto-add: read the specific resource from MCP server
		content, diag := client.GetResource(uri)
		if diag != nil && diag.Level == core.SeverityError {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("resource %s read failed: %v", uri, diag.Message),
			}
		}

		// build response
		response := MCPResourceReadResponse{
			ServerName: serverName,
			URI:        uri,
			Content:    content,
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}

// =============================================================================
// SearchMCPPrompts
// =============================================================================

// auto-add: interface assertion
var _ core.Tool = (*SearchMCPPromptsCall)(nil)

// SearchMCPPromptsCall implements core.Tool interface for searching MCP prompts.
type SearchMCPPromptsCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	Context  core.Context
}

func (f SearchMCPPromptsCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f SearchMCPPromptsCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}
	return schema
}

func (f SearchMCPPromptsCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

func (f SearchMCPPromptsCall) GetContext() core.Context {
	return f.Context
}

func (f SearchMCPPromptsCall) Validate(args map[string]any) *core.Diagnostic {
	// auto-add: validate serverName and promptName are present
	if serverName, ok := args["serverName"].(string); !ok || serverName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "serverName is required",
		}
	}
	if promptName, ok := args["promptName"].(string); !ok || promptName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "promptName is required",
		}
	}
	return nil
}

// MCPPromptGetResponse represents the response from getting an MCP prompt
type MCPPromptGetResponse struct {
	ServerName string `json:"serverName"`
	PromptName string `json:"promptName"`
	Content    string `json:"content"`
}

func (f SearchMCPPromptsCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		serverName, _ := args["serverName"].(string)
		promptName, _ := args["promptName"].(string)

		// get MCP clients from context
		mcpClients := f.Context.GetMCPClients()
		client, exists := mcpClients[serverName]
		if !exists {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("MCP server not found: %s", serverName),
			}
		}

		// auto-add: get the specific prompt from MCP server
		content, diag := client.GetPrompt(promptName)
		if diag != nil && diag.Level == core.SeverityError {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: fmt.Sprintf("prompt %s get failed: %v", promptName, diag.Message),
			}
		}

		// build response
		response := MCPPromptGetResponse{
			ServerName: serverName,
			PromptName: promptName,
			Content:    content,
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}

// =============================================================================
// DiscoverMCPServer
// =============================================================================

// auto-add: interface assertion
var _ core.Tool = (*DiscoverMCPServerCall)(nil)

// DiscoverMCPServerCall implements core.Tool interface for discovering MCP server capabilities.
type DiscoverMCPServerCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	Context  core.Context
}

func (f DiscoverMCPServerCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f DiscoverMCPServerCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}
	return schema
}

func (f DiscoverMCPServerCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

func (f DiscoverMCPServerCall) GetContext() core.Context {
	return f.Context
}

func (f DiscoverMCPServerCall) Validate(args map[string]any) *core.Diagnostic {
	serverName, _ := args["serverName"].(string)
	if serverName == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolValidateError,
			Message: "serverName is required",
		}
	}
	return nil
}

// MCPServerDefinition represents the full definition of an MCP server
type MCPServerDefinition struct {
	ServerName string                   `json:"serverName"`
	Tools      []core.MCPListToolResult `json:"tools"`
	Prompts    []core.MCPListDocResult  `json:"prompts"`
	Resources  []core.MCPListDocResult  `json:"resources"`
}

func (f DiscoverMCPServerCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		serverName, _ := args["serverName"].(string)

		// get MCP clients from context
		mcpClients := f.Context.GetMCPClients()
		if len(mcpClients) == 0 {
			return "Failed", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "no MCP server available",
			}
		}

		// find the specified server
		client, exists := mcpClients[serverName]
		if !exists {
			return "Failed", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "MCP server not found: " + serverName,
			}
		}

		// auto-add: build MCPServerDefinition
		definition := MCPServerDefinition{
			ServerName: serverName,
			Tools:      []core.MCPListToolResult{},
			Prompts:    []core.MCPListDocResult{},
			Resources:  []core.MCPListDocResult{},
		}

		// 1. Get tools
		log.Printf("[DiscoverMCPServer] Building tools for server: %s", serverName)
		tools := client.BuildTools(f.Context)
		log.Printf("[DiscoverMCPServer] Built %d tools", len(tools))
		for name, tool := range tools {
			log.Printf("[DiscoverMCPServer] Adding tool: %s, description: %s, input schema: %v", name, tool.GetDescription(), tool.GetSchema().Function.Parameters)
			definition.Tools = append(definition.Tools, core.MCPListToolResult{
				Name:        name,
				Description: tool.GetDescription(),
				InputSchema: tool.GetSchema().Function.Parameters,
			})
		}

		// 2. List prompts
		log.Printf("[DiscoverMCPServer] Listing prompts for server: %s", serverName)
		promptResp, promptDiag := client.ListPrompts()
		if promptDiag != nil {
			log.Printf("[DiscoverMCPServer] ListPrompts diagnostic: %+v", promptDiag)
		}
		if promptDiag == nil || promptDiag.Level != core.SeverityError {
			if promptResp.Result != nil {
				resultJSON, _ := json.Marshal(promptResp.Result)
				log.Printf("[DiscoverMCPServer] Prompts result JSON: %s", string(resultJSON))
				var promptsResult core.MCPListPromptsResponse
				if err := json.Unmarshal(resultJSON, &promptsResult); err != nil {
					log.Printf("[DiscoverMCPServer] Failed to unmarshal prompts: %v", err)
				} else {
					log.Printf("[DiscoverMCPServer] Parsed %d prompts", len(promptsResult.Prompts))
					definition.Prompts = promptsResult.Prompts
				}
			} else {
				log.Printf("[DiscoverMCPServer] promptResp.Result is nil")
			}
		}

		// 3. List resources
		log.Printf("[DiscoverMCPServer] Listing resources for server: %s", serverName)
		resourceResp, resourceDiag := client.ListResources()
		if resourceDiag != nil {
			log.Printf("[DiscoverMCPServer] ListResources diagnostic: %+v", resourceDiag)
		}
		if resourceDiag == nil || resourceDiag.Level != core.SeverityError {
			if resourceResp.Result != nil {
				resultJSON, _ := json.Marshal(resourceResp.Result)
				log.Printf("[DiscoverMCPServer] Resources result JSON: %s", string(resultJSON))
				var resourcesResult core.MCPListResourcesResponse
				if err := json.Unmarshal(resultJSON, &resourcesResult); err != nil {
					log.Printf("[DiscoverMCPServer] Failed to unmarshal resources: %v", err)
				} else {
					log.Printf("[DiscoverMCPServer] Parsed %d resources", len(resourcesResult.Resources))
					definition.Resources = append(definition.Resources, resourcesResult.Resources...)
				}
			} else {
				log.Printf("[DiscoverMCPServer] resourceResp.Result is nil")
			}
		}

		// serialize to JSON
		jsonResult, err := json.Marshal(definition)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		log.Printf("\n\n[MCP server]: definition of %s: %s\n", serverName, string(jsonResult))

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
