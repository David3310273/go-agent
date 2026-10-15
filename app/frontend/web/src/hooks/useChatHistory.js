import { useState, useCallback, useEffect } from 'react';
import { listSessions, getSessionHistory, deleteSession } from '../api/agent';

/**
 * Parse a JSONL (JSON Lines) string into an array of objects.
 * Each non-empty line is parsed as JSON; invalid lines are skipped.
 *
 * @param {string} text - JSONL format string
 * @returns {Array<Object>}
 */
function parseJSONL(text) {
  if (!text || typeof text !== 'string') return [];
  const lines = text.split('\n');
  const results = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === '') continue;
    try {
      results.push(JSON.parse(trimmed));
    } catch {
      // skip malformed lines
    }
  }
  return results;
}

/**
 * Format a Unix timestamp (milliseconds) to "YYYY-MM-DD HH:MM:SS".
 *
 * @param {number} ts - Unix timestamp in milliseconds
 * @returns {string}
 */
function formatTimestamp(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export function useChatHistory() {
  const [sessions, setSessions] = useState([]);
  const [activeSessionId, setActiveSessionId] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchSessions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listSessions();
      const sessionList = (data.data?.sessionsInfo || []).map(s => ({
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
      const response = await getSessionHistory(sessionId);
      const jsonlStr = response.data || '';
      const parsed = parseJSONL(jsonlStr);
      const messages = parsed.map((msg, index) => {
        const role = msg.role || 'user';
        const message = {
          id: `msg-${sessionId}-${index}`,
          role,
          content: msg.content || '',
        };
        if (role === 'user' && msg.timestamp) {
          message.formattedTime = formatTimestamp(msg.timestamp);
        }
        return message;
      });
      return {
        id: sessionId,
        messages,
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