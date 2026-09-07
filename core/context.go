package core

type Context interface {
	// set logger
	SetLogger(AgentConfig) *Diagnostic
	// set language type for answer
	SetLanguage(LanguageType)
	// load resources from history
	SetHistory(HistoryConfig) *Diagnostic
	GetHistory() []byte
	// load config
	LoadConfigs(path string) (AgentCoreConfig, *Diagnostic)
	// load prompt
	SetPrompt(PromptConfig) *Diagnostic
	GetPrompt() []byte
	// load knowledge base
	SetKnowledgeBase(KnowledgeBaseConfig) *Diagnostic
	GetKnowledgeBase() []byte
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
