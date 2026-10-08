package ui

import (
	"fmt"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/cmd/config"
	"github.com/David3310273/go-agent/core"
)

type AgentAdapter struct {
	agent  core.AgentCore
	Config config.CommandAppConfig
}

func NewAgentAdapter(agent core.AgentCore, config config.CommandAppConfig) *AgentAdapter {
	return &AgentAdapter{agent: agent, Config: config}
}

func (a *AgentAdapter) SendQuestion(query string, responseChan chan string, hintChan chan string) error {
	question := simple.NewSimpleQuestion(
		query,
		"",
		"",
		make(chan core.Answer, 1),
		make(chan core.Answer, 1),
		true,
		true,
		core.QuestionTypeNormal,
	)

	sendErr := make(chan error, 1)

	// send question to agent
	go func() {
		select {
		case a.agent.GetQuestionChan() <- question:
		case <-time.After(time.Duration(a.Config.MaxWaitingRequest) * time.Second):
			sendErr <- fmt.Errorf("send question %s to agent timeout", query)
		}
	}()

	// read response
	for {
		select {
		case err := <-sendErr:
			return err
		case response := <-question.GetResponseChan():
			responseChan <- response.ToString()
			return nil
		case hint := <-question.GetHintChan():
			hintChan <- hint.ToString()
			return nil
		case <-time.After(time.Duration(a.Config.MaxWaitingRequest) * time.Second):
			return fmt.Errorf("wait response to %s timeout", query)
		}
	}
}

func (a *AgentAdapter) Stop() error {
	errs := core.StopAgentCore(a.agent)
	if len(errs) > 0 {
		return fmt.Errorf("stop agent error: %v", errs)
	}
	return nil
}
