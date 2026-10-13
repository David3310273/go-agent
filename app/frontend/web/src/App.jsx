import React, { useState } from 'react';
import Sidebar from './components/Sidebar/Sidebar';
import ChatArea from './components/Chat/ChatArea';
import ChatInput from './components/Chat/ChatInput';
import { useChat } from './hooks/useChat';
import { useChatHistory } from './hooks/useChatHistory';
import './App.css';

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [enableThinking, setEnableThinking] = useState(false);
  const {
    messages,
    currentSession,
    isLoading,
    thought,
    pendingConfirms,
    sendMessage,
    stopGeneration,
    confirmToolCall,
    clearChat,
    restoreSession,
  } = useChat();
  const {
    sessions,
    activeSessionId,
    setActiveSessionId,
    createSession,
    updateSession,
    removeSession,
    getSession,
  } = useChatHistory();

  const handleSend = (question) => {
    if (!activeSessionId) {
      const newSession = createSession(question);
      restoreSession(newSession.id, []);
    }
    sendMessage(question, enableThinking);
  };

  const handleNewChat = () => {
    clearChat();
    setActiveSessionId(crypto.randomUUID());
  };

  const handleSelectSession = async (sessionId) => {
    const session = await getSession(sessionId);
    if (session) {
      setActiveSessionId(sessionId);
      restoreSession(session.id, session.messages);
    }
  };

  const handleDeleteSession = async (sessionId) => {
    await removeSession(sessionId);
    if (activeSessionId === sessionId) {
      clearChat();
    }
  };

  const handleConfirmTool = (confirms) => {
    confirmToolCall(confirms);
  };

  const handleToggleThinking = () => {
    setEnableThinking(prev => !prev);
  };

  return (
    <div className="app-container">
      <Sidebar
        sessions={sessions}
        activeSessionId={activeSessionId}
        onSelectSession={handleSelectSession}
        onDeleteSession={handleDeleteSession}
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

        <ChatArea
          messages={messages}
          isLoading={isLoading}
          thought={enableThinking ? thought : null}
          pendingConfirms={pendingConfirms}
          onConfirmTool={handleConfirmTool}
        />

        <ChatInput
          onSend={handleSend}
          onStop={stopGeneration}
          isLoading={isLoading}
          enableThinking={enableThinking}
          onToggleThinking={handleToggleThinking}
        />
      </main>
    </div>
  );
}

export default App;