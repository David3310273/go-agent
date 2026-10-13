import React from 'react';
import { renderMarkdown } from '../../utils/markdown';
import '../../styles/markdown.css';
import './MessageItem.css';

function MessageItem({ message }) {
  const renderContent = (content) => {
    if (!content) {
      return (
        <div className="loading-indicator">
          <div className="loading-dots">
            <div className="loading-dot"></div>
            <div className="loading-dot"></div>
            <div className="loading-dot"></div>
          </div>
        </div>
      );
    }

    const html = renderMarkdown(content);
    return <div dangerouslySetInnerHTML={{ __html: html }} />;
  };

  return (
    <div className={`message message-${message.role}`}>
      <div className="message-content">
        <div className="message-text markdown-content">
          {renderContent(message.content)}
        </div>
      </div>
    </div>
  );
}

export default MessageItem;