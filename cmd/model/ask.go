package model

import (
	"fmt"
	"io"
	"strings"
	"time"

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
	responseStdout io.Writer
	hintStdout     io.Writer
}

type AskResult struct {
	ResponseChan chan core.Answer
	HintChan     chan core.Answer
	Question     core.Question
}

const (
	AskCommandName = "/ask"
)

func (c AskQuestionCommand) GetHelp() map[string]string {
	return map[string]string{
		"format":         "/ask [your question] [--options]",
		"enableThinking": "enable thinking mode if set to true",
		"stream":         "enable stream mode if set to true",
		"questionType":   "question type, normal or tool_confirm, default is normal",
		"model":          "model name you want to use",
		"query":          "your question",
	}
}

func (c AskQuestionCommand) WithStdout(responseStdout io.Writer, hintStdout io.Writer) core.Command {
	c.responseStdout = responseStdout
	c.hintStdout = hintStdout

	return c
}

func (c AskQuestionCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "enableThinking":
		c.EnableThinking = value == "true"
	case "stream":
		c.Stream = value == "true"
	case "questionType":
		c.QuestionType = core.QuestionType(value)
	case "model":
		c.Model = value
	default:
		core.LogStd(core.LogLevelWarn, "invalid command option %s=%v", key, value)
		return c
	}

	return c
}

func (c AskQuestionCommand) GetBase() string {
	return AskCommandName
}

func (c AskQuestionCommand) Run(agent *core.AgentCore, question core.Question) *core.Diagnostic {
	query := strings.TrimSpace(c.Query)
	if query == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeInvalidQuestion,
			Message: "question is empty",
		}
	}

	// send question to agent
	go func() {
		(*agent).GetQuestionChan() <- question
	}()

	for {
		select {
		case response := <-question.GetResponseChan():
			_, err := c.responseStdout.Write([]byte(response.ToString()))
			if err != nil {
				core.LogStd(core.LogLevelError, "write response to stdout failed: %v", err)
			}
			return nil
		case hint := <-question.GetHintChan():
			_, err := c.hintStdout.Write([]byte(hint.ToString()))
			if err != nil {
				core.LogStd(core.LogLevelError, "write hint to stdout failed: %v", err)
			}
			return nil
		case <-time.After(10 * time.Second):
			core.LogStd(core.LogLevelError, "time out on processing question %s", query)
			_, err := c.hintStdout.Write(fmt.Appendf([]byte{}, "time out on processing question: %s", query))
			if err != nil {
				core.LogStd(core.LogLevelError, "write hint to stdout failed: %v", err)
			}
			return nil
		}
	}
}
