package controller

import (
	"flag"
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
	kb "github.com/David3310273/go-agent/knowledgebase"
	"github.com/David3310273/go-agent/knowledgebase/storage"
)

func parseKBCommand(cmdSlices []string) (core.Command, []core.Diagnostic) {
	if len(cmdSlices) < 2 {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: "kb subcommand is required, usage: /kb add|ls|rm [--type storage_type] [--domain domain]",
			},
		}
	}

	subCommand := cmdSlices[1]

	fs := flag.NewFlagSet("kb", flag.ContinueOnError)
	fs.SetOutput(nil)

	kbType := fs.String("type", "milvus", "storage type")
	domain := fs.String("domain", "", "document domain")
	name := fs.String("name", storage.MilvusKnowledgeBaseName, "knowledge base name")
	desc := fs.String("desc", "", "document description")
	collection := fs.String("collection", "", "collection name")
	filePath := fs.String("file", "", "file path")

	err := fs.Parse(cmdSlices[2:])
	if err != nil {
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("failed to parse kb command: %v", err),
			},
		}
	}

	args := fs.Args()
	if len(args) > 0 {
		// for simplicity, first arg must be file path
		if *filePath == "" {
			*filePath = args[0]
		}
		// parse args....
		isDesc := false
		isDomain := false
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--") {
				if arg == "--desc" {
					isDesc = true
					isDomain = false
				} else if arg == "--domain" {
					isDesc = false
					isDomain = true
				}
			} else if isDesc {
				if *desc == "" {
					*desc = arg
				} else {
					*desc += " " + arg
				}
			} else if isDomain {
				if *domain == "" {
					*domain = arg
				} else {
					*domain += " " + arg
				}
			}
		}
	}

	switch subCommand {
	case model.KBSubCommandAdd:
		if *filePath == "" {
			return nil, []core.Diagnostic{
				{
					Level:   core.SeverityError,
					Code:    core.MessageCodeInvalidCommand,
					Message: "file path is required for /kb add",
				},
			}
		}
	case model.KBSubCommandLs:
	case model.KBSubCommandRm:
		if *filePath == "" {
			return nil, []core.Diagnostic{
				{
					Level:   core.SeverityError,
					Code:    core.MessageCodeInvalidCommand,
					Message: "file path is required for /kb add",
				},
			}
		}
	default:
		return nil, []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("unknown kb subcommand: %s, expected add|ls|rm", subCommand),
			},
		}
	}

	cmd := &model.KnowledgeBaseCommand{
		SubCommand:   subCommand,
		FilePath:     *filePath,
		Base:         model.KBCommandName,
		Type:         kb.KbStorageType(*kbType),
		Domain:       *domain,
		Name:         *name,
		Collection:   *collection,
		Description:  *desc,
		ResponseChan: make(chan core.Diagnostic, 1),
	}

	return cmd, nil
}
