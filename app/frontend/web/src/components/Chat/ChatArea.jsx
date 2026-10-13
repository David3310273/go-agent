import React, { useEffect, useRef } from 'react';
import MessageItem from '../Message/MessageItem';
import './ChatArea.css';

function ChatArea({ messages, isLoading }) {
  const messagesEndRef = useRef(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

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
      {isLoading && (
        <MessageItem
          message={{
            id: 'loading',
            role: 'assistant',
            content: '',
          }}
        />
      )}
      <div ref={messagesEndRef} />
    </div>
  );
}

export default ChatArea;