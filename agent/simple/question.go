package simple

import (
	uuid "github.com/gofrs/uuid/v5"

	"github.com/David3310273/go-agent/core"
)

// moved here from app/services/agent.go.
// app/services imports agent/simple, so agent/simple can never import app/services back.
// Sub-sessions are driven from this package and need a concrete core.Question, which is why
// the implementation lives here now.

// SimpleQuestion implements core.Question interface for HTTP requests
type SimpleQuestion struct {
	id             string
	query          string
	sessionID      string
	model          string
	responseChan   chan core.Answer
	hintChan       chan core.Answer
	streaming      bool
	enableThinking bool
	timestamp      int64
	questionType   core.QuestionType
	toolCalls      map[string]any

	// plan overrides injected by middleware – zero means use agent default
	planReActMaxRounds   int
	planMaxToken         uint64
	planAvailableModels  []string
	planMemoryWindowSize uint64
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
	timestamp int64,
) *SimpleQuestion {
	idYielder, _ := uuid.NewV4()
	id := idYielder.String()

	return &SimpleQuestion{
		id:             id,
		query:          query,
		sessionID:      sessionID,
		model:          model,
		responseChan:   responseChan,
		hintChan:       hintChan,
		streaming:      streaming,
		enableThinking: enableThinking,
		questionType:   questionType,
		timestamp:      timestamp,
		toolCalls:      make(map[string]any),
	}
}

func (q *SimpleQuestion) GetID() string                     { return q.id }
func (q *SimpleQuestion) GetModelName() string              { return q.model }
func (q *SimpleQuestion) GetStreaming() bool                { return q.streaming }
func (q *SimpleQuestion) GetEnableThinking() bool           { return q.enableThinking }
func (q *SimpleQuestion) GetQuery() string                  { return q.query }
func (q *SimpleQuestion) SetQuery(query string)             { q.query = query }
func (q *SimpleQuestion) GetSendTimestamp() int64           { return q.timestamp }
func (q *SimpleQuestion) GetSessionID() string              { return q.sessionID }
func (q *SimpleQuestion) GetResponseChan() chan core.Answer { return q.responseChan }
func (q *SimpleQuestion) GetHintChan() chan core.Answer     { return q.hintChan }

func (q *SimpleQuestion) GetPlanReActMaxRounds() int       { return q.planReActMaxRounds }
func (q *SimpleQuestion) GetPlanMaxToken() uint64          { return q.planMaxToken }
func (q *SimpleQuestion) GetPlanAvailableModels() []string { return q.planAvailableModels }
func (q *SimpleQuestion) SetPlanOverrides(plan core.Plan) {
	q.planReActMaxRounds = plan.MaxRoundsReAct
	q.planMaxToken = plan.MaxToken
	q.planAvailableModels = plan.AvailableModels
	q.planMemoryWindowSize = plan.MemoryWindowSize
}

func (q *SimpleQuestion) GetUserPlan() core.Plan {
	return core.Plan{
		MaxRoundsReAct:   q.planReActMaxRounds,
		MaxToken:         q.planMaxToken,
		AvailableModels:  q.planAvailableModels,
		MemoryWindowSize: q.planMemoryWindowSize,
	}
}

func (q *SimpleQuestion) GetToolCalls() map[string]any {
	return q.toolCalls
}

// SetToolCalls sets the tool calls
func (q *SimpleQuestion) SetToolCalls(toolCall core.ToolCall) {
	if q.toolCalls[toolCall.Function.Name] == nil {
		q.toolCalls[toolCall.Function.Name] = make(map[string]int)
	}
	toolCallMap := q.toolCalls[toolCall.Function.Name].(map[string]int)
	toolCallMap[toolCall.Function.Arguments]++
}

// GetType returns the question type (normal for SimpleQuestion)
func (q *SimpleQuestion) GetType() core.QuestionType { return q.questionType }

// removed GetDefaultAnswer, default answer is now managed by harness
func (q *SimpleQuestion) ToString() string { return q.query }
