package simple

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	_ "github.com/David3310273/go-agent/agent/simple/tools"
	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
	uuid "github.com/gofrs/uuid/v5"
)

const (
	QuestionBufferSize = 10
	SimpleAgentPath    = "agent/simple"
)

// creates all registered providers from the core providerregistry
// rootPath parameter for resolving provider config file paths
func CreateProviders(rootPath string, providerNames []string) []core.Provider {
	providers := []core.Provider{}
	for _, providerName := range providerNames {
		for name, factory := range core.GetProviderFactories() {
			if name != providerName {
				continue
			}
			provider, err := factory(rootPath)
			if err != nil {
				core.LogWarn("", "failed to create provider: name=%s, error=%s", name, err.Message)
				continue
			}
			providers = append(providers, provider)
		}
	}

	return providers
}

func (a *SimpleAgent) GetSessionConfig() core.SessionConfig {
	return a.Configs.Session
}

// compile-time check that SimpleAgent implements core.AgentCore
var _ core.AgentCore = (*SimpleAgent)(nil)

type SimpleAgent struct {
	ID string
	//  project root path, set from AppConfig at startup
	RootPath string
	// event
	eventChans      map[string]chan core.Event[any]
	eventHandlers   map[string]func(core.Event[any]) core.Diagnostic
	eventBufferSize int

	// start time
	startUpTime time.Time

	// benchmarker chans
	benchmarkerChans map[string]chan core.StatEvent[any]

	// input chan
	Question chan core.Question

	// context
	ctx    context.Context
	cancel context.CancelFunc

	// configs
	Configs core.AgentCoreConfig

	sessions sync.Map

	// context fields (previously in embedded SimpleAgentContext)
	SimpleAgentContext
}

const (
	DefaultEventBufferSize = 50
)

func NewSimpleAgent(rootPath string) (*SimpleAgent, *core.Diagnostic) {
	// load config from file
	configPath := path.Join(rootPath, SimpleAgentPath, ConfigFileName)

	allConfigs := core.AgentCoreConfig{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeConfigFileFormatError,
		}
	}

	err = json.Unmarshal(data, &allConfigs)
	if err != nil {
		return nil, &core.Diagnostic{
			Level: core.SeverityError,
			Code:  core.MessageCodeConfigFileFormatError,
		}
	}

	// register root path in all sub configs
	allConfigs.Agent.RootPath = rootPath
	allConfigs.Agent.Prompt.RootPath = rootPath
	for i := range allConfigs.Agent.KnowledgeBase {
		allConfigs.Agent.KnowledgeBase[i].RootPath = rootPath
	}

	allConfigs.Session.RootPath = rootPath

	// register agent info to mcp config
	for i := range allConfigs.Agent.MCPServer {
		allConfigs.Agent.MCPServer[i].ClientName = allConfigs.Agent.Name
		allConfigs.Agent.MCPServer[i].ClientVersion = allConfigs.Agent.Version
	}

	ctx, cancel := context.WithCancel(context.Background())

	agent := &SimpleAgent{
		RootPath:         rootPath,
		eventChans:       make(map[string]chan core.Event[any]),
		eventHandlers:    make(map[string]func(core.Event[any]) core.Diagnostic),
		eventBufferSize:  DefaultEventBufferSize,
		benchmarkerChans: make(map[string]chan core.StatEvent[any]),
		// current sessions in memory, key is session id
		sessions: sync.Map{},
		// configs
		Configs:  allConfigs,
		Question: make(chan core.Question, allConfigs.Agent.QuestionBufferSize),

		ctx:    ctx,
		cancel: cancel,
	}

	diag := agent.SetID()
	if diag != nil {
		return nil, diag
	}

	return agent, nil
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
	//  use RootPath instead of hardcoded relative path
	result := path.Join(a.RootPath, SimpleAgentPath, ConfigFileName)

	return result
}

func (a *SimpleAgent) GetRootPath() string {
	return a.RootPath
}

func (a *SimpleAgent) LoadConfigs() core.AgentCoreConfig {
	return a.Configs
}

func (a *SimpleAgent) GetQuestionChan() chan core.Question {
	return a.Question
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
		// TODO: send agent during time to benchmarker
		// case <-a.eventChans[AgentEventStop]:
		// 	if !a.startTime.IsZero() {
		// 		a.benchmarkerChans
		// 	}
		case event := <-a.eventChans[core.AgentEventDeleteSessionFailed]:
			info := event.GetData()
			if sessionInfo, ok := info.(AgentEventData); ok {
				a.sessions.Delete(sessionInfo.SessionID)
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

func (a *SimpleAgent) BeforeStart() []core.Diagnostic {
	// 1. register event handlers
	a.RegisterEventChans()
	go a.OnEvent()
	// 2. model auth and init
	diagnostics := core.ValidateProviders(a.GetModelProviders())

	return diagnostics
}

func (a *SimpleAgent) Start() []core.Diagnostic {
	// get input from question, listen session output outside
	relativePath := fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))
	core.LogInfo(utils.ResolvePath(a.RootPath, relativePath), "agent started: id=%s", a.GetID())
	core.LogStd(core.LogLevelDebug, "agent Start() method entered, waiting for questions on channel, agentID=%s", a.GetID())
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
			core.LogStd(core.LogLevelDebug, "agent received question: sessionID=%s, query=%s", query.GetSessionID(), query.GetQuery())
			sessionID := query.GetSessionID()
			// organize session
			session, err := a.GetSessionOnCreate(sessionID, true)
			if err != nil {
				// send error to question's response channel when session creation fails
				core.LogStd(core.LogLevelError, "failed to create session: %s", err.Message)
				query.GetResponseChan() <- err
			} else {
				core.LogStd(core.LogLevelDebug, "session created/found, forwarding question to session: sessionID=%s", session.GetID())
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

func (a *SimpleAgent) BeforeStop() []core.Diagnostic {
	// close active sessions
	a.sessions.Range(func(key, value any) bool {
		session, ok := value.(core.Session)
		if !ok {
			return true
		}
		if err := core.StopSession(session); len(err) > 0 {
			core.Emit(a, core.CommonEvent[AgentEventData]{
				SourceType: core.SessionEventStop,
				Data: AgentEventData{
					SessionID: session.GetID(),
				},
			})
		}
		return true
	})
	return nil
}

func (a *SimpleAgent) Stop() []core.Diagnostic {
	// cancel goroutine
	a.cancel()

	return nil
}

// LockManager

func (a *SimpleAgent) Acquire() *core.Diagnostic {
	return nil
}

func (a *SimpleAgent) Release() *core.Diagnostic {
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
	value, ok := a.sessions.Load(sessionID)
	if !ok {
		return nil
	}
	session, ok := value.(core.Session)
	if !ok {
		return nil
	}
	if err := core.StopSession(session); len(err) > 0 {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSessionStopError,
			Message: "Failed to stop session",
		}
	}

	a.sessions.Delete(sessionID)

	return nil
}

// for simplicity, loading all history, no size limit here.
func (a *SimpleAgent) RecoverConversation(sessionID string) *core.Conversation {
	// resolve through ResolvePath so an absolute format is not made relative,
	// matching SimpleAgentSession.SaveMemory which writes the very same file
	memoryPath := utils.ResolvePath(a.RootPath, fmt.Sprintf(a.GetSessionConfig().MemoryFilePathFormat, sessionID))

	fp, err := os.Open(memoryPath)
	if err != nil {
		return nil
	}

	defer fp.Close()

	memories := make(core.Conversation, 0)
	reader := bufio.NewReader(fp)

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		line = strings.TrimSuffix(line, "\n")
		message := core.ReActMessage{}
		if err := json.Unmarshal([]byte(line), &message); err == nil {
			memories = append(memories, message)
		}
	}

	return &memories
}

// get session, if not exist and forceCreate is true, create a new one
func (a *SimpleAgent) GetSessionOnCreate(sessionID string, forceCreate bool) (core.Session, *core.Diagnostic) {
	// pre-check session existence, avoid unnecessary new session creation
	if value, ok := a.sessions.Load(sessionID); ok {
		return value.(core.Session), nil
	}

	if !forceCreate {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSessionNotFound,
			Message: fmt.Sprintf("Session %s not found", sessionID),
		}
	}

	memories := a.RecoverConversation(sessionID)
	session := NewAgentSession(a, sessionID, memories)

	newSession, ok := a.sessions.LoadOrStore(sessionID, session)
	if ok {
		core.LogInfo(utils.ResolvePath(a.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))), "session found, reusing: session=%s", sessionID)
		return newSession.(core.Session), nil
	} else {
		core.LogInfo(utils.ResolvePath(a.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))), "session not found, creating: session=%s", sessionID)
		// start session after registered in agent
		go core.StartSession(newSession.(core.Session))
		return session, nil
	}
}

func (a *SimpleAgent) ListSessions() []core.Session {
	result := make([]core.Session, 0)
	a.sessions.Range(func(key, value any) bool {
		session, ok := value.(core.Session)
		if ok {
			result = append(result, session)
		}
		return true
	})
	return result
}

func (a *SimpleAgent) GetSession(sessionID string) core.Session {
	value, ok := a.sessions.Load(sessionID)
	if !ok {
		return nil
	}
	session, _ := value.(core.Session)
	return session
}

func (a *SimpleAgent) RemoveSession(sessionID string) *core.Diagnostic {
	session := a.GetSession(sessionID)
	if session == nil {
		return nil
	}

	a.sessions.Delete(sessionID)

	filePath := utils.ResolvePath(a.RootPath, fmt.Sprintf(session.GetConfigs().MemoryFilePathFormat, sessionID))
	if err := os.Remove(filePath); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSessionDeleteError,
			Message: "Failed to delete session file: " + filePath,
		}
	}

	filePath = utils.ResolvePath(a.RootPath, fmt.Sprintf(session.GetConfigs().ChatHistoryFilePathFormat, sessionID))
	if err := os.Remove(filePath); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSessionDeleteError,
			Message: "Failed to delete session file: " + filePath,
		}
	}

	return nil
}
