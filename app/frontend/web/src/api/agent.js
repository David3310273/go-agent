const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

/**
 * Ask agent a question (streaming)
 *
 * 对应后端 POST /v1/agent/ask
 * 请求体: { question, sessionID, type, stream, enableThinking }
 * 响应: SSE 流 (event:message / event:thought / event:done / event:tool_confirm / event:error)
 *
 * @param {Object} params
 * @param {string} params.question - 用户问题
 * @param {string} [params.sessionID] - 会话ID（首次为null，后续带上）
 * @param {'normal'|'tool_confirm'} [params.type] - 问题类型
 * @param {boolean} [params.stream] - 是否流式
 * @param {boolean} [params.enableThinking] - 是否开启思考
 * @param {Array} [params.confirms] - tool_confirm 类型的确认列表
 * @returns {Promise<Response>}
 */
export async function askAgent({
  question,
  sessionID,
  type = 'normal',
  stream = true,
  enableThinking = false,
  confirms,
  signal,
}) {
  const body = {
    question,
    type,
    stream,
    enableThinking,
  };

  if (sessionID) {
    body.sessionID = sessionID;
  }

  if (type === 'tool_confirm' && confirms) {
    body.confirms = confirms;
  }

  const response = await fetch(`${API_BASE_URL}/v1/agent/ask`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
    signal,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response;
}

/**
 * Cancel a running session
 *
 * 对应后端 POST /v1/agent/cancel
 * 请求体: { sessionID }
 * 响应: { sessionID, message }
 *
 * @param {string} sessionID
 * @returns {Promise<{sessionID: string, message: string}>}
 */
export async function cancelSession(sessionID) {
  const response = await fetch(`${API_BASE_URL}/v1/agent/cancel`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ sessionID }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Cancel failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

/**
 * Parse SSE stream from backend
 *
 * 后端使用 gin 的 c.SSEvent() 发送，格式为:
 *   event: message\n
 *   data: "hello"\n\n
 *
 * - event:message  → data 是纯字符串（JSON字符串，如 "hello"）
 * - event:thought  → data 是纯字符串
 * - event:done     → data 是 JSON 对象 { type, sessionID, usage }
 * - event:tool_confirm → data 是 JSON 对象 { type, confirms, message, sessionID, usage }
 * - event:error    → data 是纯字符串
 *
 * @param {Response} response
 * @param {AbortSignal} [signal]
 * @yields {{ event: string, data: any }}
 */
export async function* parseSSEStream(response, signal) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  let buffer = '';
  let currentEvent = 'message';
  let dataLines = [];

  const flushEvent = () => {
    if (dataLines.length === 0) return null;

    const dataStr = dataLines.join('\n');
    dataLines = [];

    const event = currentEvent;
    currentEvent = 'message';

    if (event === 'message' || event === 'thought') {
      return { event, data: dataStr };
    }

    let data;
    try {
      data = JSON.parse(dataStr);
    } catch {
      data = dataStr;
    }

    return { event, data };
  };

  const processLine = (line) => {
    if (line.startsWith(':')) return null;

    if (line === '') {
      return flushEvent();
    }

    const colonIndex = line.indexOf(':');
    const field = colonIndex === -1
      ? line
      : line.slice(0, colonIndex);

    let value = colonIndex === -1 ? '' : line.slice(colonIndex + 1);

    if (field === 'event') {
      currentEvent = value || 'message';
    } else if (field === 'data') {
      dataLines.push(value);
    }

    return null;
  };

  try {
    while (true) {
      if (signal?.aborted) {
        await reader.cancel();
        break;
      }

      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // 保留可能跨 chunk 的 CRLF
      let start = 0;
      for (let i = 0; i < buffer.length; i++) {
        if (buffer[i] === '\n' || buffer[i] === '\r') {
          if (
            buffer[i] === '\r' &&
            i === buffer.length - 1
          ) {
            break;
          }

          const line = buffer.slice(start, i);
          const event = processLine(line);
          if (event) yield event;

          if (buffer[i] === '\r' && buffer[i + 1] === '\n') {
            i++;
          }

          start = i + 1;
        }
      }

      buffer = buffer.slice(start);
    }

    buffer += decoder.decode();

    if (buffer.length > 0) {
      const event = processLine(buffer);
      if (event) yield event;
    }

    const event = flushEvent();
    if (event) yield event;
  } finally {
    reader.releaseLock();
  }
}

/**
 * List all sessions from backend
 *
 * GET /v1/session/list
 * 响应: { sessionsInfo: [{ sessionID, title }] }
 */
export async function listSessions() {
  const response = await fetch(`${API_BASE_URL}/v1/session/list`);

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

/**
 * Get single session conversation history
 *
 * GET /v1/session?sessionID=xxx
 * 响应: { sessionID, messages: [string] }
 */
export async function getSessionHistory(sessionID) {
  const response = await fetch(`${API_BASE_URL}/v1/session?sessionID=${encodeURIComponent(sessionID)}`);

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

/**
 * Delete a session on the backend
 *
 * DELETE /v1/session?sessionID=xxx
 * 响应: { sessionID, message }
 */
export async function deleteSession(sessionID) {
  const response = await fetch(`${API_BASE_URL}/v1/session?sessionID=${encodeURIComponent(sessionID)}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json();
}