package core

type LockManager interface {
	Acquire() *Diagnostic
	Release() *Diagnostic
}

type UI interface {
	// show message
	Render(string)
	// show error message
	RenderSystemMessage([]Diagnostic)
	// display welcome message
	Welcome()
	// get user input
	GetUserInput() string
}

type Configurable interface {
	GetConfigPath() string
}

type WorkFlow interface {
	// before start
	BeforeStart(AgentCoreConfig) []Diagnostic
	// start the agent
	Start(AgentCoreConfig) []Diagnostic

	// before stop
	BeforeStop(AgentCoreConfig) []Diagnostic
	// stop the agent
	Stop(AgentCoreConfig) []Diagnostic
}

// CRUD for sessions map of agent
type SessionManager interface {
	// stop a session
	StopSession(string) *Diagnostic
	// get session, if not exist, create a new one if force is true
	GetSessionOnCreate(string, bool) (Session, *Diagnostic)
}

type AgentCore interface {
	Configurable
	Context
	WorkFlow
	Observable
	EventManager
	LockManager
	SessionManager
	// get ID
	GetID() string
	// set ID
	SetID() *Diagnostic
}

type AgentStatus int

const (
	AgentStatusActive AgentStatus = iota
	AgentStatusSuspended
	AgentStatusExpired // plan expired such as doesn't renew
)

func StartAgentCore(agent AgentCore, appConfigs AppConfig) []Diagnostic {
	diagnostics := []Diagnostic{}

	// DON'T modify the init order here.

	// load all configs
	agentConfigs, err := agent.LoadConfigs(agent.GetConfigPath())
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// set ID
	err = agent.SetID()
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// set logger
	err = agent.SetLogger(agentConfigs.Agent)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load history
	err = agent.SetHistory(agentConfigs.Agent.History)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load skills
	err = agent.SetSkills(agentConfigs.Agent.Skill)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load prompt
	err = agent.SetPrompt(agentConfigs.Agent.Prompt)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load knowledge base
	err = agent.SetKnowledgeBase(agentConfigs.Agent.KnowledgeBase)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	// load tool config
	err = agent.SetToolsConfig(agentConfigs.Agent.Tool)
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	beforeStartDiagnostics := agent.BeforeStart(agentConfigs)
	if len(beforeStartDiagnostics) > 0 {
		diagnostics = append(diagnostics, beforeStartDiagnostics...)
		return diagnostics
	}

	startDiagnostics := agent.Start(agentConfigs)
	if len(startDiagnostics) > 0 {
		diagnostics = append(diagnostics, startDiagnostics...)
		return diagnostics
	}

	return diagnostics
}

func StopAgentCore(agent AgentCore) []Diagnostic {
	// load all configs
	diagnostics := []Diagnostic{}
	agentConfigs, err := agent.LoadConfigs(agent.GetConfigPath())
	if err != nil {
		diagnostics = append(diagnostics, *err)
	}

	beforeStopDiagnostics := agent.BeforeStop(agentConfigs)
	if len(beforeStopDiagnostics) > 0 {
		diagnostics = append(diagnostics, beforeStopDiagnostics...)
		return diagnostics
	}

	stopDiagnostics := agent.Stop(agentConfigs)
	if len(stopDiagnostics) > 0 {
		diagnostics = append(diagnostics, stopDiagnostics...)
		return diagnostics
	}

	return diagnostics
}
