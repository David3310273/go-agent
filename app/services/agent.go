package services

import (
	"log"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
)

// AskParams represents the parameters for asking a question
type AskParams struct {
	Question       string
	SessionID      string
	Model          string
	Stream         bool
	EnableThinking bool
}

// AskResult contains the channels for receiving answers
type AskResult struct {
	ResponseChan chan core.Answer
	HintChan     chan core.Answer
	Question     *SimpleQuestion
}

// Ask creates a question and sends it to the agent for processing
func Ask(agent *simple.SimpleAgent, appConfig *core.AppConfig, params *AskParams) *AskResult {
	responseChan := make(chan core.Answer, 64)
	hintChan := make(chan core.Answer, appConfig.RequestQueueLength)

	question := &SimpleQuestion{
		query:          params.Question,
		responseChan:   responseChan,
		hintChan:       hintChan,
		streaming:      params.Stream,
		enableThinking: params.EnableThinking,
		sessionID:      params.SessionID,
		model:          params.Model,
	}

	// send question to agent with timeout
	requestWaitingTimeout := time.Duration(appConfig.MaxWaitingRequest) * time.Second
	go func() {
		select {
		case agent.Question <- question:
		case <-time.After(requestWaitingTimeout):
			// non-blocking send, abandon if handler already exited
			// auto-add: use harness default answer for timeout fallback, wrapped in SimpleSessionResponse
			select {
			case responseChan <- simple.SimpleSessionResponse{
				SessionID: question.GetSessionID(),
				Response:  core.AgentResponse{Response: simple.SimpleHarnessInstance.GetDefaultAnswer().ToString()},
			}:
			default:
			}
		}
	}()

	log.Printf("will process question %s in mode: enableThinking=%v, stream=%v", question.query, question.enableThinking, question.streaming)

	return &AskResult{
		ResponseChan: responseChan,
		HintChan:     hintChan,
		Question:     question,
	}
}

// SimpleQuestion implements core.Question interface for HTTP requests
type SimpleQuestion struct {
	query          string
	sessionID      string
	model          string
	responseChan   chan core.Answer
	hintChan       chan core.Answer
	streaming      bool
	enableThinking bool
}

func (q *SimpleQuestion) GetID() string                     { return q.sessionID }
func (q *SimpleQuestion) GetProviderName() string           { return q.model }
func (q *SimpleQuestion) GetStreaming() bool                { return q.streaming }
func (q *SimpleQuestion) GetEnableThinking() bool           { return q.enableThinking }
func (q *SimpleQuestion) GetQuery() string                  { return q.query }
func (q *SimpleQuestion) SetQuery(query string)             { q.query = query }
func (q *SimpleQuestion) GetRetryQuery() string             { return q.query }
func (q *SimpleQuestion) GetSessionID() string              { return q.sessionID }
func (q *SimpleQuestion) GetResponseChan() chan core.Answer { return q.responseChan }
func (q *SimpleQuestion) GetHintChan() chan core.Answer     { return q.hintChan }

// auto-add: removed GetDefaultAnswer, default answer is now managed by harness
func (q *SimpleQuestion) ToString() string { return q.query }

// SimpleAnswer implements core.Answer interface
type SimpleAnswer struct {
	Content   string `json:"content"`
	SessionID string `json:"sessionID"`
}

func (s *SimpleAnswer) ToString() string     { return s.Content }
func (s *SimpleAnswer) GetSessionID() string { return s.SessionID }
