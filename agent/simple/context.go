package simple

import (
	"fmt"
	"os"
	"path"

	_ "github.com/David3310273/go-agent/agent/simple/tools"
	"github.com/David3310273/go-agent/core"
)

// SimpleAgentContext implements core.Context interface for SimpleAgent
type SimpleAgentContext struct {
	// language
	Language core.LanguageType
	// root path
	RootPath string
	// prompt
	Prompt []byte
	// knowledge base
	// changed to []any to support multiple entity types via generics.
	KnowledgeBase []core.KnowledgeBase[any]
	// skill definitions loaded from config
	Skills []core.Skill
	// agent history
	History []byte
	// model providers, for simple agent, only one provider without model routing
	ModelProviders []core.Provider
	// support tools
	Tools []core.Tool
	// mcp server clients, server name -> client
	MCPClients map[string]core.MCPAccessible
}

type SimpleSkill struct {
	RootPath    string
	Name        string
	Description string
	Tools       []core.Tool
}

func (s SimpleSkill) GetDescription() string {
	return s.Description
}

func (s SimpleSkill) GetTools() []core.Tool {
	return s.Tools
}

func (s SimpleSkill) GetName() string {
	return s.Name
}

// interface assertion
var _ core.Context = (*SimpleAgentContext)(nil)

// Context getter/setter methods

func (c *SimpleAgentContext) GetHistory() []byte {
	return c.History
}

func (c *SimpleAgentContext) GetRootPath() string {
	return c.RootPath
}

func (a *SimpleAgentContext) SetHistory(history core.HistoryConfig) *core.Diagnostic {
	a.History = make([]byte, history.BufferSize)
	return nil
}

func (c *SimpleAgentContext) SetLanguage(language core.LanguageType) {
	c.Language = language
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
		tempPrompt, err := os.ReadFile(realPath)
		if err != nil {
			core.LogStd(core.LogLevelWarn, "failed to load prompt: path=%s", realPath)
		} else {
			hasAdded := SimpleHarnessInstance.AddPrompt(&a.Prompt, tempPrompt, prompt.BufferSizeInKB*1024)
			if !hasAdded {
				core.LogStd(core.LogLevelWarn, "prompt buffer full, truncating: path=%s", realPath)
				break
			}
		}
	}

	return nil
}

// return type changed to []any to match KnowledgeBase field.
func (c *SimpleAgentContext) GetKnowledgeBase() *[]core.KnowledgeBase[any] {
	return &c.KnowledgeBase
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
func (a *SimpleAgentContext) AddSkill(skill core.SkillDefinition) *core.Diagnostic {
	for _, loadedSkill := range a.Skills {
		if loadedSkill.GetName() == skill.Name {
			return &core.Diagnostic{
				Code:    core.MessageCodeSkillConflict,
				Message: fmt.Sprintf("skill name %s already exists", skill.Name),
			}
		}
	}

	newSkill := SimpleSkill{
		RootPath:    a.RootPath,
		Name:        skill.Name,
		Description: skill.Description,
		Tools:       []core.Tool{},
	}

	for _, toolName := range skill.Tools {
		tool := core.CreateTool(toolName, a)
		if tool == nil {
			return &core.Diagnostic{
				Code:    core.MessageCodeToolNotFound,
				Message: fmt.Sprintf("tool name %s not found in agentregistry", toolName),
			}
		}

		newSkill.Tools = append(newSkill.Tools, tool)
	}

	a.Skills = append(a.Skills, newSkill)

	return nil
}

// GetSkills returns all skill definitions.
// retrieves all skill definitions for tool loading.
func (a *SimpleAgentContext) GetSkills() *[]core.Skill {
	return &a.Skills
}

// SetToolsConfig loads a list of ToolConfig into the agent's tool list,
// skipping entries with an empty name.
func (a *SimpleAgentContext) AddTool(tool core.Tool) *core.Diagnostic {
	loadedTools := a.Tools
	for _, loadedTool := range loadedTools {
		if tool.GetName() == loadedTool.GetName() {
			return &core.Diagnostic{
				Code:    core.MessageCodeToolConflict,
				Message: fmt.Sprintf("tool name %s already exists", tool.GetName()),
			}
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

// SetMCPClient creates and stores MCP clients for the given configs
func (a *SimpleAgentContext) SetMCPClient(configs []core.MCPConfig) {
	// reset the clients map to avoid stale entries (e.g., after rm)
	a.MCPClients = make(map[string]core.MCPAccessible)

	for _, config := range configs {
		// create new MCP client
		client := &MCPRemoteUtil{
			Config: &config,
		}

		// store the client
		a.MCPClients[config.Name] = client
		core.LogStd(core.LogLevelInfo, "mcp client created: server=%s", config.Name)
	}
}

// GetMCPClients returns all MCP clients
func (c *SimpleAgentContext) GetMCPClients() *map[string]core.MCPAccessible {
	return &c.MCPClients
}

func (c *SimpleAgentContext) GetTools() *[]core.Tool {
	return &c.Tools
}

func (c *SimpleAgentContext) LoadTools(toolConfigs []core.ToolConfig) *core.Diagnostic {
	for _, cfg := range toolConfigs {
		rootPath := cfg.RootPath
		if tool := core.CreateTool(cfg.Name, c); tool != nil {
			c.Tools = append(c.Tools, tool)
		} else {
			core.LogStd(core.LogLevelWarn, "failed to load system tool: name=%s, rootPath=%s", cfg.Name, rootPath)
		}
	}

	return nil
}
