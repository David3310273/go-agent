package simple

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/core"
)

type SimpleHarness struct {
}

var SimpleHarnessInstance = NewSimpleHarness()

func NewSimpleHarness() *SimpleHarness {
	return &SimpleHarness{}
}

var _ core.Harness = (*SimpleHarness)(nil)

func (h SimpleHarness) AddPrompt(systemPrompt *[]byte, document []byte, maxSize int) bool {
	if len(document) == 0 || len(*systemPrompt)+len(document) > maxSize {
		return false
	}

	*systemPrompt = append(*systemPrompt, document...)

	return true
}

func (h SimpleHarness) AddAgentHistory(agentHistory *[]byte, maxSize int) {
}

func (h SimpleHarness) SetFinalQuery(question *core.Question, knowledge string, splitter string) {
	query := fmt.Sprintf("[Question]\n: %s", (*question).GetQuery())
	kb := fmt.Sprintf("[Knowledge]\n: %s", knowledge)
	finalQuery := fmt.Sprintf("%s%s%s", kb, splitter, query)

	(*question).SetQuery(finalQuery)
}

func (h SimpleHarness) GenerateFinalPrompt(systemPrompt string, agentHistory string, maxSize int) string {
	var prompt strings.Builder

	prompt.WriteString(systemPrompt)
	prompt.WriteString("\n")
	prompt.WriteString(agentHistory)

	return prompt.String()
}

func (h SimpleHarness) SetCurrRoundMessages(messages *core.Conversation, message core.ReActMessage, windowSize int, skip int) {
	// TODO: sliding window started from first user message
	*messages = append(*messages, message)
}

func (h SimpleHarness) GetNextRoundTools(skillName string) []core.Tool {
	return nil
}

func (h SimpleHarness) GetCurrRoundKnowledges(question core.Question) string {
	return ""
}

func (h SimpleHarness) SetNextRoundMessages(question core.Question, messages *core.Conversation) {
}
