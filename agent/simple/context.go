package simple

import (
	"log"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

// SimpleAgentContext implements core.Context interface for SimpleAgent
type SimpleAgentContext struct {
	// language
	Language core.LanguageType
	// prompt
	Prompt []byte
	// knowledge base
	// changed to []any to support multiple entity types via generics.
	KnowledgeBase []core.KnowledgeBase[any]
	// skill definitions loaded from config
	Skills []core.SkillDefinition
	// agent history
	History []byte
	// model providers, for simple agent, only one provider without model routing
	ModelProviders []core.Provider
	// support tools
	Tools []core.ToolConfig
	// mcp server configs
	MCPServers []core.MCPConfig
	// auto-add: mcp server clients, server name -> client
	MCPClients map[string]core.MCPAccessible
}

// auto-add: interface assertion
var _ core.Context = (*SimpleAgentContext)(nil)

// Context getter/setter methods

func (c *SimpleAgentContext) GetHistory() []byte {
	return c.History
}

func (a *SimpleAgentContext) SetHistory(history core.HistoryConfig) *core.Diagnostic {
	a.History = make([]byte, history.BufferSize)
	return nil
}

func (c *SimpleAgentContext) SetLanguage(language core.LanguageType) {
	c.Language = language
}

func (c *SimpleAgentContext) GetToolsConfig() []core.ToolConfig {
	return c.Tools
}

func (c *SimpleAgentContext) GetMCPServerConfigs() []core.MCPConfig {
	return c.MCPServers
}

func (c *SimpleAgentContext) GetPrompt() []byte {
	return c.Prompt
}

func (a *SimpleAgentContext) SetPrompt(prompt core.PromptConfig) *core.Diagnostic {
	// use capacity instead of length, and convert KB to bytes
	a.Prompt = make([]byte, 0, prompt.BufferSizeInKB*1024)

	for _, filename := range prompt.Paths {
		// use RootPath instead of hardcoded relative path
		realPath := path.Join(prompt.RootPath, SimpleAgentPath, filename)
		log.Printf("real prompt path: %s", realPath)
		tempPrompt, err := os.ReadFile(realPath)
		if err != nil {
			log.Printf("failed to load prompt: %s", realPath)
		} else {
			hasAdded := SimpleHarnessInstance.AddPrompt(&a.Prompt, tempPrompt, prompt.BufferSizeInKB*1024)
			if !hasAdded {
				log.Printf("cannot load whole prompt %s because buffer is full, will truncate in here...", realPath)
				break
			}
		}
	}

	return nil
}

// return type changed to []any to match KnowledgeBase field.
func (c *SimpleAgentContext) GetKnowledgeBase() []core.KnowledgeBase[any] {
	return c.KnowledgeBase
}

// complete SetKnowledgeBase to collect all KB instances via NewSimpleKnowledgeBase.
// rootPath is now set in knowledgeConfig.RootPath before calling.
func (a *SimpleAgentContext) SetKnowledgeBase(knowledgeConfigs []core.KnowledgeBaseConfig) *core.Diagnostic {
	var kbs []core.KnowledgeBase[any]
	for _, knowledgeConfig := range knowledgeConfigs {
		kb := NewSimpleKnowledgeBase(knowledgeConfig)
		kbs = append(kbs, kb)
	}

	a.KnowledgeBase = kbs

	return nil
}

// SetSkills stores skill definitions from config.
// loads skill definitions for dynamic tool loading by UseSkill.
func (a *SimpleAgentContext) SetSkills(skills []core.SkillDefinition) {
	a.Skills = skills
}

// GetSkill returns the skill definition by name.
// retrieves skill definition for dynamic tool loading.
func (a *SimpleAgentContext) GetSkill(name string) *core.SkillDefinition {
	for i := range a.Skills {
		if a.Skills[i].Name == name {
			return &a.Skills[i]
		}
	}
	return nil
}

// GetSkills returns all skill definitions.
// retrieves all skill definitions for tool loading.
func (a *SimpleAgentContext) GetSkills() []core.SkillDefinition {
	return a.Skills
}

// SetToolsConfig loads a list of ToolConfig into the agent's tool list,
// skipping entries with an empty name.
func (a *SimpleAgentContext) SetToolsConfig(tools []core.ToolConfig) *core.Diagnostic {
	for _, tool := range tools {
		if tool.Name == "" {
			continue
		}
		a.Tools = append(a.Tools, tool)
	}
	return nil
}

func (c *SimpleAgentContext) GetModelProviders() []core.Provider {
	return c.ModelProviders
}

// SetProviders caches initialized providers on the agent context
func (a *SimpleAgentContext) SetProviders(providers []core.Provider) *core.Diagnostic {
	a.ModelProviders = providers
	return nil
}

func (c *SimpleAgentContext) GetTools() []core.ToolConfig {
	return c.Tools
}

// auto-add: SetMCPClient creates and stores MCP clients for the given configs
func (a *SimpleAgentContext) SetMCPClient(configs []core.MCPConfig) *core.Diagnostic {
	a.MCPServers = configs

	if a.MCPClients == nil {
		a.MCPClients = make(map[string]core.MCPAccessible)
	}

	for _, config := range configs {
		// create new MCP client (no initialize needed for 2026-07-28 protocol)
		client := &MCPRemoteUtil{
			Config: &config,
		}

		// store the client
		a.MCPClients[config.Name] = client
		log.Printf("MCP client for %s created", config.Name)
	}

	return nil
}

// auto-add: GetMCPClients returns all MCP clients
func (c *SimpleAgentContext) GetMCPClients() map[string]core.MCPAccessible {
	return c.MCPClients
}
