package simple

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/benchmarker/metrics"
	"github.com/David3310273/go-agent/core"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
	uuid "github.com/gofrs/uuid/v5"
)

var _ core.Session = (*SimpleAgentSession)(nil)

const (
	LockTimeout = 3 * time.Second
	HistoryPath = "history.json"
)

// SimpleSessionContext holds session-specific context data
type SimpleSessionContext struct {
	*SimpleAgentContext

	Memory core.Conversation

	// loaded tool management
	RuntimeToolsMap map[string]core.Tool
	RuntimeTools    []core.Tool
}

// SimpleAgentSession represents a single agent session
/*
	- only one question can be answered, no follow-up questions nor steer mode.
*/
type SimpleAgentSession struct {
	ID            string
	ParentSession *SimpleAgentSession
	subSessions   atomic.Pointer[[]*SimpleAgentSession]
	Status        core.SessionStatus

	stopOnce  sync.Once
	stopDiags []core.Diagnostic

	// lock
	mu          chan struct{}
	lockTimeOut time.Duration

	// start time
	startTime time.Time
	// query start process time
	startProcessTime time.Time

	Config core.SessionConfig

	// context
	ctx    context.Context
	cancel context.CancelFunc
	// query context for single ProcessQuery cancellation
	queryCtx    context.Context
	queryCancel context.CancelFunc
	// query context of the parent session, captured in NewSubSession. ProcessQuery
	// derives the sub-session query context from it so cancelling the parent query cancels
	// the sub-session query as well. Nil for top level sessions.
	parentQueryCtx context.Context

	// user input chan
	Question chan core.Question
	// becnchmark chan
	benchmarkerChans      map[core.MetricType]chan core.BenchmarkEvent[any]
	benchmarkerEventChans map[core.MetricType]chan core.BenchmarkEvent[any]
	// event chans
	eventChans map[string]chan core.Event[any]
	// session context
	Context SimpleSessionContext
	// track confirmed destructive tools, value is user's answer (Yes/No)
	confirmedTools map[string]string
	// store pending MCP tool calls waiting for user confirmation, key: "serverName:toolName"
	pendingCalls map[string]*core.ConfirmToolCall
}

// NewAgentSession creates a new session from an AgentCore
// log providers count when creating session
func NewAgentSession(agent core.AgentCore, sessionID string, memory *core.Conversation) *SimpleAgentSession {
	sessionConfig := agent.GetSessionConfig()
	//  propagate RootPath from agent to session config
	if a, ok := agent.(*SimpleAgent); ok {
		sessionConfig.RootPath = a.RootPath
	}

	session := &SimpleAgentSession{
		Status:      core.SessionStatusRunning,
		mu:          make(chan struct{}, 1),
		lockTimeOut: LockTimeout,
		Question:    make(chan core.Question, 1),
		Config:      sessionConfig,

		benchmarkerChans:      make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		benchmarkerEventChans: make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		eventChans:            make(map[string]chan core.Event[any]),
		// initialize confirmation tracking maps
		confirmedTools: make(map[string]string),
		pendingCalls:   make(map[string]*core.ConfirmToolCall),
	}

	session.subSessions.Store(&[]*SimpleAgentSession{})

	// use provided sessionID or generate new one
	if sessionID != "" {
		session.ID = sessionID
	} else {
		id, _ := uuid.NewV4()
		session.ID = id.String()
	}

	// init context
	session.ctx, session.cancel = context.WithCancel(context.Background())

	// register event channels and start event listener
	session.RegisterEventChans()

	// llm context init - use pointer to share MCP configs and clients with agent
	if agentCtx, ok := agent.(*SimpleAgent); ok {
		session.Context.SimpleAgentContext = &agentCtx.SimpleAgentContext
	}

	// preload memory history and system tools
	session.Context.Memory = core.Conversation{}
	if memory != nil {
		session.Context.Memory = *memory
	}

	if session.Context.SimpleAgentContext != nil {
		session.Context.RuntimeTools = session.Context.SimpleAgentContext.Tools
		session.Context.RuntimeToolsMap = make(map[string]core.Tool)
		for _, tool := range session.Context.RuntimeTools {
			session.Context.RuntimeToolsMap[tool.GetName()] = tool
		}
	} else {
		session.Context.RuntimeToolsMap = make(map[string]core.Tool)
		session.Context.RuntimeTools = []core.Tool{}
	}

	return session
}

// NewSubSession creates a child session
// inherit the parent runtime config and reset every reference-typed field, so the
// sub-session can run its own reAct loop with its own logger and event listener.
func (s *SimpleAgentSession) NewSubSession(tools []core.Tool) core.Session {
	// create a copy of the parent's SimpleAgentContext (not sharing the pointer)
	subSessionContext := *s.Context.SimpleAgentContext

	session := &SimpleAgentSession{
		ParentSession: s,
		Status:        core.SessionStatusRunning,
		mu:            make(chan struct{}, 1),
		// zero lockTimeOut made Acquire race against time.After(0) and fail randomly
		lockTimeOut: LockTimeout,
		// zero Config meant ReActMaxRounds 0, empty RootPath and empty memory file format
		Config: s.Config,
		// Stop closes Question, close of a nil channel panics
		Question: make(chan core.Question, 1),
		// nil maps make RegisterEventChans and SetPendingToolCall panic on assignment
		eventChans:            make(map[string]chan core.Event[any]),
		benchmarkerChans:      make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		benchmarkerEventChans: make(map[core.MetricType]chan core.BenchmarkEvent[any]),
		confirmedTools:        make(map[string]string),
		pendingCalls:          make(map[string]*core.ConfirmToolCall),
		// copy of the parent context (not sharing the pointer)
		Context: SimpleSessionContext{
			SimpleAgentContext: &subSessionContext,
		},
	}
	session.subSessions.Store(&[]*SimpleAgentSession{})

	// break the references still shared with the parent after the Context copy.
	// RuntimeToolsMap is a map, so the sub-session used to write into the parent dedup map and
	// made the parent skip tools in SetLoadTools.
	session.Context.RuntimeToolsMap = make(map[string]core.Tool)
	session.Context.RuntimeTools = []core.Tool{}
	session.Context.Memory = core.Conversation{}

	// only load tools from parent session
	for _, tool := range tools {
		session.SetLoadTools(tool)
	}

	// inherit parent cancellation so stopping the parent stops the sub-session
	session.ctx, session.cancel = context.WithCancel(s.ctx)
	// queryCtx was nil, ProcessQuestion calls Done on it before the reAct loop.
	// Derive it from the parent query context when there is one, so cancelling the parent
	// query cancels the sub-session query too. ProcessQuery derives from the same base.
	session.parentQueryCtx = s.queryCtx
	if session.parentQueryCtx == nil {
		session.parentQueryCtx = session.ctx
	}
	session.queryCtx, session.queryCancel = context.WithCancel(session.parentQueryCtx)

	id, _ := uuid.NewV4()
	session.ID = id.String()

	// own event channels and listener, OnEvent returns once session.ctx is cancelled
	session.RegisterEventChans()
	go session.OnEvent()

	// register into the parent so its Stop can shut the sub-session down.
	// Uses atomic append (copy-on-write) so concurrent Stop can safely snapshot.
	// Skipped when the parent is already stopping.
	if s.ctx.Err() != nil {
		core.LogWarn(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] parent is stopping, sub-session=%s not registered", s.ID, session.ID)
	} else {
		for {
			// optimistic lock update
			old := s.subSessions.Load()
			newSlice := make([]*SimpleAgentSession, len(*old)+1)
			copy(newSlice, *old)
			newSlice[len(*old)] = session
			if s.subSessions.CompareAndSwap(old, &newSlice) {
				break
			}
		}
	}

	return session
}

// =============================================================================
// SessionContext methods
// =============================================================================

func (s *SimpleAgentSession) GetConfigs() core.SessionConfig {
	return s.Config
}

func (s *SimpleAgentSession) GetContext() core.Context {
	return &s.Context
}

func (s *SimpleAgentSession) GetQueryCtx() context.Context {
	return s.queryCtx
}

func (s *SimpleAgentSession) SetQueryContext(ctx context.Context, cancel context.CancelFunc) *core.Diagnostic {
	if diag := s.Acquire(); diag == nil {
		defer s.Release()
		s.queryCtx = ctx
		s.queryCancel = cancel
		return nil
	}

	return &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeSessionMultiQueryError,
		Message: "Failed to set query context while session is processing another query, please wait.",
	}
}

func (s *SimpleAgentSession) CancelQuery() {
	if s.queryCancel != nil {
		s.queryCancel()
	}
	// drain pending question from channel to unblock agent's blocked send
	select {
	case <-s.Question:
		core.LogStd(core.LogLevelInfo, "[session=%s] drained pending question on cancel", s.GetID())
	default:
	}
}

func (s *SimpleAgentSession) SetSessionHistory() core.Diagnostic {
	return core.Diagnostic{}
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

// IsToolConfirmed checks if user has answered for a destructive tool call (Yes or No)
// keyed by the tool call ID, a batch of destructive calls in one assistant message
// shares the same server and outer tool name, so that pair could not identify a single call
func (s *SimpleAgentSession) IsToolConfirmed(toolCallID string) bool {
	_, exists := s.confirmedTools[toolCallID]
	return exists
}

// SetToolConfirmed records user's answer for a destructive tool call
func (s *SimpleAgentSession) SetToolConfirmed(toolCallID string, answer string) {
	s.confirmedTools[toolCallID] = answer
}

func (s *SimpleAgentSession) ClearToolConfirmed(toolCallID string) {
	delete(s.confirmedTools, toolCallID)
}

// GetPendingToolCall retrieves pending MCP tool call info
func (s *SimpleAgentSession) GetPendingToolCall(toolCallID string) *core.ConfirmToolCall {
	return s.pendingCalls[toolCallID]
}

// SetPendingToolCall saves pending MCP tool call info
func (s *SimpleAgentSession) SetPendingToolCall(toolCallID string, pending *core.ConfirmToolCall) {
	s.pendingCalls[toolCallID] = pending
}

// DeletePendingToolCall deletes pending MCP tool call info after tool execution
func (s *SimpleAgentSession) DeletePendingToolCall(toolCallID string) {
	delete(s.pendingCalls, toolCallID)
}

func (s *SimpleSessionContext) GetModelProviders() []core.Provider {
	return s.ModelProviders
}

// return pointer so callers can modify the conversation in place
func (s *SimpleAgentSession) GetConversation() *core.Conversation {
	return &s.Context.Memory
}

// return pointer so callers can modify the conversation in place
func (s *SimpleAgentSession) GetLoadTools() *[]core.Tool {
	return &s.Context.RuntimeTools
}

func (s *SimpleAgentSession) SetLoadTools(tool core.Tool) {
	if _, ok := s.Context.RuntimeToolsMap[tool.GetName()]; !ok {
		s.Context.RuntimeToolsMap[tool.GetName()] = tool
		s.Context.RuntimeTools = append(s.Context.RuntimeTools, tool)
	}
}

func (s *SimpleAgentSession) DeleteMemory() {
	// resolve through ResolvePath so an absolute format is not made relative
	filePath := utils.ResolvePath(s.Config.RootPath, s.Config.MemoryFilePathFormat)
	filename := fmt.Sprintf(filePath, s.GetID())
	if err := os.Remove(filename); err != nil {
		// ignore error
		core.LogWarn(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] delete memory file failed: file=%s, error=%v", s.GetID(), filename, err)
	}
}

func (s *SimpleAgentSession) SaveMemory(memory core.ReActMessage) *core.Diagnostic {
	//  use RootPath instead of hardcoded relative path
	// resolve through ResolvePath so an absolute format is not made relative,
	// and keep it in sync with SimpleAgent.RecoverConversation which reads the same file
	filePath := utils.ResolvePath(s.Config.RootPath, s.Config.MemoryFilePathFormat)
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

	content := []byte(memory.ToString())
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

func (s *SimpleAgentSession) saveChatMessage(message core.ChatHistory) *core.Diagnostic {
	filePathFormat := utils.ResolvePath(s.Config.RootPath, s.Config.ChatHistoryFilePathFormat)
	filename := fmt.Sprintf(filePathFormat, s.GetID())
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

	content := []byte(message.ToString())
	content = append(content, '\n')

	if err := utils.RotateWrite(filename, s.Config.MemoryFileSplitter, 1024*1024*1024, content); err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeWriteHistoryError,
			Message: "save chat message failed: " + err.Error(),
		}
	}

	return &core.Diagnostic{}
}

func (s *SimpleAgentSession) ExportChatHistory(size int) ([]byte, *core.Diagnostic) {
	filePathFormat := utils.ResolvePath(s.Config.RootPath, s.Config.ChatHistoryFilePathFormat)
	filename := fmt.Sprintf(filePathFormat, s.GetID())
	chatBytes, err := utils.TailFile(filename, size)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeWriteHistoryError,
			Message: "export chat history failed: " + err.Error(),
		}
	}

	return chatBytes, nil
}

// select local kb given the question, merge into final context
// searches KB and converts results to MarkdownCollection via JSON marshal/unmarshal.
func (s *SimpleAgentSession) SelectLocalKB(query core.Question) string {
	var result strings.Builder

	for _, kbAny := range *s.Context.GetKnowledgeBase() {
		milvusKB, ok := kbAny.(*storage.MilvusKnowledgebase[any])
		if !ok {
			continue
		}

		searchResults, diag := milvusKB.Search(query.GetQuery(), 1, "")
		if diag != nil {
			continue
		}

		for _, res := range searchResults {
			result.WriteString(res.GetContent())
		}

		result.WriteString(s.Config.MemoryFileSplitter)
	}

	return result.String()
}

// SimpleNormalResponse is the standard response type for normal questions
type SimpleNormalResponse struct {
	Response  core.AgentResponse `json:"response"`
	SessionID string             `json:"sessionID"`
}

func (s SimpleNormalResponse) ToString() string {
	response, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(response)
}

// GetSessionID returns the session ID for SessionAnswer interface
func (s SimpleNormalResponse) GetSessionID() string {
	return s.SessionID
}

// ToolConfirmItem is one destructive tool call of a confirmation batch.
// the frontend echoes toolCallID and its Yes/No answer back in one request.
type ToolConfirmItem struct {
	ToolCallID string `json:"toolCallID"` // identifies the pending call in the answer
	ServerName string `json:"serverName"` // MCP server name, empty for a local tool
	ToolName   string `json:"toolName"`   // outer tool name (e.g., "UseMCPServerTools")
	MCPTool    string `json:"mcpTool"`    // inner MCP tool name, empty for a local tool
	Message    string `json:"message"`    // confirmation message for this call
}

// SimpleToolConfirmResponse is sent to the user when destructive tools need confirmation
// carries the whole batch instead of a single tool, every item is answered separately
type SimpleToolConfirmResponse struct {
	Response  core.AgentResponse `json:"response"`  // confirmation message from harness
	Confirms  []ToolConfirmItem  `json:"confirms"`  // the batch waiting for the user answers
	SessionID string             `json:"sessionID"` // session ID
}

func (r SimpleToolConfirmResponse) ToString() string {
	response, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(response)
}

// GetSessionID returns the session ID for SessionAnswer interface
func (r SimpleToolConfirmResponse) GetSessionID() string {
	return r.SessionID
}

// RunQuery executes a query synchronously and returns the final answer.
// Unlike ProcessQuery, this is a direct synchronous call without channels,
// streaming, or event emission. Designed for sub-sessions and internal use.
func (s *SimpleAgentSession) RunQuery(query core.Question) (core.Answer, []core.Diagnostic) {
	// set up query context for cancellation
	baseCtx := s.parentQueryCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	queryCtx, queryCancel := context.WithCancel(baseCtx)
	if diag := s.SetQueryContext(queryCtx, queryCancel); diag != nil {
		return SimpleNormalResponse{
			SessionID: s.GetID(),
			Response: core.AgentResponse{
				Response: diag.Message,
			},
		}, []core.Diagnostic{*diag}
	}
	defer queryCancel()

	// run reAct loop
	if query.GetStreaming() {
		return core.ProcessQuestionStream(s, query, SimpleHarnessInstance)
	}
	return core.ProcessQuestion(s, query, SimpleHarnessInstance)
}

// ProcessQuery handles a user question with full channel support, streaming, and events.
// Wraps RunQuery for user-facing scenarios.
func (s *SimpleAgentSession) ProcessQuery(query core.Question) {
	core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] question received: query_len=%d, streaming=%v, enable_thinking=%v",
		s.GetID(), len(query.GetQuery()), query.GetStreaming(), query.GetEnableThinking())

	// emit start event for benchmark
	core.Emit(s, core.CommonEvent[QuestionEventTimeData]{
		SourceType: core.SessionStartProcessQuestion,
		Data: QuestionEventTimeData{
			SnapshotTime:     time.Now(),
			SessionID:        s.GetID(),
			UserMsg:          query.GetQuery(),
			UserQuestionTime: query.GetSendTimestamp(),
			QueryID:          query.GetID(),
		},
	})

	if s.Config.EnableBenchmark {
		core.SendBenchmarkEvent(s.benchmarkerChans[core.MetricQueryTime], core.BenchmarkEvent[any]{
			TimeStamp: time.Now().UnixMilli(),
			Type:      core.MetricQueryTime,
			Data: metrics.QueryTimeMetric{
				SessionID: s.GetID(),
				QueryID:   query.GetID(),
				StartTime: time.UnixMilli(query.GetSendTimestamp()),
				EndTime:   time.Now(),
			},
		})
	}

	// run the query synchronously
	response, diagnostics := s.RunQuery(query)

	// determine what to send back; wrap non-confirm results in SimpleNormalResponse, but send
	// confirm responses directly so the stream consumer can type-switch on them.
	var finalResult core.Answer

	if len(diagnostics) > 0 {
		core.LogError(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] process question failed: diagnostics=%v", s.GetID(), diagnostics)
	}

	switch resp := response.(type) {
	case core.AgentResponse:
		finalResult = SimpleNormalResponse{
			SessionID: s.GetID(),
			Response:  resp,
		}
	case SimpleToolConfirmResponse:
		finalResult = resp
	default:
		finalResult = SimpleNormalResponse{
			SessionID: s.GetID(),
			Response:  SimpleHarnessInstance.GetDefaultAnswer(),
		}
	}

	// always send finalResult to responseChan so streaming handler can extract sessionID and usage
	// use goroutine to avoid deadlock when channel is full
	go func() {
		select {
		case query.GetResponseChan() <- finalResult:
			// finished process question
			if s.Config.EnableBenchmark {
				core.SendBenchmarkEvent(s.benchmarkerChans[core.MetricQueryTime], core.BenchmarkEvent[any]{
					TimeStamp: time.Now().UnixMilli(),
					Type:      core.MetricQueryTime,
					Data: metrics.QueryTimeMetric{
						SessionID: s.GetID(),
						QueryID:   query.GetID(),
						StartTime: time.UnixMilli(query.GetSendTimestamp()),
						EndTime:   time.Now(),
					},
				})
			}

		case <-time.After(30 * time.Second):
			// timeout to avoid deadlock
			core.LogStd(core.LogLevelWarn, "[session=%s] send finalAnswer timeout", s.GetID())
		}
	}()
}

// lock manager methods

func (s *SimpleAgentSession) Acquire() *core.Diagnostic {
	select {
	case s.mu <- struct{}{}:
		return nil
	case <-time.After(s.lockTimeOut):
		return &core.Diagnostic{
			Code:    core.MessageCodeLockError,
			Level:   core.SeverityError,
			Message: "Lock timeout",
		}
	}
}

func (s *SimpleAgentSession) Release() *core.Diagnostic {
	<-s.mu
	return nil
}

// =============================================================================
// EventManager methods
// =============================================================================

func (s *SimpleAgentSession) OnEvent() {
	// auto-add: benchmark pull channel moved to outer select so it can be
	// selected independently (fixes nested select with default).
	var benchmarkPullCh chan core.BenchmarkEvent[any]
	if s.Config.EnableBenchmark {
		benchmarkPullCh = s.benchmarkerEventChans[core.MetricSessionRunningTime]
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.eventChans[core.SessionEventStart]:
			// handle session start event
			s.startTime = time.Now()
			core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] started", s.GetID())
		case <-s.eventChans[core.SessionEventStop]:
			// handle session stop event
			core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] stopped", s.GetID())
		case e := <-s.eventChans[core.SessionHistory]:
			// handle session answer generated event
			history, ok := e.GetData().(core.ReActMessage)
			if ok {
				// save the history to storage for future use
				if diag := s.SaveMemory(history); diag.Code != 0 {
					core.LogStd(core.LogLevelError, "[session=%s] save history failed: code=%d, msg=%s", s.GetID(), diag.Code, diag.Message)
				} else {
					core.LogStd(core.LogLevelInfo, "[session=%s] answer generated: role=%s, content_len=%d",
						s.GetID(), history.Role, len(history.Content))
				}
				if history.Role == core.RoleAssistant && history.ToolCalls == nil {
					chatMessage := core.ChatHistory{
						Role:      history.Role,
						Content:   history.Content,
						Timestamp: time.Now().UnixMilli(),
					}
					if diag := s.saveChatMessage(chatMessage); diag.Code != 0 {
						core.LogStd(core.LogLevelError, "[session=%s] save chat message failed: code=%d, msg=%s", s.GetID(), diag.Code, diag.Message)
					} else {
						core.LogStd(core.LogLevelInfo, "[session=%s] chat message saved: role=%s, content_len=%d",
							s.GetID(), chatMessage.Role, len(chatMessage.Content))
					}
				}
			} else {
				core.LogStd(core.LogLevelWarn, "[session=%s] failed to cast event data to ReActMessage", s.GetID())
			}
		case e := <-s.eventChans[core.SessionStartProcessQuestion]:
			// handle session start process question event
			message, ok := e.GetData().(QuestionEventTimeData)
			if ok {
				s.startProcessTime = message.SnapshotTime

				userChatRecord := core.ChatHistory{
					Role:      core.RoleUser,
					Content:   message.UserMsg,
					Timestamp: message.UserQuestionTime,
				}

				if diag := s.saveChatMessage(userChatRecord); diag.Code != 0 {
					core.LogStd(core.LogLevelWarn, "[session=%s] save history failed: code=%d, msg=%s", s.GetID(), diag.Code, diag.Message)
				} else {
					core.LogStd(core.LogLevelInfo, "[session=%s] user message saved: role=%s, content=%s",
						s.GetID(), userChatRecord.Role, userChatRecord.Content)
				}
			}

			core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] processing question started at: %s", s.GetID(), s.startProcessTime.Format(time.RFC3339))
		case <-s.eventChans[core.SessionFinishQuestion]:
			// handle session finish question event
			core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] processing question finished at: %s", s.GetID(), time.Now().Format(time.RFC3339))
		case event := <-benchmarkPullCh:
			if event.GetSourceType() != string(core.MetricSessionRunningTime) {
				continue
			}
			core.SendBenchmarkEvent(s.benchmarkerChans[core.MetricSessionRunningTime], core.BenchmarkEvent[any]{
				Type:      core.MetricSessionRunningTime,
				TimeStamp: time.Now().UnixMilli(),
				Data: metrics.SessionRunningTimeMetric{
					SessionID: s.GetID(),
					StartTime: s.startTime,
					EndTime:   time.Now(),
				},
			})
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
	// register benchmark channels
	s.benchmarkerEventChans[core.MetricSessionRunningTime] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)

	s.benchmarkerChans[core.MetricSessionRunningTime] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
	s.benchmarkerChans[core.MetricQueryFailedCount] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
	s.benchmarkerChans[core.MetricQuerySuccessCount] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
	s.benchmarkerChans[core.MetricQueryTime] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
	s.benchmarkerChans[core.MetricQueryTokenUsage] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
	s.benchmarkerChans[core.MetricQueryReActRound] = make(chan core.BenchmarkEvent[any], DefaultEventBufferSize)
}

// Observable (implements core.Observable interface)
func (a *SimpleAgentSession) GetBenchmarkListeningChannels() map[core.MetricType]chan core.BenchmarkEvent[any] {
	return a.benchmarkerChans
}

func (a *SimpleAgentSession) GetBenchmarkPullChannels() map[core.MetricType]chan core.BenchmarkEvent[any] {
	return a.benchmarkerEventChans
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

func (s *SimpleAgentSession) BeforeStart() []core.Diagnostic {
	return nil
}

func (s *SimpleAgentSession) Start() []core.Diagnostic {
	core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] listening for questions", s.GetID())

	core.Emit(s, core.CommonEvent[SessionEventTimeData]{
		SourceType: core.SessionEventStart,
		Data: SessionEventTimeData{
			SnapshotTime: time.Now(),
			Message:      fmt.Sprintf("Session %s started", s.GetID()),
			SessionID:    s.GetID(),
		},
	})

	if s.Config.EnableBenchmark {
		core.SendBenchmarkEvent(s.benchmarkerChans[core.MetricSessionRunningTime], core.BenchmarkEvent[any]{
			Type:      core.MetricSessionRunningTime,
			TimeStamp: time.Now().UnixMilli(),
			Data: metrics.SessionRunningTimeMetric{
				SessionID: s.GetID(),
				StartTime: s.startTime,
				EndTime:   time.Now(),
			},
		})
	}

	// start event listener
	go s.OnEvent()

	for {
		select {
		// check if channel is closed to avoid goroutine leak when ctrl+c
		case request, ok := <-s.Question:
			if !ok {
				core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] question channel closed, exiting", s.GetID())
				return nil
			}
			core.LogInfo(utils.ResolvePath(s.Config.RootPath, fmt.Sprintf("%s/%s", SimpleAgentPath, fmt.Sprintf(s.GetConfigs().LogPath, s.GetID()))), "[session=%s] received query, dispatching", s.GetID())
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

func (s *SimpleAgentSession) BeforeStop() []core.Diagnostic {
	// process hook
	return nil
}

// snapshotSubSessions returns a copy of the sub-sessions slice via atomic load.
// Copy-on-write append in NewSubSession guarantees the returned snapshot is consistent.
func (s *SimpleAgentSession) snapshotSubSessions() []*SimpleAgentSession {
	old := s.subSessions.Load()
	subSessions := make([]*SimpleAgentSession, len(*old))
	copy(subSessions, *old)
	return subSessions
}

func (s *SimpleAgentSession) Stop() []core.Diagnostic {
	s.stopOnce.Do(func() {
		s.stopDiags = s.doStop()
	})

	if s.Config.EnableBenchmark {
		core.SendBenchmarkEvent(s.benchmarkerChans[core.MetricSessionRunningTime], core.BenchmarkEvent[any]{
			Type:      core.MetricSessionRunningTime,
			TimeStamp: time.Now().UnixMilli(),
			Data: metrics.SessionRunningTimeMetric{
				SessionID: s.GetID(),
				StartTime: s.startTime,
				EndTime:   time.Now(),
			},
		})
	}

	return s.stopDiags
}

func (s *SimpleAgentSession) doStop() []core.Diagnostic {
	// 1. stop all sub sessions
	diagnostics := []core.Diagnostic{}
	for _, subSession := range s.snapshotSubSessions() {
		if err := core.StopSession(subSession); len(err) > 0 {
			diagnostics = append(diagnostics, err...)
		}
	}

	if len(diagnostics) > 0 {
		return []core.Diagnostic{
			{
				Level:   core.SeverityError,
				Code:    core.MessageCodeSessionStopError,
				Message: fmt.Sprintf("Failed to stop sub sessions of session %s", s.GetID()),
			},
		}
	}

	// 2. close channels
	// mu is deliberately not closed. A closed mu makes every later Acquire panic
	// with "send on closed channel" instead of returning a lock timeout diagnostic, and the
	// shutdown signal is already carried by s.cancel below.
	close(s.Question)
	// 3. cancel goroutine
	s.cancel()

	return nil
}
