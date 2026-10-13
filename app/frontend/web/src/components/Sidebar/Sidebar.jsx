import React from 'react';
import './Sidebar.css';

function Sidebar({ sessions, activeSessionId, onSelectSession, onNewChat, isOpen, onClose }) {
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
            {session.title}
          </div>
        ))}
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