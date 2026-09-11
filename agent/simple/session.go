package simple

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/David3310273/go-agent/agent/simple/tools"
	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
	uuid "github.com/gofrs/uuid/v5"
)

var _ core.Session = (*SimpleAgentSession)(nil)

const (
	LockTimeout = 3 * time.Second
	HistoryPath = "history.json"
)

// SimpleSessionContext holds session-specific context data
type SimpleSessionContext struct {
	SimpleAgentContext

	Conversation core.Conversation
}

// SimpleAgentSession represents a single agent session
/*
	- only one question can be answered, no follow-up questions nor steer mode.
*/
type SimpleAgentSession struct {
	ID            string
	ParentSession *SimpleAgentSession
	SubSessions   []*SimpleAgentSession
	Status        core.SessionStatus
	Providers     []core.Provider
	// logger
	Logger *log.Logger

	// lock
	mu          chan struct{}
	lockTimeOut time.Duration

	// start time
	startTime time.Time
	// start process time
	startProcessTime time.Time
	// streaming
	streaming bool
	// enable thinking
	enableThinking bool

	Config core.SessionConfig

	// context
	ctx    context.Context
	cancel context.CancelFunc

	// user input chan
	Question chan core.Question
	// becnchmark chan
	benchmarkerChans map[string]chan core.StatEvent[any]
	// event chans
	eventChans map[string]chan core.Event[any]
	// session context
	SimpleSessionContext
}

// NewAgentSession creates a new session from an AgentCore
// log providers count when creating session
func NewAgentSession(agent core.AgentCore, streaming bool, enableThinking bool) *SimpleAgentSession {
	providers := agent.GetModelProviders()
	log.Printf("NewAgentSession: providers count = %d", len(providers))

	sessionConfig := agent.GetSessionConfig()
	//  propagate RootPath from agent to session config
	if a, ok := agent.(*SimpleAgent); ok {
		sessionConfig.RootPath = a.RootPath
	}

	session := &SimpleAgentSession{
		Providers:   providers,
		Status:      core.SessionStatusRunning,
		mu:          make(chan struct{}, 1),
		lockTimeOut: LockTimeout,
		Question:    make(chan core.Question),
		Config:      sessionConfig,

		benchmarkerChans: make(map[string]chan core.StatEvent[any]),
		eventChans:       make(map[string]chan core.Event[any]),
		streaming:        streaming,
		enableThinking:   true,
	}

	id, _ := uuid.NewV4()
	session.ID = id.String()

	// init context
	session.ctx, session.cancel = context.WithCancel(context.Background())

	// register event channels and start event listener
	session.RegisterEventChans()
	go session.OnEvent()

	// llm context init
	session.History = agent.GetHistory()
	session.Prompt = agent.GetPrompt()
	session.Skills = agent.GetSkills()
	session.KnowledgeBase = agent.GetKnowledgeBase()
	session.Tools = agent.GetToolsConfig()
	session.Conversation = core.Conversation{}

	// use sessionConfig (with RootPath set) instead of original config
	session.SetLogger(sessionConfig)

	return session
}

// NewSubSession creates a child session
func (s *SimpleAgentSession) NewSubSession() core.Session {
	session := &SimpleAgentSession{
		ParentSession: s,
		Providers:     s.Providers,
		Status:        core.SessionStatusRunning,
		mu:            make(chan struct{}, 1),
	}

	session.ctx, session.cancel = context.WithCancel(s.ctx)

	id, _ := uuid.NewV4()
	session.ID = id.String()

	return session
}

func (s *SimpleAgentSession) SetLogger(config core.SessionConfig) *core.Diagnostic {
	//  use RootPath for log directory instead of relative path
	realPath := path.Join(config.RootPath, config.LogPath)
	folder := path.Dir(realPath)

	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			return &core.Diagnostic{
				Level: core.SeverityError,
				Code:  core.MessageCodeConfigFileFormatError,
			}
		}
	}

	logPath := path.Join(fmt.Sprintf(realPath, s.GetID()))
	s.Logger = core.NewLogger(logPath)

	return nil
}

// =============================================================================
// SessionContext methods
// =============================================================================

func (s *SimpleAgentSession) GetConfigs() core.SessionConfig {
	return s.Config
}

func (s *SimpleAgentSession) SetSessionHistory() core.Diagnostic {
	return core.Diagnostic{}
}

func (s *SimpleAgentSession) GenerateFinalContext(query core.Question) string {
	// simple harness: mergethe context from prompt, skills, history, and session history
	result := []byte{}
	result = append(result, s.Prompt...)
	result = append(result, s.Skills...)
	result = append(result, s.History...)

	return string(result)
}

// =============================================================================
// Session methods
// =============================================================================

func (s *SimpleAgentSession) GetID() string {
	return s.ID
}

func (s *SimpleAgentSession) GetParentID() string {
	if s.ParentSession == nil {
		return ""
	}
	return s.ParentSession.ID
}

func (s *SimpleAgentSession) GetQuestionChan() chan core.Question {
	return s.Question
}

func (s *SimpleAgentSession) GetModelProviders() []core.Provider {
	return s.Providers
}

func (s *SimpleAgentSession) GetStatus() core.SessionStatus {
	return s.Status
}

func (s *SimpleAgentSession) SetStatus(status core.SessionStatus) *core.Diagnostic {
	if s.Status != status {
		s.Status = status
	}

	return nil
}

// return pointer so callers can modify the conversation in place
func (s *SimpleAgentSession) GetConversation() *core.Conversation {
	// TODO: sliding window for controlling history
	return &s.Conversation
}

func (s *SimpleAgentSession) SaveHistory(history core.ReActMessage) *core.Diagnostic {
	//  use RootPath instead of hardcoded relative path
	filePath := path.Join(s.Config.RootPath, s.Config.MemoryFilePathFormat)
	filename := fmt.Sprintf(filePath, s.GetID())

	basePath := path.Dir(filename)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		err = os.MkdirAll(basePath, 0755)
		if err != nil {
			return &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeWriteHistoryError,
				Message: err.Error(),
			}
		}
	}

	content := []byte(history.ToString())
	content = append(content, '\n')

	if err := utils.RotateWrite(filename, s.Config.MemoryFileSplitter, 1024*1024*1024, content); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeWriteHistoryError,
			Message: err.Error(),
		}
	}

	return &core.Diagnostic{}
}

// select tools given user question and tool config
// return tools used this time
// auto-added: match tools by name instead of hardcoded index.
func (s *SimpleAgentSession) SelectTools(query core.Question, message core.ReActMessage) []core.Tool {
	var result []core.Tool
	for _, cfg := range s.Tools {
		switch cfg.Name {
		case "filewriter":
			result = append(result, tools.FileWriterCall{
				Name:     cfg.Name,
				Schema:   cfg.Schema,
				RootPath: s.Config.RootPath,
			})
		case "getdate":
			result = append(result, tools.GetDateCall{
				Name:     cfg.Name,
				Schema:   cfg.Schema,
				RootPath: s.Config.RootPath,
			})
		}
	}
	return result
}

// select local kb given the question, merge into final context
// TODO: will search in local knowledge base, currently be simple here
func (s *SimpleAgentSession) SelectLocalKB(query core.Question) string {
	return string(s.KnowledgeBase)
}

// prepare for app layer
type SimpleSessionResponse struct {
	Response  core.AgentResponse `json:"response"`
	SessionID string             `json:"sessionID"`
}

func (s SimpleSessionResponse) ToString() string {
	response, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(response)
}

// GetSessionID returns the session ID for SessionAnswer interface
func (s SimpleSessionResponse) GetSessionID() string {
	return s.SessionID
}

func (s *SimpleAgentSession) ProcessQuery(query core.Question) {
	// debug log
	log.Printf("ProcessQuery: session %s, query = %s", s.GetID(), query.GetQuery())

	// use goroutine and channel to handle timeout
	type processResult struct {
		response    core.Answer
		diagnostics []core.Diagnostic
	}
	resultChan := make(chan processResult, 1)
	// emit start event for benchmark
	core.Emit(s, core.CommonEvent[SessionEventTimeData]{
		SourceType: core.SessionStartProcessQuestion,
		Data: SessionEventTimeData{
			SnapshotTime: time.Now(),
			SessionID:    s.GetID(),
		},
	})

	// async process query
	go func() {
		var response core.Answer
		var diagnostics []core.Diagnostic

		if s.streaming {
			response, diagnostics = core.ProcessQuestionStream(s, query)
		} else {
			response, diagnostics = core.ProcessQuestion(s, query)
		}

		// use select to avoid goroutine leak when context is cancelled
		select {
		case resultChan <- processResult{response: response, diagnostics: diagnostics}:
		case <-s.ctx.Done():
		}
	}()

	// wait for result or context cancellation
	select {
	case result := <-resultChan:
		// default answer if failed
		finalAnswer := SimpleSessionResponse{
			SessionID: s.GetID(),
			Response: core.AgentResponse{
				Response: query.GetDefaultAnswer().ToString(),
			},
		}

		if len(result.diagnostics) > 0 {
			log.Printf("[Session] processQuery error: %v", result.diagnostics)
		} else {
			answer, ok := result.response.(core.AgentResponse)
			if ok {
				finalAnswer.Response = answer
			}
		}

		// always send finalAnswer to responseChan so streaming handler can extract sessionID and usage
		select {
		case query.GetResponseChan() <- finalAnswer:
			// finished process question
			core.Emit(s, core.CommonEvent[SessionEventTimeData]{
				SourceType: core.SessionFinishQuestion,
				Data: SessionEventTimeData{
					SnapshotTime: time.Now(),
					SessionID:    s.GetID(),
				},
			})
		case <-s.ctx.Done():
		}

	case <-s.ctx.Done():
		// session cancelled, exit gracefully
		return
	}
}

// =============================================================================
// EventManager methods
// =============================================================================

func (s *SimpleAgentSession) OnEvent() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.eventChans[core.SessionEventStart]:
			// handle session start event
			s.startTime = time.Now()
			if s.Logger != nil {
				s.Logger.Printf("session %s started", s.GetID())
			}
		case <-s.eventChans[core.SessionEventStop]:
			// handle session stop event
			// TODO: send session duration time to benchmarker
			// duration := time.Since(s.startTime)
			if s.Logger != nil {
				s.Logger.Printf("session %s stopped", s.GetID())
			}
		case e := <-s.eventChans[core.SessionHistory]:
			// handle session answer generated event
			log.Printf("Session %s: received SessionHistory event", s.GetID())
			history, ok := e.GetData().(core.ReActMessage)
			if ok {
				//  save the history to storage for future use
				log.Printf("Session %s saving history: %s", s.GetID(), history.ToString())
				if diag := s.SaveHistory(history); diag.Code != 0 {
					log.Printf("Session %s: save history error: %s", s.GetID(), diag.Message)
				}
				if s.Logger != nil {
					s.Logger.Printf("session %s final answer: %v", s.GetID(), history.ToString())
				}
			} else {
				log.Printf("Session %s: failed to cast event data to Conversation", s.GetID())
			}
		case <-s.eventChans[core.SessionStartProcessQuestion]:
			// handle session start process question event
			s.startProcessTime = time.Now()
			if s.Logger != nil {
				s.Logger.Printf("session %s started processing question", s.GetID())
			}
		case <-s.eventChans[core.SessionFinishQuestion]:
			// handle session start process question event
			// TODO: send session duration time to benchmarker
			// duration := time.Since(s.startProcessingTime)
			if s.Logger != nil {
				s.Logger.Printf("session %s finished processing question", s.GetID())
			}
		}
	}
}

func (s *SimpleAgentSession) GetEventChans() map[string]chan core.Event[any] {
	return s.eventChans
}

// RegisterEventChans registers all session event channels
func (s *SimpleAgentSession) RegisterEventChans() {
	s.eventChans[core.SessionEventStart] = make(chan core.Event[any], DefaultEventBufferSize)
	s.eventChans[core.SessionEventStop] = make(chan core.Event[any], DefaultEventBufferSize)
	s.eventChans[core.SessionEventSaveHistoryFailed] = make(chan core.Event[any], DefaultEventBufferSize)
	s.eventChans[core.SessionHistory] = make(chan core.Event[any], DefaultEventBufferSize)
	//  register missing event channels
	s.eventChans[core.SessionStartProcessQuestion] = make(chan core.Event[any], DefaultEventBufferSize)
	s.eventChans[core.SessionFinishQuestion] = make(chan core.Event[any], DefaultEventBufferSize)
}

// Observable (implements core.Observable interface)
func (a *SimpleAgentSession) GetBenchmarkListeningChannels() map[string]chan core.StatEvent[any] {
	return a.benchmarkerChans
}

func (a *SimpleAgentSession) CloseBenchmarkListeningChannels() {
	for key, ch := range a.benchmarkerChans {
		close(ch)
		delete(a.benchmarkerChans, key)
	}
}

// =============================================================================
// WorkFlow methods
// =============================================================================

func (s *SimpleAgentSession) BeforeStart(config core.AgentCoreConfig) []core.Diagnostic {
	return nil
}

func (s *SimpleAgentSession) Start(config core.AgentCoreConfig) []core.Diagnostic {
	// log when session starts listening
	log.Printf("Session %s Start: listening for questions, providers count = %d", s.GetID(), len(s.Providers))

	core.Emit(s, core.CommonEvent[SessionEventTimeData]{
		SourceType: core.SessionEventStart,
		Data: SessionEventTimeData{
			SnapshotTime: time.Now(),
			Message:      fmt.Sprintf("Session %s started", s.GetID()),
			SessionID:    s.GetID(),
		},
	})

	for {
		select {
		// check if channel is closed to avoid goroutine leak when ctrl+c
		case request, ok := <-s.Question:
			if !ok {
				log.Printf("Session %s: Question channel closed, exiting", s.GetID())
				return nil
			}
			log.Printf("Session %s: received query, calling ProcessQuery", s.GetID())
			s.ProcessQuery(request)
		case <-s.ctx.Done():
			return []core.Diagnostic{
				{
					Level:   core.SeverityWarn,
					Code:    core.MessageCodeSessionStopError,
					Message: fmt.Sprintf("Session %s stopped due to context cancellation", s.GetID()),
				},
			}
		}
	}
}

func (s *SimpleAgentSession) BeforeStop(config core.AgentCoreConfig) []core.Diagnostic {
	// process hook
	return nil
}

func (s *SimpleAgentSession) Stop(config core.AgentCoreConfig) []core.Diagnostic {
	// 1. stop all sub sessions
	diagnostics := []core.Diagnostic{}
	for _, subSession := range s.SubSessions {
		if err := core.StopSession(subSession, config); len(err) > 0 {
			diagnostics = append(diagnostics, err...)
		}
	}

	if len(diagnostics) > 0 {
		// high severity error, need return
		return []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSessionStopError,
				Message: fmt.Sprintf("Failed to stop sub sessions of session %s", s.GetID()),
			},
		}
	}

	// 2. close channels
	close(s.Question)
	close(s.mu)
	// 3. cancel goroutine
	s.cancel()

	return nil
}
