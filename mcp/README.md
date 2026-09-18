# MCP server plugin

Publish tools/resources in this framework as MCP services.

## Configuration

`config.json`:

```json
{
    "rootPath": "[Your root path of the project]",
    "port": "8080",
    "tools": ["YourToolName"]
}
```

| Field | Description |
|-------|-------------|
| `rootPath` | Project root directory path, used to load tool schema files |
| `port` | HTTP server port |
| `tools` | List of tool names to register |

## Start Server

```bash
cd mcp
go build -o mcp-server ./server
./mcp-server
```

Output:
```
[MCP Server] root path: [Your root path of the project], port: 8080, tools: [GetDate]
[MCP] registered tool: GetDate
[MCP Server] starting on :8080
[MCP Server] endpoint: http://localhost:8080/mcp
```

## Interaction Protocol examples

MCP uses **JSON-RPC 2.0** protocol, interacting with the `/mcp` endpoint via HTTP POST requests. For more details, see the [official documentation](https://modelcontextprotocol.io/docs/2026-07-28/learn/server-concepts).

### 1. Initialize Connection

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "test-client",
        "version": "1.0"
      }
    }
  }'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "capabilities": {"logging": {}, "tools": {"listChanged": true}},
    "protocolVersion": "2024-11-05",
    "serverInfo": {"name": "go-agent-mcp-server", "version": "1.0.0"}
  }
}
```

### 2. List Available Tools

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "GetDate",
        "description": "get system current date and time.",
        "inputSchema": {"type": "object", "additionalProperties": true}
      }
    ]
  }
}
```

### 3. Call Tool

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "GetDate",
      "arguments": {}
    }
  }'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {"type": "text", "text": "2026-09-16 14:18:01"}
    ]
  }
}
```

## Test Client

```bash
cd mcp/client
go build -o mcp-client .
./mcp-client
```

## Available example tools

| Tool Name | Description | Parameters |
|-----------|-------------|------------|
| `GetDate` | Get current date and time | None |

## Adding New Tools

1. Create tool implementation in `tools/`
2. Call `core.RegisterTool()` in `init()` to register
3. Add tool name to `tools` array in `mcp/config.json`
