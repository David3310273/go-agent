import { useState, useCallback, useEffect } from 'react';

const STORAGE_KEY = 'chatSessions';

export function useChatHistory() {
  const [sessions, setSessions] = useState([]);
  const [activeSessionId, setActiveSessionId] = useState(null);

  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored);
        setSessions(parsed);
      }
    } catch (error) {
      console.error('Failed to load chat history:', error);
    }
  }, []);

  const saveSessions = useCallback((newSessions) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(newSessions));
      setSessions(newSessions);
    } catch (error) {
      console.error('Failed to save chat history:', error);
    }
  }, []);

  const createSession = useCallback((title, messages, sessionId) => {
    const newSession = {
      id: sessionId || `session-${Date.now()}`,
      title: title.substring(0, 50) + (title.length > 50 ? '...' : ''),
      messages,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    const newSessions = [newSession, ...sessions];
    saveSessions(newSessions);
    setActiveSessionId(newSession.id);
    return newSession;
  }, [sessions, saveSessions]);

  const updateSession = useCallback((sessionId, messages) => {
    const newSessions = sessions.map(session =>
      session.id === sessionId
        ? { ...session, messages, updatedAt: new Date().toISOString() }
        : session
    );
    saveSessions(newSessions);
  }, [sessions, saveSessions]);

  const deleteSession = useCallback((sessionId) => {
    const newSessions = sessions.filter(s => s.id !== sessionId);
    saveSessions(newSessions);
    if (activeSessionId === sessionId) {
      setActiveSessionId(null);
    }
  }, [sessions, activeSessionId, saveSessions]);

  const getSession = useCallback((sessionId) => {
    return sessions.find(s => s.id === sessionId);
  }, [sessions]);

  return {
    sessions,
    activeSessionId,
    setActiveSessionId,
    createSession,
    updateSession,
    deleteSession,
    getSession,
  };
}