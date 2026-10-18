package core

// context only includes runtime hot data, for static configs, use AgentCoreConfig.
type Context interface {
	// get project root path
	GetRootPath() string
	// load resources from history
	SetHistory(HistoryConfig) *Diagnostic
	GetHistory() []byte
	// load prompt
	SetPrompt(PromptConfig) *Diagnostic
	GetPrompt() []byte
	// load knowledge base
	SetKnowledgeBase([]KnowledgeBaseConfig) *Diagnostic
	// returns knowledge bases
	GetKnowledgeBase() *[]KnowledgeBase[any]
	// get model providers
	GetModelProviders() []Provider
	// set initialized model providers
	SetProviders([]Provider) *Diagnostic
	// get mcp server tools
	SetMCPClient([]MCPConfig)
	// get mcp client
	GetMCPClients() *map[string]MCPAccessible
	// only can get skills
	GetSkills() *[]Skill
	AddSkill(skill SkillDefinition) *Diagnostic
	// get tools
	GetTools() *[]Tool
	AddTool(tool Tool) *Diagnostic

	// load tools
	LoadTools([]ToolConfig) *Diagnostic
}
