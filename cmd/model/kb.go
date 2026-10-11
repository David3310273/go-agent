package model

import (
	"fmt"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
	kb "github.com/David3310273/go-agent/knowledgebase"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
)

var _ core.Command = (*KnowledgeBaseCommand)(nil)

type KnowledgeBaseCommand struct {
	SubCommand   string
	FilePath     string
	Base         string
	Type         kb.KbStorageType
	Domain       string
	Name         string
	Collection   string
	ResponseChan chan core.Diagnostic
}

const (
	KBCommandName = "/kb"

	KBSubCommandAdd = "add"
	KBSubCommandLs  = "ls"
	KBSubCommandRm  = "rm"
)

func init() {
	RegisterCommand(KBCommandName, func() core.Command {
		return &KnowledgeBaseCommand{}
	})
}

func (c KnowledgeBaseCommand) GetHelp() map[string]any {
	return map[string]any{
		"format": "/kb add|ls|rm [file_path] [--type storage_type] [--domain domain]",
		"desc":   "manage knowledge base",
		"options": map[string]string{
			"add":    "add a local file to knowledge base, usage: /kb add <file_path> [--type milvus] [--domain domain]",
			"ls":     "list files in knowledge base, usage: /kb ls [--type milvus] [--domain domain]",
			"rm":     "remove files from knowledge base, usage: /kb rm [file_path] [--type milvus] [--domain domain] (with --domain deletes all docs in domain)",
			"type":   "storage type, default: milvus",
			"domain": "document domain filter",
		},
	}
}

func (c *KnowledgeBaseCommand) WithOption(key string, value string) core.Command {
	switch key {
	case "type":
		c.Type = kb.KbStorageType(value)
	case "domain":
		c.Domain = value
	case "filePath":
		c.FilePath = value
	case "collection":
		c.Collection = value
	default:
		return c
	}
	return c
}

func (c KnowledgeBaseCommand) GetBase() string {
	return KBCommandName
}

func (c *KnowledgeBaseCommand) Run(app core.CommandApp) *core.Diagnostic {
	switch c.SubCommand {
	case KBSubCommandAdd:
		// TODO: implement kb add
	case KBSubCommandLs:
		// /kb ls
		if c.Type != storage.MilvusKnowledgeBaseStorageType {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("type %s is not supported now", c.Type),
			}
		}
		agent := app.GetAgentCore()
		kbs := (*agent).GetKnowledgeBase()
		var kbClient core.KnowledgeBase[any]
		for _, kb := range kbs {
			if kb.GetName() == c.Name {
				kbClient = kb
				break
			}
		}

		filename := ""
		if c.FilePath != "" {
			filename = path.Base(c.FilePath)
		}

		results, diags := kbClient.Query(c.Collection, c.Domain, filename, []string{"filename"}, []string{"filename"})
		if diags != nil {
			return diags
		}

		core.LogStd(core.LogLevelDebug, "kb ls result: %#v", results)

		fileList := []string{}
		for _, result := range results {
			for key, value := range result {
				if key == "filename" {
					fileList = append(fileList, value.(string))
				}
			}
		}
		core.LogStd(core.LogLevelDebug, "kb ls fileList: %#v", fileList)
		return &core.Diagnostic{
			Code:    core.MessageCodeSuccess,
			Message: fmt.Sprintf("list %d files", len(fileList)),
			Data:    fileList,
		}
	case KBSubCommandRm:
		// implement kb rm
		if c.Type != storage.MilvusKnowledgeBaseStorageType {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeInvalidCommand,
				Message: fmt.Sprintf("type %s is not supported now", c.Type),
			}
		}
		agent := app.GetAgentCore()
		kbs := (*agent).GetKnowledgeBase()
		var kbClient core.KnowledgeBase[any]
		for _, kb := range kbs {
			if kb.GetName() == c.Name {
				kbClient = kb
				break
			}
		}
		filename := path.Base(c.FilePath)
		diag := kbClient.Delete(c.Collection, c.Domain, filename)
		if diag == nil {
			diag = &core.Diagnostic{
				Level:   core.SeverityInfo,
				Code:    core.MessageCodeSuccess,
				Message: fmt.Sprintf("remove %s", filename),
			}
		}
		select {
		case c.ResponseChan <- *diag:
		default:
		}
		return nil
	}
	return nil
}

func (c *KnowledgeBaseCommand) GetOutputChans() []core.OutputChannel {
	return []core.OutputChannel{
		{Chan: c.ResponseChan, Type: "response"},
	}
}

func (c *KnowledgeBaseCommand) ToString() string {
	parts := []string{KBCommandName, c.SubCommand}
	if c.FilePath != "" {
		parts = append(parts, c.FilePath)
	}
	return strings.Join(parts, " ")
}
