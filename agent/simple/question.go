package simple

import (
	"github.com/David3310273/go-agent/core"
)

// moved here from app/services/agent.go.
// app/services imports agent/simple, so agent/simple can never import app/services back.
// Sub-sessions are driven from this package and need a concrete core.Question, which is why
// the implementation lives here now.

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

var _ core.Question = (*SimpleQuestion)(nil)

// NewSimpleQuestion builds a SimpleQuestion.
// the fields are unexported, so packages outside simple need a constructor.
func NewSimpleQuestion(
	query string,
	sessionID string,
	model string,
	responseChan chan core.Answer,
	hintChan chan core.Answer,
	streaming bool,
	enableThinking bool,
	questionType core.QuestionType,
) *SimpleQuestion {
	return &SimpleQuestion{
		query:          query,
		sessionID:      sessionID,
		model:          model,
		responseChan:   responseChan,
		hintChan:       hintChan,
		streaming:      streaming,
		enableThinking: enableThinking,
		questionType:   questionType,
	}
}

func (q *SimpleQuestion) GetID() string                     { return q.sessionID }
func (q *SimpleQuestion) GetModelName() string              { return q.model }
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
