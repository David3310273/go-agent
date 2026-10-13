import React from 'react';
import './ChatInput.css';

function ChatInput({ onSend, onStop, isLoading, disabled }) {
  const [input, setInput] = React.useState('');
  const textareaRef = React.useRef(null);

  const handleSubmit = () => {
    if (input.trim() && !isLoading && !disabled) {
      onSend(input.trim());
      setInput('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleChange = (e) => {
    setInput(e.target.value);
    autoResize();
  };

  const autoResize = () => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
    }
  };

  return (
    <div className="chat-input-container">
      <div className="chat-input-wrapper">
        <textarea
          ref={textareaRef}
          className="chat-input"
          value={input}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder="Message AI Agent..."
          rows={1}
          disabled={disabled}
        />
        {!isLoading ? (
          <button
            className="send-btn"
            onClick={handleSubmit}
            disabled={!input.trim() || disabled}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="22" y1="2" x2="11" y2="13"></line>
              <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
            </svg>
          </button>
        ) : (
          <button className="stop-btn" onClick={onStop}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <rect x="6" y="6" width="12" height="12" rx="2"></rect>
            </svg>
          </button>
        )}
      </div>
      <div className="input-hint">
        AI Agent can make mistakes. Consider checking important information.
      </div>
    </div>
  );
}

export default ChatInput;