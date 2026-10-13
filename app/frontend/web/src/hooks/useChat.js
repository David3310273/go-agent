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
  const [isLoading, setIsLoading] = useState(false);
  const [thought, setThought] = useState('');
  const [pendingConfirms, setPendingConfirms] = useState(null);
  const [usage, setUsage] = useState(null);
  const abortControllerRef = useRef(null);

  /**
   * 发送消息
   * @param {string} question - 用户输入的问题
   * @param {boolean} enableThinking - 是否开启深度思考
   * @param {boolean} enableSearch - 是否开启搜索
   */
  const sendMessage = useCallback(async (question, enableThinking = false) => {
    if (!question.trim() || isLoading) return;

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

    setMessages(prev => [...prev, userMessage, assistantMessage]);
    setIsLoading(true);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const response = await askAgent({
        question: question.trim(),
        sessionID: currentSession,
        type: 'normal',
        stream: true,
        enableThinking,
        signal: controller.signal,
      });

      // 5. 解析 SSE 流
      let accumulatedContent = '';

      for await (const { event, data } of parseSSEStream(response, abortControllerRef.current.signal)) {
        switch (event) {
          case 'message':
            // event:message → data 是字符串片段，追加到回答内容
            accumulatedContent += data;
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent } // 持续追加回答内容
                  : msg
              )
            );
            break;

          case 'thought':
            // event:thought → data 是思考过程字符串
            setThought(prev => prev + data);
            break;

          case 'done':
            // event:done → data = { type: "normal", sessionID: "xxx", usage: {...} }
            if (data.sessionID) {
              setCurrentSession(data.sessionID);
            }
            if (data.usage) {
              setUsage(data.usage);
            }
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              )
            );
            break;

          case 'tool_confirm':
            // event:tool_confirm → data = { type, confirms, message, sessionID, usage }
            setPendingConfirms(data);
            if (data.sessionID) {
              setCurrentSession(data.sessionID);
            }
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              )
            );
            break;

          case 'error':
            // event:error → data 是错误消息字符串
            console.error('Stream error:', data);
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent || `Error: ${data}`, isStreaming: false }
                  : msg
              )
            );
            break;

          default:
            console.warn('Unknown SSE event:', event, data);
        }
      }
    } catch (error) {
      if (error.name === 'AbortError') {
        console.log('Request aborted by user');
        setMessages(prev =>
          prev.map(msg =>
            msg.id === assistantMessageId
              ? { ...msg, isStreaming: false }
              : msg
          )
        );
      } else {
        console.error('API Error:', error);
        setMessages(prev =>
          prev.map(msg =>
            msg.id === assistantMessageId
              ? { ...msg, content: `请求失败: ${error.message}`, isStreaming: false }
              : msg
          )
        );
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [currentSession, isLoading]); // 依赖项

  /**
   * 停止生成 - 调用后端 cancel 接口 + 前端 abort
   */
  const stopGeneration = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsLoading(false);
    setThought('');
    setMessages(prev =>
      prev.map(msg =>
        msg.isStreaming ? { ...msg, isStreaming: false } : msg
      )
    );
    if (currentSession) {
      cancelSession(currentSession).catch((error) => {
        console.error('Cancel failed:', error);
      });
    }
  }, [currentSession]);

  /**
   * 确认工具调用 - 用户对 tool_confirm 事件做出 Yes/No 回答
   * @param {Array} confirms - [{ toolCallID, serverName, toolName, answer: "Yes"|"No" }]
   */
  const confirmToolCall = useCallback(async (confirms) => {
    if (!currentSession || !pendingConfirms) return;

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

    setMessages(prev => [...prev, assistantMessage]);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const response = await askAgent({
        question: '',
        sessionID: currentSession,
        type: 'tool_confirm',
        stream: true,
        confirms,
        signal: controller.signal,
      });

      let accumulatedContent = '';

      for await (const { event, data } of parseSSEStream(response, abortControllerRef.current.signal)) {
        switch (event) {
          case 'message':
            accumulatedContent += data;
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent }
                  : msg
              )
            );
            break;

          case 'done':
            if (data.sessionID) {
              setCurrentSession(data.sessionID);
            }
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              )
            );
            break;

          case 'tool_confirm':
            setPendingConfirms(data);
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, isStreaming: false }
                  : msg
              )
            );
            break;

          case 'error':
            console.error('Stream error:', data);
            setMessages(prev =>
              prev.map(msg =>
                msg.id === assistantMessageId
                  ? { ...msg, content: accumulatedContent || `Error: ${data}`, isStreaming: false }
                  : msg
              )
            );
            break;
        }
      }
    } catch (error) {
      if (error.name !== 'AbortError') {
        console.error('Confirm error:', error);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [currentSession, pendingConfirms]);

  /**
   * 清空聊天 - 重置所有状态
   */
  const clearChat = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setMessages([]);
    setCurrentSession(crypto.randomUUID());
    setIsLoading(false);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);
  }, []);

  const restoreSession = useCallback((sessionID, sessionMessages) => {
    setCurrentSession(sessionID);
    setMessages(sessionMessages || []);
    setThought('');
    setPendingConfirms(null);
    setUsage(null);
  }, []);

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