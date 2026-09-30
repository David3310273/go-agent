package services

import (
	"fmt"
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

// ToolConfirmAskParams carries a batch of user answers for the pending destructive tool calls.
// replaces the single ToolName/ServerName pair, the whole batch is confirmed in one
// request and every item is answered with its own Yes or No.
type ToolConfirmAskParams struct {
	AskParams
	Confirms []core.ToolConfirmAnswer
}

// AskResult contains the channels for receiving answers
type AskResult struct {
	ResponseChan chan core.Answer
	HintChan     chan core.Answer
	Question     core.Question
}

// Ask creates a question and sends it to the agent for processing
func Ask(agent *simple.SimpleAgent, appConfig *core.AppConfig, params any) (*AskResult, *core.Diagnostic) {
	responseChan := make(chan core.Answer, 64)
	hintChan := make(chan core.Answer, appConfig.RequestQueueLength)

	var question core.Question

	switch askType := params.(type) {
	case *AskParams:
		question = simple.NewSimpleQuestion(
			askType.Question,
			askType.SessionID,
			askType.Model,
			responseChan,
			hintChan,
			askType.Stream,
			askType.EnableThinking,
			askType.Type,
		)
	case *ToolConfirmAskParams:
		question = &SimpleToolConfirmQuestion{
			SimpleQuestion: *simple.NewSimpleQuestion(
				askType.Question,
				askType.SessionID,
				askType.Model,
				responseChan,
				hintChan,
				askType.Stream,
				askType.EnableThinking,
				core.QuestionTypeToolConfirm,
			),
			Confirms: askType.Confirms,
		}
	default:
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: fmt.Sprintf("invalid type for params: %T", params),
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

	core.LogStd(core.LogLevelDebug, "processing question: query_len=%d, enable_thinking=%v, stream=%v",
		len(question.GetQuery()), question.GetEnableThinking(), question.GetStreaming())

	return &AskResult{
		ResponseChan: responseChan,
		HintChan:     hintChan,
		Question:     question,
	}, nil
}

// SimpleToolConfirmQuestion represents the user's answers for a batch of destructive tool calls
// SimpleQuestion moved to agent/simple, so it is embedded from there now
type SimpleToolConfirmQuestion struct {
	simple.SimpleQuestion
	// Confirms carries one answer per pending destructive tool call
	Confirms []core.ToolConfirmAnswer `json:"confirms"`
}

var _ core.Question = (*SimpleToolConfirmQuestion)(nil)
var _ core.ToolConfirmable = (*SimpleToolConfirmQuestion)(nil)

func (q *SimpleToolConfirmQuestion) GetType() core.QuestionType { return core.QuestionTypeToolConfirm }
func (q *SimpleToolConfirmQuestion) GetConfirmAnswers() []core.ToolConfirmAnswer {
	return q.Confirms
}

// ValidateConfirmAnswers reports whether every item of the batch is answerable
func (q *SimpleToolConfirmQuestion) ValidateConfirmAnswers() bool {
	if len(q.Confirms) == 0 {
		return false
	}
	for _, confirm := range q.Confirms {
		if confirm.ToolCallID == "" {
			return false
		}
		if confirm.Answer != "Yes" && confirm.Answer != "No" {
			return false
		}
	}
	return true
}

// SimpleAnswer implements core.Answer interface
type SimpleAnswer struct {
	Content   string `json:"content"`
	SessionID string `json:"sessionID"`
}

func (s *SimpleAnswer) ToString() string     { return s.Content }
func (s *SimpleAnswer) GetSessionID() string { return s.SessionID }
