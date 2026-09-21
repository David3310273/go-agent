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
	Type           core.QuestionType
}

type ToolConfirmAskParams struct {
	AskParams
	ToolName   string
	ServerName string
}

// AskResult contains the channels for receiving answers
type AskResult struct {
	ResponseChan chan core.Answer
	HintChan     chan core.Answer
	Question     core.Question
}

// Ask creates a question and sends it to the agent for processing
func Ask(agent *simple.SimpleAgent, appConfig *core.AppConfig, params any) *AskResult {
	responseChan := make(chan core.Answer, 64)
	hintChan := make(chan core.Answer, appConfig.RequestQueueLength)

	var question core.Question
	if askParams, ok := params.(*AskParams); ok {
		question = &SimpleQuestion{
			query:          askParams.Question,
			responseChan:   responseChan,
			hintChan:       hintChan,
			streaming:      askParams.Stream,
			enableThinking: askParams.EnableThinking,
			sessionID:      askParams.SessionID,
			model:          askParams.Model,
			questionType:   askParams.Type,
		}
	} else if toolConfirmParams, ok := params.(*ToolConfirmAskParams); ok {
		question = &SimpleToolConfirmQuestion{
			SimpleQuestion: SimpleQuestion{
				query:          toolConfirmParams.Question,
				responseChan:   responseChan,
				hintChan:       hintChan,
				streaming:      toolConfirmParams.Stream,
				enableThinking: toolConfirmParams.EnableThinking,
				sessionID:      toolConfirmParams.SessionID,
				model:          toolConfirmParams.Model,
				questionType:   core.QuestionTypeToolConfirm,
			},
			ToolName:   toolConfirmParams.ToolName,
			ServerName: toolConfirmParams.ServerName,
		}
	}

	// send question to agent with timeout
	requestWaitingTimeout := time.Duration(appConfig.MaxWaitingRequest) * time.Second
	go func() {
		select {
		case agent.Question <- question:
		case <-time.After(requestWaitingTimeout):
			// non-blocking send, abandon if handler already exited
			// use harness default answer for timeout fallback, wrapped in SimpleNormalResponse
			select {
			case responseChan <- simple.SimpleNormalResponse{
				SessionID: question.GetSessionID(),
				Response:  simple.SimpleHarnessInstance.GetDefaultAnswer(),
			}:
			default:
			}
		}
	}()

	log.Printf("will process question %s in mode: enableThinking=%v, stream=%v", question.GetQuery(), question.GetEnableThinking(), question.GetStreaming())

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
	questionType   core.QuestionType
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

// GetType returns the question type (normal for SimpleQuestion)
func (q *SimpleQuestion) GetType() core.QuestionType { return q.questionType }

// removed GetDefaultAnswer, default answer is now managed by harness
func (q *SimpleQuestion) ToString() string { return q.query }

// SimpleToolConfirmQuestion represents a user's confirmation for a destructive tool
type SimpleToolConfirmQuestion struct {
	SimpleQuestion
	ToolName   string `json:"toolName"`   // outer tool name
	ServerName string `json:"serverName"` // MCP server name
}

var _ core.Question = (*SimpleToolConfirmQuestion)(nil)
var _ core.ToolConfirmable = (*SimpleToolConfirmQuestion)(nil)

func (q *SimpleToolConfirmQuestion) GetType() core.QuestionType      { return core.QuestionTypeToolConfirm }
func (q *SimpleToolConfirmQuestion) GetConfirmAnswer() string        { return q.GetQuery() }
func (q *SimpleToolConfirmQuestion) GetConfirmToolName() string      { return q.ToolName }
func (q *SimpleToolConfirmQuestion) GetConfirmMCPServerName() string { return q.ServerName }
func (q *SimpleToolConfirmQuestion) ValiateConfirmAnswer() bool {
	return (q.GetConfirmAnswer() == "Yes" || q.GetConfirmAnswer() == "No") && q.GetConfirmToolName() != ""
}

// SimpleAnswer implements core.Answer interface
type SimpleAnswer struct {
	Content   string `json:"content"`
	SessionID string `json:"sessionID"`
}

func (s *SimpleAnswer) ToString() string     { return s.Content }
func (s *SimpleAnswer) GetSessionID() string { return s.SessionID }
