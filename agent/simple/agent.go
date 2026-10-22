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

	"github.com/David3310273/go-agent/agent/simple/utils"
	benchmarker "github.com/David3310273/go-agent/benchmarker"
	"github.com/David3310273/go-agent/benchmarker/metrics"
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
	cfg := a.Configs.Session
	cfg.EnableBenchmark = a.Configs.Agent.EnableBenchmark
	return cfg
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
	benchmarkerChans      map[core.MetricType]chan core.BenchmarkEvent[any]
	benchmarkerEventChans map[core.MetricType]chan core.BenchmarkEvent[any]

	// auto-add: agent holds a reference to the benchmarker so that when
	// new sessions are created (and EnableBenchmark is true), the agent
	// can notify the benchmarker to subscribe to the session's metrics channels.
	benchmarker *benchmarker.Benchmarker

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

	// broadcast root path in all sub configs
	allConfigs.Agent.RootPath = rootPath
	allConfigs.Agent.Prompt.RootPath = rootPath

	for i := range allConfigs.Agent.KnowledgeBase {
		allConfigs.Agent.KnowledgeBase[i].RootPath = rootPath
	}
	for i := range allConfigs.Agent.Tool {
		allConfigs.Agent.Tool[i].RootPath = rootPath
	}

	allConfigs.Session.RootPath = rootPath
	allConfigs.Benchmarker.RootPath = rootPath

	// register agent info to mcp config
	for i := range allConfigs.Agent.MCPServer {
		allConfigs.Agent.MCPServer[i].ClientName = allConfigs.Agent.Name
		allConfigs.Agent.MCPServer[i].ClientVersion = allConfigs.Agent.Version
	}

	ctx, cancel := context.WithCancel(context.Background())

	agent := &SimpleAgent{
		RootPath:        rootPath,
		eventChans:      make(map[string]chan core.Event[any]),
		eventHandlers:   make(map[string]func(core.Event[any]) core.Diagnostic),
		eventBufferSize: DefaultEventBufferSize,
		// benchmarker pull or push channels
		benchmarkerChans:      make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		benchmarkerEventChans: make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		// current sessions in memory, key is session id
		sessions: sync.Map{},
		// configs
		Configs:  allConfigs,
		Question: make(chan core.Question, allConfigs.Agent.QuestionBufferSize),

		ctx:    ctx,
		cancel: cancel,

		SimpleAgentContext: SimpleAgentContext{
			RootPath: rootPath,
		},
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

func (a *SimpleAgent) SetBenchmarker(bm *benchmarker.Benchmarker) {
	a.benchmarker = bm
}

// get available tools
func (a *SimpleAgent) GetToolsConfig() []core.ToolConfig {
	return a.Configs.Agent.Tool
}

// get mcp config
func (a *SimpleAgent) GetMCPServerConfigs() []core.MCPConfig {
	return a.Configs.Agent.MCPServer
}

func (a *SimpleAgent) SetMCPServerConfigs(configs []core.MCPConfig) {
	a.Configs.Agent.MCPServer = configs
}

// get skill
func (a *SimpleAgent) GetSkillConfig(name string) core.SkillDefinition {
	skillDefs := a.Configs.Agent.Skill

	for _, skillDef := range skillDefs {
		if skillDef.Name == name {
			return skillDef
		}
	}
	return core.SkillDefinition{}
}

// get all skills config
func (a *SimpleAgent) GetSkillConfigs() []core.SkillDefinition {
	return a.Configs.Agent.Skill
}

// EventManager
/*
	- Event handler is hard to be uniformed because of different param type and count, so define this in each module
*/
func (a *SimpleAgent) OnEvent() {
	// auto-add: benchmark pull channel moved to outer select so it can be
	// selected independently (fixes nested select with default that silently
	// dropped every benchmarker pull request).
	var benchmarkPullCh chan core.BenchmarkEvent[any]
	if a.Configs.Agent.EnableBenchmark {
		benchmarkPullCh = a.benchmarkerEventChans[core.MetricAgentRunningTime]
	}

	for {
		select {
		case <-a.ctx.Done():
			return
		// for benchmarker
		case <-a.eventChans[core.AgentEventStart]:
			a.startUpTime = time.Now()
		case event := <-a.eventChans[core.AgentEventDeleteSessionFailed]:
			info := event.GetData()
			if sessionInfo, ok := info.(AgentEventData); ok {
				a.sessions.Delete(sessionInfo.SessionID)
			}
		case event := <-benchmarkPullCh:
			if core.MetricType(event.GetSourceType()) == core.MetricAgentRunningTime {
				core.LogInfo("", "[agent] OnEvent: received pull request, sending running time to listening channel agentID=%s", a.GetID())
				core.SendBenchmarkEvent(a.benchmarkerChans[core.MetricAgentRunningTime], core.BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      core.MetricAgentRunningTime,
					Data: metrics.AgentRunningTimeMetric{
						AgentID:   a.GetID(),
						StartTime: a.startUpTime,
						EndTime:   time.Now(),
					},
				})
			}
		}
	}
}

func (a *SimpleAgent) RegisterEventChans() {
	// auto-add: make RegisterEventChans idempotent so it can be called
	// from both BeforeStart and GetBenchmarkListeningChannels safely.
	if _, ok := a.eventChans[core.AgentEventStart]; !ok {
		core.LogInfo("", "[agent] RegisterEventChans: creating event channels")
		a.eventChans[core.AgentEventStart] = make(chan core.Event[any], a.eventBufferSize)
	}
	if _, ok := a.eventChans[core.AgentEventDeleteSessionFailed]; !ok {
		a.eventChans[core.AgentEventDeleteSessionFailed] = make(chan core.Event[any], a.eventBufferSize)
	}
	if _, ok := a.eventChans[core.AgentEventCreateSessionFailed]; !ok {
		a.eventChans[core.AgentEventCreateSessionFailed] = make(chan core.Event[any], a.eventBufferSize)
	}
	// register benchmarker event chan
	if a.Configs.Agent.EnableBenchmark {
		core.LogInfo("", "[agent] RegisterEventChans: EnableBenchmark=true, creating benchmark channels")
		if _, ok := a.benchmarkerEventChans[core.MetricAgentRunningTime]; !ok {
			core.LogInfo("", "[agent] RegisterEventChans: creating benchmarkerEventChans[MetricAgentRunningTime]")
			a.benchmarkerEventChans[core.MetricAgentRunningTime] = make(chan core.BenchmarkEvent[any], a.eventBufferSize)
		}
		if _, ok := a.benchmarkerChans[core.MetricAgentRunningTime]; !ok {
			core.LogInfo("", "[agent] RegisterEventChans: creating benchmarkerChans[MetricAgentRunningTime]")
			a.benchmarkerChans[core.MetricAgentRunningTime] = make(chan core.BenchmarkEvent[any], a.eventBufferSize)
		}
	} else {
		core.LogInfo("", "[agent] RegisterEventChans: EnableBenchmark=false, skipping benchmark channels")
	}
}

func (a *SimpleAgent) GetEventChans() map[string]chan core.Event[any] {
	a.RegisterEventChans()
	return a.eventChans
}

// WorkFlow

func (a *SimpleAgent) BeforeStart() []core.Diagnostic {
	// 1. register event handlers
	a.RegisterEventChans()

	// auto-add: initialise benchmarker if enabled by config.
	// Must run after RegisterEventChans (so channels exist) and before
	// go a.OnEvent() (so benchmarker can subscribe before event loop starts).
	if a.Configs.Agent.EnableBenchmark {
		a.benchmarker = benchmarker.NewBenchmarker(a.Configs.Benchmarker)
		a.benchmarker.OnEvent(a)
		if errs := a.benchmarker.Start(); len(errs) > 0 {
			core.LogError("", "[agent] benchmarker start failed: %v", errs)
		}
	}

	go a.OnEvent()
	// 2. model auth and init
	diagnostics := core.ValidateProviders(a.GetModelProviders())

	return diagnostics
}

func (a *SimpleAgent) Start() []core.Diagnostic {
	// get input from question, listen session output outside
	relativePath := fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))
	core.LogInfo(utils.ResolvePath(a.RootPath, relativePath), "agent started: id=%s", a.GetID())

	// emit start event for benchmark
	core.Emit(a, core.CommonEvent[AgentEventTimeData]{
		SourceType: core.AgentEventStart,
		Data: AgentEventTimeData{
			SnapshotTime: time.Now(),
			Message:      fmt.Sprintf("Agent %s started", a.GetID()),
		},
	})
	// determine worker count: config value or default to 1
	workerCount := a.Configs.Agent.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	core.LogStd(core.LogLevelInfo, "agent starting with %d dispatcher workers, agentID=%s", workerCount, a.GetID())

	// launch worker goroutines to consume from the shared Question channel
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			core.LogStd(core.LogLevelDebug, "agent dispatcher worker %d started, agentID=%s", workerID, a.GetID())

			for {
				select {
				case query, ok := <-a.Question:
					if !ok {
						core.LogStd(core.LogLevelDebug, "agent Question channel closed, worker %d exiting, agentID=%s", workerID, a.GetID())
						return
					}
					core.LogStd(core.LogLevelDebug, "agent worker %d received question: sessionID=%s, query=%s", workerID, query.GetSessionID(), query.GetQuery())
					sessionID := query.GetSessionID()
					session, err := a.GetSessionOnCreate(sessionID, true)
					if err != nil {
						core.LogStd(core.LogLevelError, "failed to create session: %s", err.Message)
						select {
						case query.GetResponseChan() <- err:
						default:
						}
					} else {
						core.LogStd(core.LogLevelDebug, "worker %d forwarding question to session: sessionID=%s", workerID, session.GetID())
						session.GetQuestionChan() <- query
					}

				case <-a.ctx.Done():
					core.LogStd(core.LogLevelDebug, "agent context done, worker %d exiting, agentID=%s", workerID, a.GetID())
					return
				}
			}
		}(i)
	}

	// wait for all workers to finish before returning
	wg.Wait()
	core.LogStd(core.LogLevelDebug, "all dispatcher workers exited, agentID=%s", a.GetID())

	// send start event
	if a.Configs.Agent.EnableBenchmark {
		core.LogInfo("", "[agent] Start: sending agent running time via benchmarkerEventChans (pull trigger) agentID=%s", a.GetID())
		currentTime := time.Now()
		core.SendBenchmarkEvent(a.benchmarkerEventChans[core.MetricAgentRunningTime], core.BenchmarkEvent[any]{
			TimeStamp: currentTime.UnixMilli(),
			Type:      core.MetricAgentRunningTime,
			Data: metrics.AgentRunningTimeMetric{
				AgentID:   a.GetID(),
				StartTime: a.startUpTime,
				EndTime:   currentTime,
			},
		})
	}

	return []core.Diagnostic{
		{
			Level: core.SeverityWarn,
			Code:  core.MessageSystemClosed,
		},
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

	// notify benchmarker to clean up pull channels for the agent itself
	if a.benchmarker != nil && a.Configs.Agent.EnableBenchmark {
		core.LogInfo(utils.ResolvePath(a.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))), "[agent=%s] BeforeStop: unregistering from benchmarker", a.GetID())
		a.benchmarker.Unregister(a.GetID())
	}

	// close opened kb
	for _, kb := range a.SimpleAgentContext.KnowledgeBase {
		kb.Close()
	}

	return nil
}

func (a *SimpleAgent) Stop() []core.Diagnostic {
	// cancel goroutine
	a.cancel()

	// emit stop event for benchmark
	if a.Configs.Agent.EnableBenchmark {
		core.LogInfo("", "[agent] Stop: sending agent running time via benchmarkerChans (direct push) agentID=%s startUpTime=%v",
			a.GetID(), a.startUpTime)
		core.SendBenchmarkEvent(a.benchmarkerChans[core.MetricAgentRunningTime], core.BenchmarkEvent[any]{
			TimeStamp: time.Now().UnixMilli(),
			Type:      core.MetricAgentRunningTime,
			Data: metrics.AgentRunningTimeMetric{
				AgentID:   a.GetID(),
				StartTime: a.startUpTime,
				EndTime:   time.Now(),
			},
		})

		a.CloseBenchmarkListeningChannels()

		// shut down the embedded benchmarker
		if a.benchmarker != nil {
			core.StopBenchmark(a.benchmarker)
		}
	}

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
func (a *SimpleAgent) GetObjectID() string {
	return a.GetID()
}

func (a *SimpleAgent) GetBenchmarkListeningChannels() map[core.MetricType]chan core.BenchmarkEvent[any] {
	// auto-add: ensure channels are registered before returning.
	// This fixes a race condition where bm.OnEvent(agent) is called
	// before StartAgentServer (which calls RegisterEventChans) completes.
	a.RegisterEventChans()
	return a.benchmarkerChans
}

func (a *SimpleAgent) GetBenchmarkPullChannels() map[core.MetricType]chan core.BenchmarkEvent[any] {
	// auto-add: ensure channels are registered before returning.
	a.RegisterEventChans()
	return a.benchmarkerEventChans
}

func (a *SimpleAgent) CloseBenchmarkListeningChannels() {
	// notify benchmarker to clean up pull channels for this agent
	if a.benchmarker != nil && a.Configs.Agent.EnableBenchmark {
		a.benchmarker.Unregister(a.GetID())
	}
	for key, ch := range a.benchmarkerChans {
		core.LogInfo(utils.ResolvePath(a.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(a.Configs.Agent.LogPath, a.GetID()))), "[agent=%s] closing benchmark listening channel metric=%s", a.GetID(), key)
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

	// auto-add: register session's benchmark listening/pull channels
	// with the benchmarker so session-level metrics (query time, token
	// usage, etc.) flow to Prometheus.
	if a.benchmarker != nil && a.Configs.Agent.EnableBenchmark {
		core.LogInfo("", "[agent] registering session metrics with benchmarker: session=%s", sessionID)
		a.benchmarker.OnEvent(session)
	}

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

	if err := a.StopSession(sessionID); err != nil {
		return err
	}

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
