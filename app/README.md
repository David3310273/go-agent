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
  "model": "string (optional)",
  "enableThinking": "bool (optional)",
  "stream": "bool (optional)"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| sessionID | string | No | Session ID for conversation continuity. If not provided, a new session will be created. |
| question | string | Yes | The question to ask the agent. |
| model | string | No | The model agent used in this request. |
| enableThinking | bool | No | Whether to enable thinking. |
| stream | bool | No | Whether to stream the response. |

**Example:**
```bash
curl -X POST http://localhost:8080/v1/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Go?", "sessionID": "test-123"}'
```

#### Response

**Success (200 OK):**
```json
{
  "sessionID": "test-123",
  "answer": "...",
  "model": "...",
  "usage": {
    "promptTokens": 0,
    "completionTokens": 0,
    "totalTokens": 0
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| sessionID | string | The session ID associated with this conversation. |
| answer | string | The agent's response. |
| model | string | The model agent used in this request. |
| usage | object | The token usage of this request. |

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
- **Also support streaming mode**, try with header `Accept: text/event-stream` and `stream: true` in request body
- **Create schema first if you use milvus knowledge base in the app, please refer to [milvus official doc](https://milvus.io/api-reference/restful/v2.6.x/v2/Collection%20%28v2%29/Create.md]**

---

## Internal Knowledge Base API

Internal APIs for managing knowledge base entries when it's a toB app. All endpoints require `storageType` parameter for validation.

### POST /v1/internal/knowledgebase

Create a new knowledge base entry by uploading a file.

#### Request

**Headers:**
```
Content-Type: multipart/form-data
```

**Form Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| storageType | string | Yes | Storage backend type. Currently only `milvus` is supported. |
| domain | string | Yes | Domain of the knowledge base. Currently only `technology` is supported. |
| contentType | string | Yes | Content type for the knowledge entry. |
| file | file | Yes | The file to upload (.md, .txt, .pdf, .docx, .xlsx). |

**Example:**
```bash
curl -X POST http://localhost:8080/v1/internal/knowledgebase \
  -F "storageType=milvus" \
  -F "domain=technology" \
  -F "contentType=markdown" \
  -F "file=@document.md"
```

#### Response

**Success (201 Created):**
```json
{
  "insertCount": 10
}
```

| Field | Type | Description |
|-------|------|-------------|
| insertCount | int | Number of chunks inserted into the knowledge base. |

---

### GET /v1/internal/knowledgebase

Search knowledge base entries by keyword, filename, or contentType.

#### Request

**Query Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| storageType | string | Yes | Storage backend type. Currently only `milvus` is supported. |
| domain | string | Yes | Domain of the knowledge base. Currently only `technology` is supported. |
| keyword | string | Conditional | Search keyword for semantic search. At least one of `keyword`, `filename`, or `contentType` is required. |
| filename | string | Conditional | Filter by filename. At least one of `keyword`, `filename`, or `contentType` is required. |
| contentType | string | Conditional | Filter by content type. At least one of `keyword`, `filename`, or `contentType` is required. |
| topK | int | No | Number of results to return. Default is 10. |

**Example:**
```bash
curl -X GET "http://localhost:8080/v1/internal/knowledgebase?storageType=milvus&domain=technology&keyword=sliding+window&topK=5"
```

#### Response

**Success (200 OK):**
```json
{
  "results": [
    {
      "content": "...",
      "filename": "document.md",
      "domain": "technology"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| results | array | Array of matching knowledge base entries. |

---

### PUT /v1/internal/knowledgebase

Update a knowledge base entry by deleting existing chunks and creating new ones.

#### Request

**Headers:**
```
Content-Type: multipart/form-data
```

**Form Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| storageType | string | Yes | Storage backend type. Currently only `milvus` is supported. |
| domain | string | Yes | Domain of the knowledge base. Currently only `technology` is supported. |
| contentType | string | Yes | Content type for the knowledge entry. |
| file | file | Yes | The new file to upload. Existing chunks with the same filename will be deleted. |

**Example:**
```bash
curl -X PUT http://localhost:8080/v1/internal/knowledgebase \
  -F "storageType=milvus" \
  -F "domain=technology" \
  -F "contentType=markdown" \
  -F "file=@updated_document.md"
```

#### Response

**Success (200 OK):**
```json
{
  "insertCount": 12
}
```

| Field | Type | Description |
|-------|------|-------------|
| insertCount | int | Number of new chunks inserted. |

---

### DELETE /v1/internal/knowledgebase

Delete knowledge base entries by filename and/or contentType.

#### Request

**Query Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| storageType | string | Yes | Storage backend type. Currently only `milvus` is supported. |
| domain | string | Yes | Domain of the knowledge base. Currently only `technology` is supported. |
| filename | string | Conditional | Delete entries by filename. At least one of `filename` or `contentType` is required. |
| contentType | string | Conditional | Delete entries by content type. At least one of `filename` or `contentType` is required. |

**Example:**
```bash
# Delete by filename
curl -X DELETE "http://localhost:8080/v1/internal/knowledgebase?storageType=milvus&domain=technology&filename=document.md"

# Delete by contentType
curl -X DELETE "http://localhost:8080/v1/internal/knowledgebase?storageType=milvus&domain=technology&contentType=markdown"
```

#### Response

**Success (200 OK):**
```json
{
  "success": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| success | bool | Whether the deletion was successful. |

---

#### Notes

- All internal endpoints validate `storageType` in middleware before processing.
- Currently only `milvus` storage type and `technology` domain are supported.
- Files are processed using format-aware loaders that chunk and embed the content.