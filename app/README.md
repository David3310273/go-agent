# Simple agent server

## Start your first agent service

1. **MUST** cd to `app` directory
2. run `go run main.go`
3. send post request to `/v1/ask`

## V1.0.0

happy path for simple agent

### why it is named simple?

- no router for model select
- no streaming response
- no support for local knowledge base
- no dynamic context selected from knowledge base

## API

### POST /v1/ask

Ask a question to the agent.

#### Request

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "sessionID": "string (optional)",
  "question": "string (required)",
  "model": "string (optional)"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| sessionID | string | No | Session ID for conversation continuity. If not provided, a new session will be created. |
| question | string | Yes | The question to ask the agent. |
| model | string | No | The model agent used in this request. |

**Example:**
```bash
curl -X POST http://localhost:8080/v1/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Go?", "sessionId": "test-123"}'
```

#### Response

**Success (200 OK):**
```json
{
  "sessionId": "test-123",
  "answer": "...",
  "model": "..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| sessionID | string | The session ID associated with this conversation. |
| answer | string | The agent's response. |
| model | string | The model agent used in this request. |

**Bad Request (400):**
```json
{
  "error": "invalid request: ..."
}
```

**Timeout (408):**
```json
{
  "error": "request timeout"
}
```

**Internal Server Error (500):**
```json
{
  "error": "agent not available"
}
```

#### Notes

- The request timeout is configured in `config.json` (`maxWaitingSeconds`).
- If the agent cannot respond within the timeout, a 408 status is returned.