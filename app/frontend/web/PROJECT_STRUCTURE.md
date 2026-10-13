# React Chat Frontend - Project Structure

## Directory Structure

```
src/
├── api/                    # API service layer
│   └── agent.js           # Agent API calls (ask, cancel, SSE parser)
│
├── components/            # React components
│   ├── Chat/             # Chat-related components
│   │   ├── ChatArea.jsx   # Chat messages display area
│   │   ├── ChatArea.css
│   │   ├── ChatInput.jsx  # Message input component
│   │   └── ChatInput.css
│   ├── Message/          # Message components
│   │   ├── MessageItem.jsx # Single message renderer
│   │   └── MessageItem.css
│   └── Sidebar/          # Sidebar components
│       ├── Sidebar.jsx    # Chat history sidebar
│       └── Sidebar.css
│
├── hooks/                # Custom React hooks
│   ├── useChat.js        # Chat logic (send, stop, streaming)
│   └── useChatHistory.js # Chat history management (localStorage)
│
├── styles/               # Global styles
│   └── markdown.css      # Markdown rendering styles
│
├── types/                # Type definitions (JSDoc)
│   └── index.js
│
├── utils/                # Utility functions
│   └── markdown.js       # Markdown rendering with highlight.js
│
├── App.jsx               # Main application component
├── App.css               # Global app styles
└── main.jsx              # Application entry point
```

## Key Features

### API Layer (`src/api/agent.js`)
- `askAgent()` - Send question to agent backend
- `cancelSession()` - Cancel running session
- `parseSSEStream()` - Async generator for parsing SSE streams

### Custom Hooks
- **useChat** - Manages chat state, message streaming, and API calls
- **useChatHistory** - Persists chat sessions to localStorage

### Components
- **Sidebar** - Chat history list with session management
- **ChatArea** - Message display with auto-scroll
- **ChatInput** - Text input with send/stop buttons
- **MessageItem** - Individual message with markdown rendering

## Tech Stack
- React 19
- Vite 8
- marked (markdown parsing)
- highlight.js (code syntax highlighting)

## API Integration
The frontend connects to the Go backend via:
- `POST /v1/agent/ask` - Send questions (SSE streaming)
- `POST /v1/agent/cancel` - Cancel session

## Development
```bash
npm run dev      # Start dev server
npm run build    # Build for production
npm run preview  # Preview production build
```

## Environment Variables
Create `.env` file:
```
VITE_API_BASE_URL=http://localhost:8080
```

The Vite dev server proxies `/v1` requests to the backend automatically.