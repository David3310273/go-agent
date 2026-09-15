package core

type Context interface {
	// load resources from history
	SetHistory(HistoryConfig) *Diagnostic
	GetHistory() []byte
	// load prompt
	SetPrompt(PromptConfig) *Diagnostic
	GetPrompt() []byte
	// load knowledge base
	SetKnowledgeBase([]KnowledgeBaseConfig) *Diagnostic
	// returns knowledge bases
	GetKnowledgeBase() []KnowledgeBase[any]
	// load skills
	SetSkills(SkillConfig) *Diagnostic
	GetSkills() []byte

	// get model providers
	GetModelProviders() []Provider
	// set initialized model providers
	SetProviders([]Provider) *Diagnostic

	// load tool config into agent context
	SetToolsConfig([]ToolConfig) *Diagnostic
	// get available tools
	GetToolsConfig() []ToolConfig
}
