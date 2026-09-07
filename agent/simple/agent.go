package simple

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/David3310273/go-agent/core"
	uuid "github.com/gofrs/uuid/v5"
)

type SimpleAgentContext struct {
	// language
	Language core.LanguageType
	// prompt
	Prompt []byte
	// knowledge base
	KnowledgeBase []byte
	// skills
	Skills []byte
	// history
	History []byte
	// model providers, for simple agent, only one provider without model routing
	ModelProviders []core.Provider
	// support tools
	Tools []core.ToolConfig
}

const (
	QuestionBufferSize = 10
)

// Context getter methods

func (c *SimpleAgentContext) GetHistory() []byte {
	return c.History
}

func (a *SimpleAgent) SetHistory(history core.HistoryConfig) *core.Diagnostic {
	a.History = make([]byte, a.Configs.Agent.History.BufferSize)
	return nil
}

func (c *SimpleAgentContext) SetLanguage(language core.LanguageType) {
	c.Language = language
}

func (c *SimpleAgentContext) GetToolsConfig() []core.ToolConfig {
	return c.Tools
}

func (c *SimpleAgentContext) GetPrompt() []byte {
	return c.Prompt
}

func (a *SimpleAgent) SetPrompt(prompt core.PromptConfig) *core.Diagnostic {
	// use capacity instead of length, and convert KB to bytes
	a.Prompt = make([]byte, 0, a.Configs.Agent.Prompt.BufferSizeInKB*1024)
	// load prompt in memory per file as much as possible
	contentSize := 0
	cwd, _ := os.Getwd()

	for _, filename := range prompt.Paths {
		// hard code here, for simplicity
		realPath := path.Join(cwd, "..", "agent/simple", filename)
		log.Printf("real prompt path: %s", realPath)
		tempPrompt, err := os.ReadFile(realPath)
		if err != nil {
			core.Emit(a, core.CommonEvent[AgentEventData]{
				SourceType: core.AgentEventPromptLoadFailed,
				Data: AgentEventData{
					AgentID: a.ID,
				},
			})
		} else if contentSize+len(tempPrompt) <= a.Configs.Agent.Prompt.BufferSizeInKB*1024 {
			a.Prompt = append(a.Prompt, tempPrompt...)
			contentSize += len(tempPrompt)
		} else {
			a.Logger.Printf("cannot load whole prompt %s because buffer is full, will truncate in here...", realPath)
			break
		}
	}

	return nil
}

func (c *SimpleAgentContext) GetKnowledgeBase() []byte {
	return c.KnowledgeBase
}

func (a *SimpleAgent) SetKnowledgeBase(knowledge core.KnowledgeBaseConfig) *core.Diagnostic {
	return nil
}

func (c *SimpleAgentContext) GetSkills() []byte {
	return c.Skills
}

func (a *SimpleAgent) SetSkills(skill core.SkillConfig) *core.Diagnostic {
	return nil
}

// SetToolsConfig loads a list of ToolConfig into the agent's tool list,
// skipping entries with an empty name.
func (a *SimpleAgent) SetToolsConfig(tools []core.ToolConfig) *core.Diagnostic {
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
func (a *SimpleAgent) SetProviders(providers []core.Provider) *core.Diagnostic {
	a.ModelProviders = providers
	return nil
}

// creates all registered providers from the core providerregistry
func CreateProviders() []core.Provider {
	providers := []core.Provider{}
	for name, factory := range core.GetProviderFactories() {
		provider, err := factory()
		if err != nil {
			log.Printf("failed to create provider %s: %s", name, err.Message)
			continue
		}
		providers = append(providers, provider)
	}

	return providers
}

func (a *SimpleAgent) GetSessionConfig() core.SessionConfig {
	return a.Configs.Session
}

func (c *SimpleAgentContext) GetTools() []core.ToolConfig {
	return c.Tools
}

// compile-time check that SimpleAgent implements core.AgentCore
var _ core.AgentCore = (*SimpleAgent)(nil)

type SimpleAgent struct {
	ID string
	// event
	eventChans      map[string]chan core.Event[any]
	eventHandlers   map[string]func(core.Event[any]) core.Diagnostic
	eventBufferSize int

	// logger
	Logger *log.Logger

	// start time
	startUpTime time.Time

	// benchmarker chans
	benchmarkerChans map[string]chan core.StatEvent[any]

	// input chan
	Question chan core.Question

	// lock
	mu chan struct{}

	// context
	ctx    context.Context
	cancel context.CancelFunc

	// configs
	Configs core.AgentCoreConfig

	sessions map[string]*SimpleAgentSession

	// context fields (previously in embedded SimpleAgentContext)
	SimpleAgentContext
}

const (
	DefaultEventBufferSize = 50
)

func NewSimpleAgent() *SimpleAgent {
	ctx, cancel := context.WithCancel(context.Background())
	agent := &SimpleAgent{
		eventChans:       make(map[string]chan core.Event[any]),
		eventHandlers:    make(map[string]func(core.Event[any]) core.Diagnostic),
		eventBufferSize:  DefaultEventBufferSize,
		benchmarkerChans: make(map[string]chan core.StatEvent[any]),
		// current sessions in memory, key is session id
		sessions: make(map[string]*SimpleAgentSession),

		mu:       make(chan struct{}, 1),
		Question: make(chan core.Question, QuestionBufferSize),

		ctx:    ctx,
		cancel: cancel,
	}

	return agent
}

const (
	ConfigFileName = "config.json"
)

func (a *SimpleAgent) GetID() string {
	return a.ID
}

func (a *SimpleAgent) SetID() *core.Diagnostic {
	uuid, err := uuid.NewV4()
	if err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeAgentCoreConfigError,
			Message: fmt.Sprintf("Failed to generate UUID: %v", err),
		}
	}

	a.ID = uuid.String()

	return nil
}

// Configure

func (a *SimpleAgent) GetConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	// hard code here, for simplicity
	result := path.Join(cwd, "..", "agent/simple", ConfigFileName)

	return result
}

func (a *SimpleAgent) LoadConfigs(path string) (core.AgentCoreConfig, *core.Diagnostic) {
	if path == "" {
		path = a.GetConfigPath()
	}

	allConfigs := core.AgentCoreConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return allConfigs, &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeConfigFileFormatError,
		}
	}

	err = json.Unmarshal(data, &allConfigs)
	if err != nil {
		return allConfigs, &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeConfigFileFormatError,
		}
	}

	return allConfigs, nil
}

// Logger

func (a *SimpleAgent) SetLogger(config core.AgentConfig) *core.Diagnostic {
	pathFormat := config.LogPath
	folder := path.Dir(pathFormat)

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			return &core.Diagnostic{
				Level: core.SeverityError,
				Code:  core.MessageCodeConfigFileFormatError,
			}
		}
	}

	logPath := fmt.Sprintf(pathFormat, a.GetID())
	log.Printf("set agent log path: %s, %s", pathFormat, logPath)
	a.Logger = core.NewLogger(logPath)

	return nil
}

// EventManager
/*
	- Event handler is hard to be uniformed because of different param type and count, so define this in each module
*/
func (a *SimpleAgent) OnEvent() {
	for {
		select {
		case <-a.ctx.Done():
			return
		// for benchmarker
		case <-a.eventChans[core.AgentEventStart]:
			a.startUpTime = time.Now()
		// TODO: send agent during time to benckmarker
		// case <-a.eventChans[AgentEventStop]:
		// 	if !a.startTime.IsZero() {
		// 		a.benchmarkerChans
		// 	}
		case event := <-a.eventChans[core.AgentEventDeleteSessionFailed]:
			info := event.GetData()
			if sessionInfo, ok := info.(AgentEventData); ok {
				if err := a.Acquire(); err == nil {
					delete(a.sessions, sessionInfo.SessionID)
				} else {
					// random sleep and retry
					randomSec := rand.Intn(10) + 1
					time.Sleep(time.Duration(randomSec) * time.Second)
					core.Emit(a, event)
				}
			}
		}
	}
}

func (a *SimpleAgent) RegisterEventChans() {
	a.eventChans[core.AgentEventStart] = make(chan core.Event[any], a.eventBufferSize)
	a.eventChans[core.AgentEventStop] = make(chan core.Event[any], a.eventBufferSize)
	a.eventChans[core.AgentEventDeleteSessionFailed] = make(chan core.Event[any], a.eventBufferSize)
	a.eventChans[core.AgentEventCreateSessionFailed] = make(chan core.Event[any], a.eventBufferSize)
}

func (a *SimpleAgent) GetEventChans() map[string]chan core.Event[any] {
	a.RegisterEventChans()
	return a.eventChans
}

// WorkFlow

func (a *SimpleAgent) BeforeStart(config core.AgentCoreConfig) []core.Diagnostic {
	// 1. register event handlers
	a.RegisterEventChans()
	go a.OnEvent()
	// 2. model auth and init
	diagnostics := core.ValidateProviders(a.GetModelProviders())

	return diagnostics
}

func (a *SimpleAgent) Start(config core.AgentCoreConfig) []core.Diagnostic {
	// get input from question, listen session output outside
	a.Logger.Printf("Agent %s started", a.GetID())
	// emit start event for benchmark
	core.Emit(a, core.CommonEvent[AgentEventTimeData]{
		SourceType: core.AgentEventStart,
		Data: AgentEventTimeData{
			SnapshotTime: time.Now(),
			Message:      fmt.Sprintf("Agent %s started", a.GetID()),
		},
	})

	for {
		select {
		case query := <-a.Question:
			sessionID := query.GetSessionID()
			// organize session
			session, err := a.GetSessionOnCreate(sessionID, true)
			if err != nil {
				// send error to question's response channel when session creation fails
				query.GetResponseChan() <- err
			} else {
				session.GetQuestionChan() <- query
			}

		case <-a.ctx.Done():
			return []core.Diagnostic{
				{
					Level: core.SeverityWarn,
					Code:  core.MessageSystemClosed,
				},
			}
		}
	}
}

func (a *SimpleAgent) BeforeStop(config core.AgentCoreConfig) []core.Diagnostic {
	// close active sessions
	for _, session := range a.sessions {
		if err := core.StopSession(session, config); len(err) > 0 {
			core.Emit(a, core.CommonEvent[AgentEventData]{
				SourceType: core.SessionEventStop,
				Data: AgentEventData{
					SessionID: session.GetID(),
				},
			})
		}
	}
	return nil
}

func (a *SimpleAgent) Stop(config core.AgentCoreConfig) []core.Diagnostic {
	// cancel goroutine
	a.cancel()

	return nil
}

// LockManager

func (a *SimpleAgent) Acquire() *core.Diagnostic {
	select {
	case a.mu <- struct{}{}:
		return nil
	case <-a.ctx.Done():
		return nil
	case <-time.After(LockTimeout):
		return &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeLockFailed,
		}
	}
}

func (a *SimpleAgent) Release() *core.Diagnostic {
	<-a.mu
	return nil
}

// Observable for benchmarker
func (a *SimpleAgent) GetBenchmarkListeningChannels() map[string]chan core.StatEvent[any] {
	return a.benchmarkerChans
}

func (a *SimpleAgent) CloseBenchmarkListeningChannels() {
	for key, ch := range a.benchmarkerChans {
		close(ch)
		delete(a.benchmarkerChans, key)
	}
}

// SessionManager

func (a *SimpleAgent) StopSession(sessionID string) *core.Diagnostic {
	session, ok := a.sessions[sessionID]
	if !ok {
		return nil
	} else {
		if err := core.StopSession(session, a.Configs); len(err) > 0 {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSessionStopError,
				Message: "Failed to stop session",
			}
		}

		// auto-added: acquire lock first, only release if acquired
		if err := a.Acquire(); err == nil {
			defer a.Release()
			delete(a.sessions, sessionID)
		} else {
			event := core.CommonEvent[AgentEventData]{
				SourceType: core.AgentEventDeleteSessionFailed,
				Data:       AgentEventData{SessionID: sessionID},
			}
			core.Emit(a, event)
		}
	}

	return nil
}

// get session, if not exist and forceCreate is true, create a new one
func (a *SimpleAgent) GetSessionOnCreate(sessionID string, forceCreate bool) (core.Session, *core.Diagnostic) {
	if session, ok := a.sessions[sessionID]; ok {
		return session, nil
	} else if forceCreate {
		log.Printf("session %s not found, will create new one", sessionID)

		session := NewAgentSession(a)

		// acquire lock first
		if err := a.Acquire(); err == nil {
			defer a.Release()
			a.sessions[session.GetID()] = session
			// start session after registered in agent
			go core.StartSession(session, a.Configs)

			return session, nil
		}

		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSessionCreateError,
			Message: "Failed to acquire lock for session creation",
		}

	} else {
		return nil, &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeSessionCreateError,
		}
	}
}
