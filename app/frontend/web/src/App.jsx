import React, { useState } from 'react';
import Sidebar from './components/Sidebar/Sidebar';
import ChatArea from './components/Chat/ChatArea';
import ChatInput from './components/Chat/ChatInput';
import { useChat } from './hooks/useChat';
import { useChatHistory } from './hooks/useChatHistory';
import './App.css';

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const { messages, isLoading, sendMessage, stopGeneration, clearChat } = useChat();
  const { sessions, activeSessionId, setActiveSessionId, createSession, updateSession, getSession } = useChatHistory();

  const handleSend = (question) => {
    if (!activeSessionId) {
      const newSession = createSession(question, [], null);
      sendMessage(question);
    } else {
      sendMessage(question);
    }
  };

  const handleNewChat = () => {
    clearChat();
  };

  const handleSelectSession = (sessionId) => {
    const session = getSession(sessionId);
    if (session) {
      setActiveSessionId(sessionId);
    }
  };

  return (
    <div className="app-container">
      <Sidebar
        sessions={sessions}
        activeSessionId={activeSessionId}
        onSelectSession={handleSelectSession}
        onNewChat={handleNewChat}
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />
      <main className="main-content">
        <header className="chat-header">
          <button className="menu-toggle" onClick={() => setSidebarOpen(true)}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="3" y1="6" x2="21" y2="6"></line>
              <line x1="3" y1="12" x2="21" y2="12"></line>
              <line x1="3" y1="18" x2="21" y2="18"></line>
            </svg>
          </button>
          <h1 className="chat-title">AI Agent</h1>
        </header>

        <ChatArea messages={messages} isLoading={isLoading} />

        <ChatInput
          onSend={handleSend}
          onStop={stopGeneration}
          isLoading={isLoading}
        />
      </main>
    </div>
  );
}

export default App;