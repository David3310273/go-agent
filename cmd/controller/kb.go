package controller

import (
	"flag"
	"fmt"

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
	collection := fs.String("collection", "", "collection name")

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
	var filePath string

	switch subCommand {
	case model.KBSubCommandAdd:
		if len(args) < 1 {
			return nil, []core.Diagnostic{
				{
					Level:   core.SeverityError,
					Code:    core.MessageCodeInvalidCommand,
					Message: "file path is required for /kb add",
				},
			}
		}
		filePath = args[0]
	case model.KBSubCommandLs:
		// no file path required, type and domain are optional filters
	case model.KBSubCommandRm:
		if *domain == "" {
			if len(args) < 1 {
				return nil, []core.Diagnostic{
					{
						Level:   core.SeverityError,
						Code:    core.MessageCodeInvalidCommand,
						Message: "file path is required for /kb rm, or use --domain to delete all docs in a domain",
					},
				}
			}
			filePath = args[0]
		} else {
			// --domain specified: delete all docs in that domain, file path optional
			if len(args) > 0 {
				filePath = args[0]
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
		SubCommand: subCommand,
		FilePath:   filePath,
		Base:       model.KBCommandName,
		Type:       kb.KbStorageType(*kbType),
		Domain:     *domain,
		Name:       *name,
		Collection: *collection,
	}

	return cmd, nil
}
