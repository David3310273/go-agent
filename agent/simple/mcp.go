package simple

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
)

// MCPToolWrapper represents a tool provided by MCP server
type MCPToolWrapper struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Schema        map[string]any `json:"params,omitempty"`
	isDestructive bool
	context       core.Context // agent session runtime context
}

// MCPRemoteUtil provides utilities for interacting with MCP server
type MCPRemoteUtil struct {
	Config *core.MCPConfig
	tools  map[string]core.Tool // cached MCPToolWrapper instances, name -> tool
}

// interface assertions
var _ core.Tool = (*MCPToolWrapper)(nil)
var _ core.MCPAccessible = (*MCPRemoteUtil)(nil)

// MCPToolWrapper implements core.Tool interface

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

func (m *MCPToolWrapper) IsDestructive() bool {
	return m.isDestructive
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
// validates serverName and toolName are present
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
// parses serverName and toolName from args, gets client from context
func (m *MCPToolWrapper) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		// parse serverName and toolName from args
		serverName, _ := args["serverName"].(string)
		toolName, _ := args["toolName"].(string)

		// get MCP client from context by server name
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
			core.LogStd(core.LogLevelError, "mcp tool call failed: tool=%s, error=%v", m.Name, err)
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

// MCPRemoteUtil implements core.MCPAccessible interface

// GetConfig returns MCP configuration
func (m *MCPRemoteUtil) GetConfig() *core.MCPConfig {
	return m.Config
}

// SetConfig sets MCP configuration
func (m *MCPRemoteUtil) SetConfig(config *core.MCPConfig) {
	m.Config = config
}

// BuildTools fetches tools from MCP server and builds Tool instances.
// combines list and build, returns cached MCPToolWrapper instances
func (m *MCPRemoteUtil) BuildTools(context core.Context) map[string]core.Tool {
	// use sendRequest to get SSE support
	resp, diag := m.sendRequest(core.MCPMethodToolsList, nil)
	if diag != nil {
		core.LogStd(core.LogLevelError, "%s failed: %s", core.MCPMethodToolsList, diag.Message)
		return nil
	}

	// directly parse Result into MCPListToolsResponse
	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult core.MCPListToolsResponse
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		core.LogStd(core.LogLevelError, "failed to parse tools list result: %v", err)
		return nil
	}

	// build MCPToolWrapper instances
	toolsMap := make(map[string]core.Tool)
	for _, toolInfo := range toolsResult.Tools {
		mcpTool := &MCPToolWrapper{
			Name:          toolInfo.Name,
			Description:   toolInfo.Description,
			Schema:        toolInfo.InputSchema,
			isDestructive: toolInfo.IsDestructive(),
			context:       context,
		}
		toolsMap[toolInfo.Name] = mcpTool
	}

	// cache the built tools
	m.tools = toolsMap
	core.LogStd(core.LogLevelInfo, "built and cached %d tools from mcp server", len(toolsMap))
	return toolsMap
}

// ListPrompts lists available prompts from MCP server.
// sends prompts/list request to MCP server
// now returns MCPServerResponse
func (m *MCPRemoteUtil) ListPrompts() (core.MCPServerResponse, *core.Diagnostic) {
	resp, diag := m.sendRequest(core.MCPMethodPromptsList, nil)
	if diag != nil {
		return core.MCPServerResponse{}, diag
	}
	return *resp, nil
}

// GetPrompt gets a specific prompt from MCP server.
// sends prompts/get request to MCP server
// parses response and extracts prompt content
func (m *MCPRemoteUtil) GetPrompt(name string) (string, *core.Diagnostic) {
	params := map[string]any{
		"name": name,
	}
	resp, diag := m.sendRequest(core.MCPMethodPromptsGet, params)
	if diag != nil {
		return "", diag
	}

	// parse the response to extract prompt content
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

		// extract text from messages
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
// sends resources/list request to MCP server
// now returns MCPServerResponse
func (m *MCPRemoteUtil) ListResources() (core.MCPServerResponse, *core.Diagnostic) {
	resp, diag := m.sendRequest(core.MCPMethodResourcesList, nil)
	if diag != nil {
		return core.MCPServerResponse{}, diag
	}
	return *resp, nil
}

// GetResource gets a specific resource from MCP server.
// sends resources/read request to MCP server
// parses response and extracts resource content
func (m *MCPRemoteUtil) GetResource(name string) (string, *core.Diagnostic) {
	params := map[string]any{
		"uri": name,
	}
	resp, diag := m.sendRequest(core.MCPMethodResourcesRead, params)
	if diag != nil {
		return "", diag
	}

	// parse the response to extract resource content
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

		// extract text from contents
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

// parseSSEResponse parses SSE format and extracts JSON data
// SSE format: "event: message\ndata: {...}\n\n"
// supports multiple data lines, concatenates them
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
	// if multiple data lines, concatenate them (for streaming responses)
	return []byte(strings.Join(dataLines, ""))
}

// sendRequest sends a request to MCP server with the given method and params.
// uses utils.SendRequest, generic helper for all MCP requests.
// logs only a concise summary with method name, status, latency, and body sizes.
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

	bodyJSON, _ := json.Marshal(body)
	startTime := time.Now()

	// send request
	config := utils.HTTPConfig{URL: m.getURL()}
	resp, err := utils.SendRequest(config, request)
	if err != nil {
		core.LogStd(core.LogLevelError, "mcp request failed: method=%s, server=%s, error=%v, latency=%dms",
			jsonrpcMethod, m.Config.Name, err, time.Since(startTime).Milliseconds())
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: fmt.Sprintf("%s failed", jsonrpcMethod),
			Data:    err.Error(),
		}
	}
	defer resp.Body.Close()

	// read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		core.LogStd(core.LogLevelError, "mcp read response body failed: method=%s, server=%s, error=%v",
			jsonrpcMethod, m.Config.Name, err)
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to read response body",
			Data:    err.Error(),
		}
	}

	latency := time.Since(startTime).Milliseconds()

	// parse response based on content-type
	contentType := resp.Header.Get("Content-Type")
	var jsonBody []byte

	if strings.Contains(contentType, "text/event-stream") {
		// parse SSE format
		jsonBody = parseSSEResponse(respBody)
		if jsonBody == nil {
			core.LogStd(core.LogLevelError, "mcp sse parse failed: method=%s, server=%s", jsonrpcMethod, m.Config.Name)
			return nil, &core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: "failed to parse SSE response",
				Data:    "no data field found in SSE response",
			}
		}
	} else {
		// direct JSON response
		jsonBody = respBody
	}

	// parse response body
	var serverResp core.MCPServerResponse
	if err := json.Unmarshal(jsonBody, &serverResp); err != nil {
		core.LogStd(core.LogLevelError, "mcp decode response failed: method=%s, server=%s, error=%v",
			jsonrpcMethod, m.Config.Name, err)
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to decode response",
			Data:    err.Error(),
		}
	}

	// concise summary: method, status, latency, body sizes, includes key params if present
	if params != nil {
		core.LogStd(core.LogLevelDebug, "mcp call: server=%s, method=%s, status=%d, latency=%dms, req=%dB, resp=%dB",
			m.Config.Name, jsonrpcMethod, resp.StatusCode, latency, len(bodyJSON), len(respBody))
	} else {
		core.LogStd(core.LogLevelDebug, "mcp call: server=%s, method=%s, status=%d, latency=%dms, req=%dB, resp=%dB",
			m.Config.Name, jsonrpcMethod, resp.StatusCode, latency, len(bodyJSON), len(respBody))
	}
	return &serverResp, nil
}

// CallTool calls a tool on the MCP server with the given name and arguments.
// uses sendRequest to get SSE support
func (m *MCPRemoteUtil) CallTool(name string, arguments map[string]any) (string, any, error) {
	// build params for tools/call
	params := map[string]any{
		"name":      name,
		"arguments": arguments,
	}

	// use sendRequest
	resp, diag := m.sendRequest(core.MCPMethodToolsCall, params)
	if diag != nil {
		return "", nil, fmt.Errorf("failed to call tool %s: %s", name, diag.Message)
	}

	// check for error
	if resp.Error != nil {
		return "", nil, fmt.Errorf("tool call failed: %s", resp.Error.Message)
	}

	// parse result using MCPCallToolResult
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal tool result: %w", err)
	}

	var callResult core.MCPCallToolResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}

	// extract text from content array
	var texts []string
	for _, content := range callResult.Content {
		if content.Text != "" {
			texts = append(texts, content.Text)
		}
	}

	return strings.Join(texts, "\n"), resp.Result, nil
}
