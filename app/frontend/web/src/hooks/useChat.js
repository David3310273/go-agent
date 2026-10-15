import { useState, useCallback, useRef } from 'react';
import { askAgent, cancelSession, parseSSEStream } from '../api/agent';

/**
 * useChat - 聊天核心 Hook
 *
 * 完整对接后端 SSE 流式接口:
 *   POST /v1/agent/ask   → 发送问题，接收流式回答
 *   POST /v1/agent/cancel → 取消正在进行的会话
 *
 * SSE 事件处理:
 *   event:message      → 追加到回答内容，实时渲染
 *   event:thought      → 思考过程（暂存，可选展示）
 *   event:done         → 流结束，保存 sessionID 和 usage
 *   event:tool_confirm → 危险工具需用户确认（暂存，待用户操作）
 *   event:error        → 错误处理
 */
export function useChat() {
  const [messages, setMessages] = useState([]);
  const [currentSession, setCurrentSession] = useState(() => crypto.randomUUID());
  const currentSessionRef = useRef(currentSession);
  const [isLoading, setIsLoading] = useState(false);
  const [thought, setThought] = useState('');
  const [pendingConfirms, setPendingConfirms] = useState(null);
  const [usage, setUsage] = useState(null);
  const abortControllerRef = useRef(null);

  // per-session message cache: sessionId → messages[]
  const messagesCacheRef = useRef(new Map());
  // per-session generation counter: sessionId → gen (incremented each sendMessage/confirmToolCall)
  const sessionGenRef = useRef(new Map());

  const updateCurrentSession = (sessionID) => {
    currentSessionRef.current = sessionID;
    setCurrentSession(sessionID);
  };

  /**
   * 发送消息
   * @param {string} question - 用户输入的问题
   * @param {boolean} enableThinking - 是否开启深度思考
   */
  const sendMessage = useCallback(async (question, enableThinking = false) => {
    if (!question.trim() || isLoading) return;

    const sid = currentSessionRef.current;
    const gen = (sessionGenRef.current.get(sid) || 0) + 1;
    sessionGenRef.current.set(sid, gen);

    const userMessage = {
      id: `msg-${Date.now()}-user`,
      role: 'user',
      content: question.trim(),
      timestamp: new Date().toISOString(),
    };

    const assistantMessageId = `msg-${Date.now()}-assistant`;
    const assistantMessage = {
      id: assistantMessageId,
      role: 'assistant',
      content: '',
      timestamp: new Date().toISOString(),
      isStreaming: true,
    };

    const prevMessages = messagesCacheRef.current.get(sid) || [];
    const newMessages = [...prevMessages, userMessage, assistantMessage];
    messagesCacheRef.current.set(sid, newMessages);
    setMessages(newMessages);
    setIsLoading(true);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    let isTimeout = false;
    let timeoutId;
    const resetTimeout = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        isTimeout = true;
        controller.abort();
      }, 60000);
    };
    resetTimeout();

    try {
      const response = await askAgent({
        question: question.trim(),
        sessionID: sid,
        type: 'normal',
        stream: true,
        enableThinking,
        signal: controller.signal,
      });

      let accumulatedContent = '';

      for await (const { event, data } of parseSSEStream(response, controller.signal)) {
        resetTimeout();
        if (sessionGenRef.current.get(sid) !== gen) return;

        switch (event) {
          case 'message':
            accumulatedContent += data;
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;

          case 'thought':
            setThought(prev => prev + data);
            break;

          case 'done':
            if (data.sessionID && data.sessionID !== sid) {
              const msgs = messagesCacheRef.current.get(sid) || [];
              messagesCacheRef.current.delete(sid);
              messagesCacheRef.current.set(data.sessionID, msgs);
              if (currentSessionRef.current === sid) {
                updateCurrentSession(data.sessionID);
              }
            } else if (data.sessionID) {
              if (currentSessionRef.current === sid) {
                updateCurrentSession(data.sessionID);
              }
            }
            if (data.usage) {
              setUsage(data.usage);
            }
            {
              const finalSid = data.sessionID || sid;
              const cached = messagesCacheRef.current.get(finalSid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(finalSid, updated);
              if (currentSessionRef.current === finalSid) {
                setMessages(updated);
              }
            }
            break;

          case 'tool_confirm':
            setPendingConfirms(data);
            if (data.sessionID && data.sessionID !== sid) {
              const msgs = messagesCacheRef.current.get(sid) || [];
              messagesCacheRef.current.delete(sid);
              messagesCacheRef.current.set(data.sessionID, msgs);
              if (currentSessionRef.current === sid) {
                updateCurrentSession(data.sessionID);
              }
            } else if (data.sessionID) {
              if (currentSessionRef.current === sid) {
                updateCurrentSession(data.sessionID);
              }
            }
            {
              const finalSid = data.sessionID || sid;
              const cached = messagesCacheRef.current.get(finalSid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(finalSid, updated);
              if (currentSessionRef.current === finalSid) {
                setMessages(updated);
              }
            }
            break;

          case 'error':
            console.error('Stream error:', data);
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent || `Error: ${data}`, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;

          default:
            console.warn('Unknown SSE event:', event, data);
        }
      }
    } catch (error) {
      if (sessionGenRef.current.get(sid) !== gen) return;

      if (isTimeout) {
        console.error('Request timeout');
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, content: '请求超时: Request Timeout', isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      } else if (error.name === 'AbortError') {
        console.log('Request aborted by user');
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      } else {
        console.error('API Error:', error);
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, content: `请求失败: ${error.message}`, isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      }
    } finally {
      clearTimeout(timeoutId);
      if (sessionGenRef.current.get(sid) === gen) {
        sessionGenRef.current.delete(sid);
        messagesCacheRef.current.delete(sid);
        if (currentSessionRef.current === sid) {
          setIsLoading(false);
        }
        abortControllerRef.current = null;
      }
    }
  }, [isLoading]);

  /**
   * 停止生成 - 调用后端 cancel 接口 + 前端 abort
   */
  const stopGeneration = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    const sid = currentSessionRef.current;
    sessionGenRef.current.delete(sid);
    setIsLoading(false);
    setThought('');
    const cached = messagesCacheRef.current.get(sid) || [];
    const updated = cached.map(msg =>
      msg.isStreaming ? { ...msg, isStreaming: false } : msg
    );
    messagesCacheRef.current.set(sid, updated);
    setMessages(updated);
    if (sid) {
      cancelSession(sid).catch((error) => {
        console.error('Cancel failed:', error);
      });
    }
  }, []);

  /**
   * 确认工具调用 - 用户对 tool_confirm 事件做出 Yes/No 回答
   * @param {Array} confirms - [{ toolCallID, serverName, toolName, answer: "Yes"|"No" }]
   */
  const confirmToolCall = useCallback(async (confirms) => {
    if (!currentSessionRef.current || !pendingConfirms) return;

    const sid = currentSessionRef.current;
    const gen = (sessionGenRef.current.get(sid) || 0) + 1;
    sessionGenRef.current.set(sid, gen);

    setIsLoading(true);
    setPendingConfirms(null);

    const assistantMessageId = `msg-${Date.now()}-assistant-confirm`;
    const assistantMessage = {
      id: assistantMessageId,
      role: 'assistant',
      content: '',
      timestamp: new Date().toISOString(),
      isStreaming: true,
    };

    const prevMessages = messagesCacheRef.current.get(sid) || [];
    const newMessages = [...prevMessages, assistantMessage];
    messagesCacheRef.current.set(sid, newMessages);
    setMessages(newMessages);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    let isTimeout = false;
    let timeoutId;
    const resetTimeout = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        isTimeout = true;
        controller.abort();
      }, 60000);
    };
    resetTimeout();

    try {
      const response = await askAgent({
        question: '',
        sessionID: sid,
        type: 'tool_confirm',
        stream: true,
        confirms,
        signal: controller.signal,
      });

      let accumulatedContent = '';

      for await (const { event, data } of parseSSEStream(response, controller.signal)) {
        resetTimeout();
        if (sessionGenRef.current.get(sid) !== gen) return;

        switch (event) {
          case 'message':
            accumulatedContent += data;
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;

          case 'done':
            if (data.sessionID && currentSessionRef.current === sid) {
              updateCurrentSession(data.sessionID);
            }
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;

          case 'tool_confirm':
            setPendingConfirms(data);
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;

          case 'error':
            console.error('Stream error:', data);
            {
              const cached = messagesCacheRef.current.get(sid) || [];
              const updated = cached.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent || `Error: ${data}`, isStreaming: false }
                  : msg
              );
              messagesCacheRef.current.set(sid, updated);
              if (currentSessionRef.current === sid) {
                setMessages(updated);
              }
            }
            break;
        }
      }
    } catch (error) {
      if (sessionGenRef.current.get(sid) !== gen) return;

      if (isTimeout) {
        console.error('Confirm request timeout');
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, content: '请求超时: Request Timeout', isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      } else if (error.name === 'AbortError') {
        console.log('Confirm request aborted by user');
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      } else {
        console.error('Confirm error:', error);
        const cached = messagesCacheRef.current.get(sid) || [];
        const updated = cached.map(msg =>
          msg.id === assistantMessageId
            ? { ...msg, content: `请求失败: ${error.message}`, isStreaming: false }
            : msg
        );
        messagesCacheRef.current.set(sid, updated);
        if (currentSessionRef.current === sid) {
          setMessages(updated);
        }
      }
    } finally {
      clearTimeout(timeoutId);
      if (sessionGenRef.current.get(sid) === gen) {
        sessionGenRef.current.delete(sid);
        messagesCacheRef.current.delete(sid);
        if (currentSessionRef.current === sid) {
          setIsLoading(false);
        }
        abortControllerRef.current = null;
      }
    }
  }, [pendingConfirms]);

  /**
   * 清空聊天 - 重置所有状态，保留当前会话的 stream 继续运行
   */
  const clearChat = useCallback((newSessionId) => {
    // save current session messages to cache before switching
    if (currentSessionRef.current) {
      messagesCacheRef.current.set(currentSessionRef.current, [...messages]);
    }
    const newId = newSessionId || crypto.randomUUID();
    messagesCacheRef.current.set(newId, []);
    setMessages([]);
    updateCurrentSession(newId);
    setIsLoading(false);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);
  }, [messages]);

  const restoreSession = useCallback((sessionID, sessionMessages) => {
    // save current session messages to cache before switching
    messagesCacheRef.current.set(currentSessionRef.current, [...messages]);

    updateCurrentSession(sessionID);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);

    const isStreaming = sessionGenRef.current.has(sessionID);
    const cached = messagesCacheRef.current.get(sessionID);

    if (isStreaming && cached !== undefined) {
      // active stream — show cached messages with live rendering
      setMessages(cached);
      setIsLoading(true);
    } else if (sessionMessages && sessionMessages.length > 0) {
      // stream completed or never streamed — load from API
      const msgs = sessionMessages.map((msg) => ({
        id: `msg-${Date.now()}-${msg.role}-${Math.random().toString(36).slice(2, 8)}`,
        role: msg.role,
        content: msg.content || '',
        timestamp: new Date().toISOString(),
      }));
      messagesCacheRef.current.set(sessionID, msgs);
      setMessages(msgs);
      setIsLoading(false);
    } else {
      setMessages([]);
      setIsLoading(false);
    }
  }, [messages]);

  return {
    messages,
    currentSession,
    isLoading,
    thought,
    pendingConfirms,
    usage,
    sendMessage,
    stopGeneration,
    confirmToolCall,
    clearChat,
    restoreSession,
  };
}