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

// MCPRemoteUtil provides utilities for interacting with MCP server
type MCPRemoteUtil struct {
	Config *core.MCPConfig
	tools  []core.MCPListToolResult // cached tool definitions from tools/list
}

// interface assertions
var _ core.MCPAccessible = (*MCPRemoteUtil)(nil)

// MCPRemoteUtil implements core.MCPAccessible interface

// GetConfig returns MCP configuration
func (m *MCPRemoteUtil) GetConfig() *core.MCPConfig {
	return m.Config
}

// SetConfig sets MCP configuration
func (m *MCPRemoteUtil) SetConfig(config *core.MCPConfig) {
	m.Config = config
}

// ListTools fetches tools from MCP server and returns tool definitions.
func (m *MCPRemoteUtil) ListTools() ([]core.MCPListToolResult, *core.Diagnostic) {
	// use sendRequest to get SSE support
	resp, diag := m.sendRequest(core.MCPMethodToolsList, nil)
	if diag != nil {
		core.LogStd(core.LogLevelError, "%s failed: %s", core.MCPMethodToolsList, diag.Message)
		return nil, diag
	}

	// directly parse Result into MCPListToolsResponse
	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult core.MCPListToolsResponse
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		core.LogStd(core.LogLevelError, "failed to parse tools list result: %v", err)
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSystemError,
			Message: "failed to parse tools list result",
			Data:    err.Error(),
		}
	}

	// cache the tools
	m.tools = toolsResult.Tools
	core.LogStd(core.LogLevelInfo, "fetched and cached %d tools from mcp server", len(m.tools))
	return m.tools, nil
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
