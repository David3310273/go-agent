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
		// auto-add: SimpleQuestion now lives in agent/simple, so build it via its constructor
		question = simple.NewSimpleQuestion(
			askParams.Question,
			askParams.SessionID,
			askParams.Model,
			responseChan,
			hintChan,
			askParams.Stream,
			askParams.EnableThinking,
			askParams.Type,
		)
	} else if toolConfirmParams, ok := params.(*ToolConfirmAskParams); ok {
		question = &SimpleToolConfirmQuestion{
			SimpleQuestion: *simple.NewSimpleQuestion(
				toolConfirmParams.Question,
				toolConfirmParams.SessionID,
				toolConfirmParams.Model,
				responseChan,
				hintChan,
				toolConfirmParams.Stream,
				toolConfirmParams.EnableThinking,
				core.QuestionTypeToolConfirm,
			),
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

// SimpleToolConfirmQuestion represents a user's confirmation for a destructive tool
// auto-add: SimpleQuestion moved to agent/simple, so it is embedded from there now
type SimpleToolConfirmQuestion struct {
	simple.SimpleQuestion
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
