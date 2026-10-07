package controller

import (
	"flag"
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
)

func parseAskCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	// format: /ask [your question] [--enableThinking] [--stream] [--model model_name] [--questionType type]
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
	stream := fs.Bool("stream", false, "enable stream mode")
	modelName := fs.String("model", "", "model name")
	questionType := fs.String("questionType", "normal", "question type")
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

	questionParts := fs.Args()

	// optional --query param
	var question string
	if *query != "" {
		question = *query
	} else if len(questionParts) > 0 {
		question = strings.Join(questionParts, " ")
	} else {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "invalid command format: /ask [your question] [--enableThinking] [--stream] [--model model_name] [--questionType type]",
			},
		}
	}

	cmd := &model.AskQuestionCommand{
		Query:          question,
		EnableThinking: *enableThinking,
		Stream:         *stream,
		Model:          *modelName,
		QuestionType:   core.QuestionType(*questionType),
	}

	return cmd, nil
}
