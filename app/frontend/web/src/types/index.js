// Message types
export const ROLE_USER = 'user';
export const ROLE_ASSISTANT = 'assistant';

// Question types for API
export const QUESTION_TYPE_NORMAL = 'normal';
export const QUESTION_TYPE_TOOL_CONFIRM = 'tool_confirm';

// Response types
export const RESPONSE_TYPE_NORMAL = 'normal';
export const RESPONSE_TYPE_TOOL_CONFIRM = 'tool_confirm';

/**
 * @typedef {Object} Message
 * @property {string} id - Unique message ID
 * @property {'user' | 'assistant'} role - Message role
 * @property {string} content - Message content
 * @property {string} timestamp - ISO timestamp
 * @property {boolean} [isStreaming] - Whether message is still streaming
 */

/**
 * @typedef {Object} ChatSession
 * @property {string} id - Session ID
 * @property {string} title - Chat title
 * @property {Message[]} messages - Array of messages
 * @property {string} createdAt - Creation timestamp
 * @property {string} updatedAt - Last update timestamp
 */

/**
 * @typedef {Object} AskRequest
 * @property {string} [sessionID] - Optional session ID
 * @property {string} question - User question
 * @property {string} type - Question type
 * @property {boolean} [stream] - Enable streaming
 * @property {boolean} [enableThinking] - Enable thinking mode
 */

/**
 * @typedef {Object} AskResponse
 * @property {string} type - Response type
 * @property {string} sessionID - Session ID
 * @property {string} answer - Answer content
 * @property {string} [thought] - Thinking content
 * @property {Object} [usage] - Token usage
 */