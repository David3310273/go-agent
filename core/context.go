package core

import (
	"log"
)

type Context interface {
	// set logger
	SetLogger(AgentConfig) *Diagnostic
	GetLogger() *log.Logger
	// set language type for answer
	SetLanguage(LanguageType)
	// load resources from history
	SetHistory(HistoryConfig) *Diagnostic
	GetHistory() []byte
	// load config
	LoadConfigs() AgentCoreConfig
	// load prompt
	SetPrompt(PromptConfig) *Diagnostic
	GetPrompt() []byte
	// load knowledge base
	SetKnowledgeBase([]KnowledgeBaseConfig) *Diagnostic
	// changed to []any to support multiple entity types via generics.
	GetKnowledgeBase() []KnowledgeBase[any]
	// load skills
	SetSkills(SkillConfig) *Diagnostic
	GetSkills() []byte

	// get session config
	GetSessionConfig() SessionConfig

	// get model providers
	GetModelProviders() []Provider
	// set initialized model providers
	SetProviders([]Provider) *Diagnostic

	// load tool config into agent context
	SetToolsConfig([]ToolConfig) *Diagnostic
	// get available tools
	GetToolsConfig() []ToolConfig
}

type SessionContext interface {
	// generate final context given question
	GenerateFinalContext(query Question) string
}
