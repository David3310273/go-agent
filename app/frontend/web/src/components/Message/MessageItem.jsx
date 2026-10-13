import React, { useMemo } from 'react';
import { renderMarkdown } from '../../utils/markdown';
import '../../styles/markdown.css';
import './MessageItem.css';

function MessageItem({ message }) {
  const html = useMemo(() => {
    if (!message.content) return '';
    return renderMarkdown(message.content);
  }, [message.content]);

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
        <div className="message-text markdown-content" dangerouslySetInnerHTML={{ __html: html }} />
      </div>
    </div>
  );
}

export default MessageItem;