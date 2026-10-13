import React from 'react';
import './Sidebar.css';

function Sidebar({ sessions, activeSessionId, onSelectSession, onDeleteSession, onNewChat, isOpen, onClose }) {
  const handleDelete = (e, sessionId) => {
    e.stopPropagation();
    onDeleteSession(sessionId);
  };

  return (
    <aside className={`sidebar ${isOpen ? 'open' : ''}`}>
      <div className="sidebar-header">
        <button className="new-chat-btn" onClick={() => {
          onNewChat();
          onClose();
        }}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span>New Chat</span>
        </button>
      </div>
      <div className="chat-history">
        {sessions.map(session => (
          <div
            key={session.id}
            className={`chat-history-item ${session.id === activeSessionId ? 'active' : ''}`}
            onClick={() => {
              onSelectSession(session.id);
              onClose();
            }}
          >
            <span className="chat-history-title">{session.title}</span>
            <button
              className="chat-history-delete"
              onClick={(e) => handleDelete(e, session.id)}
              title="Delete session"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        ))}
        {sessions.length === 0 && (
          <div className="chat-history-empty">No sessions yet</div>
        )}
      </div>
      <div className="sidebar-footer">
        <div className="user-info">
          <div className="user-avatar">U</div>
          <span className="user-name">User</span>
        </div>
      </div>
    </aside>
  );
}

export default Sidebar;