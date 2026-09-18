package simple

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
)

// MCPToolWrapper represents a tool provided by MCP server
type MCPToolWrapper struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"params,omitempty"`
	context     core.Context   // auto-add: agent session runtime context
}

// MCPRemoteUtil provides utilities for interacting with MCP server
type MCPRemoteUtil struct {
	Config *core.MCPConfig
	tools  map[string]core.Tool // auto-add: cached MCPToolWrapper instances, name -> tool
}

// auto-add: interface assertions
var _ core.Tool = (*MCPToolWrapper)(nil)
var _ core.MCPAccessible = (*MCPRemoteUtil)(nil)

// auto-add: MCPToolWrapper implements core.Tool interface

// GetSchema returns tool schema in provider-agnostic format
func (m *MCPToolWrapper) GetSchema() core.ToolSchema {
	return core.ToolSchema{
		Type: "function",
		Function: core.FunctionSchema{
			Name:        m.Name,
			Description: m.Description,
			Parameters:  m.Schema,
		},
	}
}

// GetName returns tool name
func (m *MCPToolWrapper) GetName() string {
	return m.Name
}

// GetDescription returns tool description
func (m *MCPToolWrapper) GetDescription() string {
	return m.Description
}

// Validate validates tool arguments
// auto-add: validates serverName and toolName are present
func (m *MCPToolWrapper) Validate(args map[string]any) *core.Diagnostic {
	if serverName, ok := args["serverName"].(string); !ok || serverName == "" {
		return &core.Diagnostic{
			Code:    core.MessageCodeToolValidateError,
			Level:   core.SeverityError,
			Message: "serverName is required",
		}
	}
	if toolName, ok := args["toolName"].(string); !ok || toolName == "" {
		return &core.Diagnostic{
			Code:    core.MessageCodeToolValidateError,
			Level:   core.SeverityError,
			Message: "toolName is required",
		}
	}
	return nil
}

// GetRunner returns the runner function that executes the tool.
// auto-add: parses serverName and toolName from args, gets client from context
func (m *MCPToolWrapper) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		// auto-add: parse serverName and toolName from args
		serverName, _ := args["serverName"].(string)
		toolName, _ := args["toolName"].(string)

		// auto-add: get MCP client from context by server name
		mcpClients := m.context.GetMCPClients()
		client, exists := mcpClients[serverName]
		if !exists {
			return "", &core.Diagnostic{
				Code:    core.MessageCodeToolRunError,
				Level:   core.SeverityError,
				Message: fmt.Sprintf("MCP server not found: %s", serverName),
			}
		}

		// call the tool via MCP using the client reference
		_, result, err := client.CallTool(toolName, args)
		if err != nil {
			log.Printf("tool %s call failed: %v", m.Name, err)
			return "", &core.Diagnostic{
				Code:    core.MessageCodeToolRunError,
				Level:   core.SeverityError,
				Message: fmt.Sprintf("tool %s call failed", m.Name),
				Data:    err.Error(),
			}
		}

		// parse content from result
		// MCP tool result typically has "content" field
		if resultMap, ok := result.(map[string]any); ok {
			if content, exists := resultMap["content"]; exists {
				// content can be a string or an array
				switch c := content.(type) {
				case string:
					return c, nil
				case []any:
					// extract text from content array
					var texts []string
					for _, item := range c {
						if itemMap, ok := item.(map[string]any); ok {
							if text, exists := itemMap["text"]; exists {
								if textStr, ok := text.(string); ok {
									texts = append(texts, textStr)
								}
							}
						}
					}
					return strings.Join(texts, "\n"), nil
				}
			}
		}

		// fallback: return empty string
		return "", nil
	}
}

// GetContext returns agent session runtime context
func (m *MCPToolWrapper) GetContext() core.Context {
	return m.context
}

// auto-add: MCPRemoteUtil implements core.MCPAccessible interface

// GetConfig returns MCP configuration
func (m *MCPRemoteUtil) GetConfig() *core.MCPConfig {
	return m.Config
}

// SetConfig sets MCP configuration
func (m *MCPRemoteUtil) SetConfig(config *core.MCPConfig) {
	m.Config = config
}

// BuildTools fetches tools from MCP server and builds Tool instances.
// auto-add: combines list and build, returns cached MCPToolWrapper instances
func (m *MCPRemoteUtil) BuildTools(context core.Context) map[string]core.Tool {
	// auto-add: use sendRequest to get SSE support
	resp, diag := m.sendRequest(core.MCPMethodToolsList, nil)
	if diag != nil {
		log.Printf("%s failed: %v", core.MCPMethodToolsList, diag.Message)
		return nil
	}

	// auto-add: directly parse Result into MCPListToolsResponse
	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult core.MCPListToolsResponse
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		log.Printf("failed to parse tools result: %v", err)
		return nil
	}

	// build MCPToolWrapper instances
	toolsMap := make(map[string]core.Tool)
	for _, toolInfo := range toolsResult.Tools {
		mcpTool := &MCPToolWrapper{
			Name:        toolInfo.Name,
			Description: toolInfo.Description,
			Schema:      toolInfo.InputSchema,
			context:     context,
		}
		toolsMap[toolInfo.Name] = mcpTool
	}

	// cache the built tools
	m.tools = toolsMap
	log.Printf("built and cached %d tools from MCP server", len(toolsMap))
	return toolsMap
}

// ListPrompts lists available prompts from MCP server.
// auto-add: sends prompts/list request to MCP server
// auto-add: now returns MCPServerResponse
func (m *MCPRemoteUtil) ListPrompts() (core.MCPServerResponse, *core.Diagnostic) {
	resp, diag := m.sendRequest(core.MCPMethodPromptsList, nil)
	if diag != nil {
		return core.MCPServerResponse{}, diag
	}
	return *resp, nil
}

// GetPrompt gets a specific prompt from MCP server.
// auto-add: sends prompts/get request to MCP server
// auto-add: parses response and extracts prompt content
func (m *MCPRemoteUtil) GetPrompt(name string) (string, *core.Diagnostic) {
	params := map[string]any{
		"name": name,
	}
	resp, diag := m.sendRequest(core.MCPMethodPromptsGet, params)
	if diag != nil {
		return "", diag
	}

	// auto-add: parse the response to extract prompt content
	if resp.Result != nil {
		resultJSON, err := json.Marshal(resp.Result)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSystemError,
				Message: "failed to marshal prompt result",
				Data:    err.Error(),
			}
		}

		var promptResult core.MCPGetPromptResult
		if err := json.Unmarshal(resultJSON, &promptResult); err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSystemError,
				Message: "failed to unmarshal prompt result",
				Data:    err.Error(),
			}
		}

		// auto-add: extract text from messages
		var texts []string
		for _, msg := range promptResult.Messages {
			if msg.Content.Type == "text" && msg.Content.Text != "" {
				texts = append(texts, msg.Content.Text)
			}
		}

		return strings.Join(texts, "\n"), nil
	}

	return "", nil
}

// ListResources lists available resources from MCP server.
// auto-add: sends resources/list request to MCP server
// auto-add: now returns MCPServerResponse
func (m *MCPRemoteUtil) ListResources() (core.MCPServerResponse, *core.Diagnostic) {
	resp, diag := m.sendRequest(core.MCPMethodResourcesList, nil)
	if diag != nil {
		return core.MCPServerResponse{}, diag
	}
	return *resp, nil
}

// GetResource gets a specific resource from MCP server.
// auto-add: sends resources/read request to MCP server
// auto-add: parses response and extracts resource content
func (m *MCPRemoteUtil) GetResource(name string) (string, *core.Diagnostic) {
	params := map[string]any{
		"uri": name,
	}
	resp, diag := m.sendRequest(core.MCPMethodResourcesRead, params)
	if diag != nil {
		return "", diag
	}

	// auto-add: parse the response to extract resource content
	if resp.Result != nil {
		resultJSON, err := json.Marshal(resp.Result)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSystemError,
				Message: "failed to marshal resource result",
				Data:    err.Error(),
			}
		}

		var resourceResult core.MCPGetResourceResult
		if err := json.Unmarshal(resultJSON, &resourceResult); err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSystemError,
				Message: "failed to unmarshal resource result",
				Data:    err.Error(),
			}
		}

		// auto-add: extract text from contents
		var texts []string
		for _, content := range resourceResult.Contents {
			if content.Text != "" {
				texts = append(texts, content.Text)
			}
		}

		return strings.Join(texts, "\n"), nil
	}

	return "", nil
}

// getURL returns the MCP server URL
func (m *MCPRemoteUtil) getURL() string {
	return fmt.Sprintf("%s:%d/%s", m.Config.Host, m.Config.Port, m.Config.BaseUrl)
}

// createHeaders returns common headers for MCP requests
func (m *MCPRemoteUtil) createHeaders() map[string]string {
	return map[string]string{
		"Content-Type":           "application/json",
		"Accept":                 "text/event-stream, application/json",
		"X-MCP-Protocol-Version": m.Config.ProtocolVersion,
		"X-MCP-Client-Name":      m.Config.ClientName,
		"X-MCP-Client-Version":   m.Config.ClientVersion,
	}
}

// auto-add: parseSSEResponse parses SSE format and extracts JSON data
// SSE format: "event: message\ndata: {...}\n\n"
// auto-add: supports multiple data lines, concatenates them
func parseSSEResponse(body []byte) []byte {
	lines := strings.Split(string(body), "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			dataLines = append(dataLines, data)
		}
	}
	if len(dataLines) == 0 {
		return nil
	}
	// auto-add: if multiple data lines, concatenate them (for streaming responses)
	return []byte(strings.Join(dataLines, ""))
}

// sendRequest sends a request to MCP server with the given method and params.
// auto-add: uses utils.SendRequest, generic helper for all MCP requests
// auto-add: now returns MCPServerResponse to parse the result
func (m *MCPRemoteUtil) sendRequest(jsonrpcMethod string, params map[string]any) (*core.MCPServerResponse, *core.Diagnostic) {
	// build JSON-RPC request body
	body := map[string]any{
		"jsonrpc": m.Config.JSONRPCVersion,
		"method":  jsonrpcMethod,
		"id":      1,
	}
	if params != nil {
		body["params"] = params
	}

	// build request
	request := utils.Request[map[string]any]{
		Body:    body,
		Headers: m.createHeaders(),
		Method:  "POST",
	}

	// auto-add: log request body and headers before sending
	bodyJSON, _ := json.Marshal(body)
	log.Printf("[MCP] %s request body: %s", jsonrpcMethod, string(bodyJSON))
	log.Printf("[MCP] %s request headers: %v", jsonrpcMethod, request.Headers)

	// send request
	config := utils.HTTPConfig{URL: m.getURL()}
	resp, err := utils.SendRequest(config, request)
	if err != nil {
		log.Printf("%s failed: %v", jsonrpcMethod, err)
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: fmt.Sprintf("%s failed", jsonrpcMethod),
			Data:    err.Error(),
		}
	}
	defer resp.Body.Close()

	// auto-add: read response body for debugging
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("%s: failed to read response body: %v", jsonrpcMethod, err)
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to read response body",
			Data:    err.Error(),
		}
	}
	log.Printf("[MCP] %s response status: %d", jsonrpcMethod, resp.StatusCode)
	log.Printf("[MCP] %s response content-type: %s", jsonrpcMethod, resp.Header.Get("Content-Type"))
	log.Printf("[MCP] %s response body: %s", jsonrpcMethod, string(respBody))

	// auto-add: parse response based on content-type
	contentType := resp.Header.Get("Content-Type")
	var jsonBody []byte

	if strings.Contains(contentType, "text/event-stream") {
		// auto-add: parse SSE format
		jsonBody = parseSSEResponse(respBody)
		if jsonBody == nil {
			return nil, &core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: "failed to parse SSE response",
				Data:    "no data field found in SSE response",
			}
		}
	} else {
		// auto-add: direct JSON response
		jsonBody = respBody
	}

	// auto-add: parse response body
	var serverResp core.MCPServerResponse
	if err := json.Unmarshal(jsonBody, &serverResp); err != nil {
		log.Printf("%s: failed to decode response: %v", jsonrpcMethod, err)
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to decode response",
			Data:    err.Error(),
		}
	}

	log.Printf("%s completed successfully", jsonrpcMethod)
	return &serverResp, nil
}

// CallTool calls a tool on the MCP server with the given name and arguments.
// auto-add: uses sendRequest to get SSE support
func (m *MCPRemoteUtil) CallTool(name string, arguments map[string]any) (string, any, error) {
	// auto-add: build params for tools/call
	params := map[string]any{
		"name":      name,
		"arguments": arguments,
	}

	// auto-add: use sendRequest
	resp, diag := m.sendRequest(core.MCPMethodToolsCall, params)
	if diag != nil {
		return "", nil, fmt.Errorf("failed to call tool %s: %s", name, diag.Message)
	}

	// check for error
	if resp.Error != nil {
		return "", nil, fmt.Errorf("tool call failed: %s", resp.Error.Message)
	}

	// auto-add: parse result using MCPCallToolResult
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal tool result: %w", err)
	}

	var callResult core.MCPCallToolResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}

	// auto-add: extract text from content array
	var texts []string
	for _, content := range callResult.Content {
		if content.Text != "" {
			texts = append(texts, content.Text)
		}
	}

	return strings.Join(texts, "\n"), resp.Result, nil
}
