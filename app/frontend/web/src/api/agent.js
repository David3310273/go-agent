const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

/**
 * Ask agent a question
 * @param {Object} params - Request parameters
 * @param {string} params.question - User question
 * @param {string} [params.sessionID] - Optional session ID
 * @param {string} [params.type] - Question type
 * @param {boolean} [params.stream] - Enable streaming
 * @returns {Promise<Response>} Fetch response
 */
export async function askAgent({ question, sessionID, type = 'normal', stream = true }) {
  const response = await fetch(`${API_BASE_URL}/v1/agent/ask`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      question,
      sessionID,
      type,
      stream,
    }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response;
}

/**
 * Cancel a running session
 * @param {string} sessionID - Session ID to cancel
 * @returns {Promise<Object>} Response data
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
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

/**
 * Parse SSE stream and yield events
 * @param {Response} response - Fetch response
 * @param {AbortSignal} signal - Abort signal
 * @yields {Object} Parsed SSE event
 */
export async function* parseSSEStream(response, signal) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      if (signal.aborted) {
        reader.cancel();
        break;
      }

      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      let currentEvent = 'message';

      for (const line of lines) {
        const trimmed = line.trim();

        if (trimmed.startsWith('event:')) {
          currentEvent = trimmed.slice(6).trim();
          continue;
        }

        if (trimmed.startsWith('data:')) {
          const dataStr = trimmed.slice(5).trim();
          if (!dataStr || dataStr === '[DONE]') continue;

          try {
            const data = JSON.parse(dataStr);
            yield { event: currentEvent, data };
          } catch {
            if (dataStr) {
              yield { event: currentEvent, data: dataStr };
            }
          }
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}