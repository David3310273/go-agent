package core

// auto-add: MCP protocol method constants
const (
	// tools
	MCPMethodToolsList = "tools/list"
	MCPMethodToolsCall = "tools/call"

	// prompts
	MCPMethodPromptsList = "prompts/list"
	MCPMethodPromptsGet  = "prompts/get"

	// resources
	MCPMethodResourcesList          = "resources/list"
	MCPMethodResourcesRead          = "resources/read"
	MCPMethodResourcesTemplatesList = "resources/templates/list"
)

// auto-add: MCPServerResponse is a generic JSON-RPC 2.0 response structure for MCP server
type MCPServerResponse struct {
	JSONRPC string    `json:"jsonrpc"`          // JSON-RPC version, should be "2.0"
	ID      int       `json:"id"`               // Request ID
	Result  any       `json:"result,omitempty"` // Response result, can be any type
	Error   *MCPError `json:"error,omitempty"`  // Error object, if request failed
}

// auto-add: MCPError represents a JSON-RPC 2.0 error object
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// auto-add: MCPToolWrapperInfo represents a tool from MCP server
type MCPListToolResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// auto-add: MCPListDocInfo represents a prompt or resource from MCP server
type MCPListDocResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Uri         string `json:"uri"`
}

// auto-add: MCPToolsListResult represents the result of tools/list
type MCPListToolsResponse struct {
	TTLMs      int                 `json:"ttlMs,omitempty"`
	CacheScope string              `json:"cacheScope,omitempty"`
	Tools      []MCPListToolResult `json:"tools"`
}

type MCPListResourcesResponse struct {
	TTLMs      int                `json:"ttlMs,omitempty"`
	CacheScope string             `json:"cacheScope,omitempty"`
	Resources  []MCPListDocResult `json:"resources"`
}

type MCPListPromptsResponse struct {
	TTLMs      int                `json:"ttlMs,omitempty"`
	CacheScope string             `json:"cacheScope,omitempty"`
	Prompts    []MCPListDocResult `json:"prompts"`
}

// auto-add: MCPGetPromptResult represents the result of prompts/get
type MCPGetPromptResult struct {
	Description string `json:"description"`
	Messages    []struct {
		Role    string `json:"role"`
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
}

type MCPGetResourceResult struct {
	Contents []struct {
		Uri  string `json:"uri"`
		Text string `json:"text"`
	} `json:"contents"`
}

type MCPCallToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type MCPAccessible interface {
	GetConfig() *MCPConfig
	SetConfig(config *MCPConfig)
	BuildTools(context Context) map[string]Tool
	CallTool(name string, args map[string]any) (string, any, error)
	ListPrompts() (MCPServerResponse, *Diagnostic)
	GetPrompt(name string) (string, *Diagnostic)
	ListResources() (MCPServerResponse, *Diagnostic)
	GetResource(name string) (string, *Diagnostic)
}
