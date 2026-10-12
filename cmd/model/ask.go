package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
)

var _ core.Command = (*AskQuestionCommand)(nil)

type AskQuestionCommand struct {
	Query          string
	EnableThinking bool
	Stream         bool
	QuestionType   core.QuestionType
	Model          string
	Base           string
	ResponseChan   chan core.Answer
	HintChan       chan core.Answer
	SessionID      string
}

const (
	AskCommandName = "/ask"
)

func init() {
	RegisterCommand(AskCommandName, func() core.Command {
		return &AskQuestionCommand{}
	})
}

func (c AskQuestionCommand) GetHelp() map[string]any {
	return map[string]any{
		"format": "/ask <your question> [--enableThinking]",
		"desc":   "ask the agent a question",
		"options": map[string]string{
			"enableThinking": "enable thinking mode if set to true",
		},
	}
}

func (c *AskQuestionCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "enableThinking":
		c.EnableThinking = value == "true"
	case "sessionID":
		c.SessionID = value
	default:
		core.LogStd(core.LogLevelWarn, "invalid command option %s=%v", key, value)
		return c
	}

	return c
}

func (c AskQuestionCommand) GetBase() string {
	return AskCommandName
}

func (c *AskQuestionCommand) GetID() string                     { return c.SessionID }
func (c *AskQuestionCommand) GetModelName() string              { return c.Model }
func (c *AskQuestionCommand) GetStreaming() bool                { return c.Stream }
func (c *AskQuestionCommand) GetEnableThinking() bool           { return c.EnableThinking }
func (c *AskQuestionCommand) GetQuery() string                  { return c.Query }
func (c *AskQuestionCommand) SetQuery(query string)             { c.Query = query }
func (c *AskQuestionCommand) GetRetryQuery() string             { return c.Query }
func (c *AskQuestionCommand) GetSessionID() string              { return c.SessionID }
func (c *AskQuestionCommand) GetResponseChan() chan core.Answer { return c.ResponseChan }
func (c *AskQuestionCommand) GetHintChan() chan core.Answer     { return c.HintChan }
func (c *AskQuestionCommand) GetType() core.QuestionType        { return c.QuestionType }
func (c *AskQuestionCommand) ToString() string {
	parts := []string{AskCommandName, c.Query}
	if c.EnableThinking {
		parts = append(parts, "--enableThinking")
	}
	return strings.Join(parts, " ")
}

func (c *AskQuestionCommand) Run(app core.CommandApp) *core.Diagnostic {
	query := strings.TrimSpace(c.Query)
	if query == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidQuestion,
			Message: "question is empty",
		}
	}

	// initialize output channels
	c.ResponseChan = make(chan core.Answer, 64)
	c.HintChan = make(chan core.Answer, 64)

	// send question to agent with timeout
	go func() {
		agent := *app.GetAgentCore()
		if agent == nil {
			core.LogStd(core.LogLevelError, "agent not initialized")
			if c.ResponseChan != nil {
				c.ResponseChan <- simple.SimpleNormalResponse{
					SessionID: c.SessionID,
					Response:  core.AgentResponse{Response: "Agent not initialized"},
				}
			}
			return
		}

		select {
		case agent.GetQuestionChan() <- c:
		case <-time.After(time.Duration(app.GetConfig().MaxWaitingRequest) * time.Second):
			core.LogStd(core.LogLevelError, "send question to agent timeout")
			if c.ResponseChan != nil {
				timeoutMsg := simple.SimpleNormalResponse{
					SessionID: c.SessionID,
					Response: core.AgentResponse{
						Response: fmt.Sprintf("Request timeout: failed to send question to agent within %d seconds", app.GetConfig().MaxWaitingRequest),
					},
				}
				select {
				case c.ResponseChan <- timeoutMsg:
				default:
				}
			}
		}
	}()

	return nil
}

func (c *AskQuestionCommand) GetOutputChans() []core.OutputChannel {
	return []core.OutputChannel{
		{Chan: c.ResponseChan, Type: "response"},
		{Chan: c.HintChan, Type: "hint"},
	}
}
