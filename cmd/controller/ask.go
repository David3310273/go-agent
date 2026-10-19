package controller

import (
	"flag"
	"fmt"
	"strings"

	uuid "github.com/gofrs/uuid/v5"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

func parseAskCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	// format: /ask [your question] [--enableThinking]
	if len(cmdSlices) < 2 {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "question is required for /ask command",
			},
		}
	}

	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	fs.SetOutput(nil)

	enableThinking := fs.Bool("enableThinking", false, "enable thinking mode")
	query := fs.String("query", "", "your question")

	err := fs.Parse(cmdSlices[1:])
	if err != nil {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("failed to parse command: %v", err),
			},
		}
	}

	rawArgs := fs.Args()
	var question string

	if *query != "" {
		question = *query
	}

	isQuestionPart := true
	for idx, arg := range rawArgs {
		if strings.HasPrefix(arg, "--") {
			if idx > 0 {
				isQuestionPart = false
			}
			if arg == "--enableThinking" {
				*enableThinking = true
			}
		} else if isQuestionPart {
			if question != "" {
				question = question + " " + arg
			} else {
				question = arg
			}
		}
	}

	if question == "" {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "invalid command format: /ask <your question> [--enableThinking]",
			},
		}
	}

	idYielder, _ := uuid.NewV4()
	cmd := &model.AskQuestionCommand{
		ID:             idYielder.String(),
		Query:          question,
		Base:           model.AskCommandName,
		EnableThinking: *enableThinking,
		Stream:         true,
		Model:          "",
		QuestionType:   core.QuestionTypeNormal,
	}

	return cmd, nil
}
