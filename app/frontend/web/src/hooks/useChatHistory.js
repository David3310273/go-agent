import { useState, useCallback, useEffect } from 'react';
import { listSessions, getSessionHistory, deleteSession } from '../api/agent';

export function useChatHistory() {
  const [sessions, setSessions] = useState([]);
  const [activeSessionId, setActiveSessionId] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchSessions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listSessions();
      const sessionList = (data.sessionsInfo || []).map(s => ({
        id: s.sessionID,
        title: s.title || 'Untitled',
      }));
      setSessions(sessionList);
    } catch (error) {
      console.error('Failed to fetch sessions:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  const createSession = useCallback((title, sessionId) => {
    const id = sessionId || crypto.randomUUID();
    const newSession = {
      id: id,
      title: title.substring(0, 50) + (title.length > 50 ? '...' : ''),
    };
    setSessions(prev => [newSession, ...prev]);
    setActiveSessionId(newSession.id);
    return newSession;
  }, []);

  const updateSession = useCallback((sessionId, messages) => {
    setSessions(prev =>
      prev.map(session =>
        session.id === sessionId ? { ...session } : session
      )
    );
  }, []);

  const removeSession = useCallback(async (sessionId) => {
    try {
      await deleteSession(sessionId);
    } catch (error) {
      console.error('Failed to delete session on server:', error);
    }
    setSessions(prev => prev.filter(s => s.id !== sessionId));
    if (activeSessionId === sessionId) {
      setActiveSessionId(null);
    }
  }, [activeSessionId]);

  const getSession = useCallback(async (sessionId) => {
    try {
      const data = await getSessionHistory(sessionId);
      return {
        id: sessionId,
        messages: (data.messages || []).map((msg, index) => ({
          id: `msg-${sessionId}-${index}`,
          role: msg.role,
          content: msg.content || '',
          timestamp: new Date().toISOString(),
        })),
      };
    } catch (error) {
      console.error('Failed to get session history:', error);
      return null;
    }
  }, []);

  return {
    sessions,
    activeSessionId,
    setActiveSessionId,
    createSession,
    updateSession,
    removeSession,
    getSession,
    refreshSessions: fetchSessions,
    loading,
  };
}