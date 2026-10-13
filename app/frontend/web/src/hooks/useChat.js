import { useState, useCallback, useRef } from 'react';
import { askAgent, parseSSEStream } from '../api/agent';

export function useChat() {
  const [messages, setMessages] = useState([]);
  const [currentSession, setCurrentSession] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const abortControllerRef = useRef(null);

  const sendMessage = useCallback(async (question) => {
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

    abortControllerRef.current = new AbortController();

    try {
      const response = await askAgent({
        question: question.trim(),
        sessionID: currentSession,
        type: 'normal',
        stream: true,
      });

      let accumulatedContent = '';

      for await (const { event, data } of parseSSEStream(response, abortControllerRef.current.signal)) {
        if (event === 'message') {
          if (typeof data === 'string') {
            accumulatedContent += data;
          } else if (data.content) {
            accumulatedContent += data.content;
          }

          setMessages(prev =>
            prev.map(msg =>
              msg.id === assistantMessageId
                ? { ...msg, content: accumulatedContent }
                : msg
            )
          );
        } else if (event === 'done') {
          if (data.sessionID && !currentSession) {
            setCurrentSession(data.sessionID);
          }

          setMessages(prev =>
            prev.map(msg =>
              msg.id === assistantMessageId
                ? { ...msg, isStreaming: false }
                : msg
            )
          );
        } else if (event === 'error') {
          console.error('Stream error:', data);
        }
      }
    } catch (error) {
      if (error.name === 'AbortError') {
        console.log('Request aborted');
      } else {
        console.error('Error:', error);
        setMessages(prev =>
          prev.map(msg =>
            msg.id === assistantMessageId
              ? { ...msg, content: 'Sorry, I encountered an error. Please try again.', isStreaming: false }
              : msg
          )
        );
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [currentSession, isLoading]);

  const stopGeneration = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
  }, []);

  const clearChat = useCallback(() => {
    setMessages([]);
    setCurrentSession(null);
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setIsLoading(false);
  }, []);

  return {
    messages,
    currentSession,
    isLoading,
    sendMessage,
    stopGeneration,
    clearChat,
  };
}