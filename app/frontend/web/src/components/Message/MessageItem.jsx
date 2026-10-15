import React from 'react';
import { Streamdown } from 'streamdown';
import './MessageItem.css';

function MessageItem({ message }) {
  if (!message.content) {
    if (!message.isStreaming) return null;
    return (
      <div className={`message message-${message.role}`}>
        <div className="message-content">
          <div className="loading-indicator">
            <div className="loading-dots">
              <div className="loading-dot"></div>
              <div className="loading-dot"></div>
              <div className="loading-dot"></div>
            </div>
          </div>
        </div>
      </div>
    );
  }
  return (
    <div className={`message message-${message.role}`}>
      <div className="message-content">
        <Streamdown key={message.id} controls={{
            table: {
              copy: true,
              download: true,
              fullscreen: false,
            },
          }}>
            {message.content} 
        </Streamdown>
      </div>
    </div>
  );
}

export default MessageItem;