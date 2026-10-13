import React, { useEffect, useRef } from 'react';
import MessageItem from '../Message/MessageItem';
import './ChatArea.css';

function ToolConfirmCard({ pendingConfirms, onConfirmTool }) {
  if (!pendingConfirms) return null;

  const { confirms = [], message } = pendingConfirms;

  const handleConfirm = (toolCallID, serverName, toolName, answer) => {
    const result = confirms.map(c => ({
      toolCallID: c.toolCallID,
      serverName: c.serverName || '',
      toolName: c.toolName || '',
      answer: c.toolCallID === toolCallID ? answer : c.answer || 'No',
    }));
    onConfirmTool(result);
  };

  return (
    <div className="tool-confirm-card">
      {message && <p className="tool-confirm-message">{message}</p>}
      {confirms.map((confirm) => (
        <div key={confirm.toolCallID} className="tool-confirm-item">
          <div className="tool-confirm-info">
            <span className="tool-confirm-name">
              {confirm.serverName}/{confirm.toolName}
            </span>
            {confirm.arguments && (
              <pre className="tool-confirm-args">
                {typeof confirm.arguments === 'string'
                  ? confirm.arguments
                  : JSON.stringify(confirm.arguments, null, 2)}
              </pre>
            )}
          </div>
          <div className="tool-confirm-actions">
            <button
              className="tool-confirm-yes"
              onClick={() => handleConfirm(confirm.toolCallID, confirm.serverName, confirm.toolName, 'Yes')}
            >
              Yes
            </button>
            <button
              className="tool-confirm-no"
              onClick={() => handleConfirm(confirm.toolCallID, confirm.serverName, confirm.toolName, 'No')}
            >
              No
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

function ChatArea({ messages, isLoading, thought, pendingConfirms, onConfirmTool }) {
  const messagesEndRef = useRef(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, thought]);

  const lastAssistantMsg = [...messages].reverse().find(m => m.role === 'assistant');
  const hasAssistantContent = lastAssistantMsg && lastAssistantMsg.content;

  if (messages.length === 0) {
    return (
      <div className="welcome-screen">
        <div className="welcome-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M12 2L2 7l10 5 10-5-10-5z"></path>
            <path d="M2 17l10 5 10-5"></path>
            <path d="M2 12l10 5 10-5"></path>
          </svg>
        </div>
        <h2 className="welcome-title">How can I help you today?</h2>
        <p className="welcome-subtitle">Ask me anything, I'm here to help</p>
      </div>
    );
  }

  return (
    <div className="chat-messages">
      {messages.map(message => (
        <MessageItem key={message.id} message={message} />
      ))}

      {thought && !hasAssistantContent && (
        <div className="thought-block">
          <div className="thought-header">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10"></circle>
              <path d="M12 16v-4"></path>
              <path d="M12 8h.01"></path>
            </svg>
            <span>思考中...</span>
          </div>
          <div className="thought-content">{thought}</div>
        </div>
      )}

      {pendingConfirms && (
        <ToolConfirmCard
          pendingConfirms={pendingConfirms}
          onConfirmTool={onConfirmTool}
        />
      )}

      <div ref={messagesEndRef} />
    </div>
  );
}

export default ChatArea;