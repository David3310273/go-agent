// auto-add: regression tests for SimpleAgentSession.NewSubSession.
// Verifies the sub-session is usable on its own: inherited config, non-nil query context,
// working lock, own logger and event listener, isolation from the parent context, and
// registration into the parent so Stop shuts it down.
package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// newTestParentSession builds a real SimpleAgentSession on top of a mocked AgentCore.
// auto-add: log and memory paths point into t.TempDir so the test writes nothing into the repo.
func newTestParentSession(t *testing.T, ctrl *gomock.Controller, providers ...core.Provider) *simple.SimpleAgentSession {
	t.Helper()

	tmpDir := t.TempDir()
	mockAgent := testmock.NewMockAgentCore(ctrl)
	sessionConfig := core.SessionConfig{
		RootPath:             repoRootPath,
		ReActMaxRounds:       7,
		MemoryFilePathFormat: tmpDir + "/memories/%s.jsonl",
		MemoryFileSplitter:   "\n",
		MemoryWindowSize:     10,
		PromptFileMaxSize:    1024,
		LogPath:              tmpDir + "/logs/session_%s.log",
	}

	mockAgent.EXPECT().GetSessionConfig().Return(sessionConfig).AnyTimes()
	mockAgent.EXPECT().GetModelProviders().Return(providers).AnyTimes()
	mockAgent.EXPECT().GetHistory().Return(nil).AnyTimes()
	mockAgent.EXPECT().GetPrompt().Return([]byte("parent prompt")).AnyTimes()
	mockAgent.EXPECT().GetSkills().Return(nil).AnyTimes()
	mockAgent.EXPECT().GetKnowledgeBase().Return(nil).AnyTimes()
	mockAgent.EXPECT().GetToolsConfig().Return([]core.ToolConfig{
		{Name: "ParentTool"},
		{Name: core.SubSessionToolName},
	}).AnyTimes()
	mockAgent.EXPECT().GetMCPServerConfigs().Return(nil).AnyTimes()
	mockAgent.EXPECT().GetMCPClients().Return(nil).AnyTimes()

	return simple.NewAgentSession(mockAgent, "parent-session", nil)
}

// TestNewSubSession_InheritsRuntimeConfig covers Config / lockTimeOut / queryCtx,
// all of which were zero valued before and made a nested reAct loop unusable.
func TestNewSubSession_InheritsRuntimeConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	sub := parent.NewSubSession(nil)

	if sub == nil {
		t.Fatal("expected sub session, got nil")
	}
	if sub.GetID() == parent.GetID() {
		t.Errorf("expected a new ID, got the parent ID %s", sub.GetID())
	}

	// Config was a zero value, so ReActMaxRounds was 0 and the reAct loop never ran
	if got := sub.GetConfigs().ReActMaxRounds; got != 7 {
		t.Errorf("expected inherited ReActMaxRounds 7, got %d", got)
	}
	if got := sub.GetConfigs().RootPath; got == "" {
		t.Error("expected inherited RootPath, got empty string")
	}

	// queryCtx was nil, ProcessQuestion calls Done on it before entering the loop
	if sub.GetQueryCtx() == nil {
		t.Error("expected a non-nil query context")
	}

	// lockTimeOut was 0, so Acquire raced against time.After(0) and failed randomly
	for i := 0; i < 20; i++ {
		if diag := sub.Acquire(); diag != nil {
			t.Fatalf("Acquire failed on attempt %d: %s", i, diag.Message)
		}
		if diag := sub.Release(); diag != nil {
			t.Fatalf("Release failed on attempt %d: %s", i, diag.Message)
		}
	}

	// sub-session fields are isolated from parent
	_ = sub.(*simple.SimpleAgentSession)
}

// TestNewSubSession_IsolatedFromParent covers the reference typed fields that the
// Context value copy used to share with the parent.
func TestNewSubSession_IsolatedFromParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)

	// seed parent state that must not leak into the sub-session
	*parent.GetConversation() = append(*parent.GetConversation(), core.ReActMessage{
		Role:    core.RoleUser,
		Content: "parent conversation",
	})
	// auto-add: the confirm state is keyed by the tool call ID now, not by server and tool name
	parent.SetToolConfirmed("parent-call", "Yes")
	parent.SetPendingMCPToolCall("parent-call", &core.PendingMCPToolCall{ToolCallID: "parent-call"})

	sub := parent.NewSubSession(nil)

	// Conversation is empty by design
	if got := len(*sub.GetConversation()); got != 0 {
		t.Errorf("expected empty sub-session conversation, got %d messages", got)
	}

	// confirmedTools is empty by design, the sub-session confirms on its own
	if sub.IsToolConfirmed("parent-call") {
		t.Error("expected the sub-session not to inherit the parent confirmed tools")
	}

	// pendingMCPCalls was a nil map, writing to it panicked
	sub.SetPendingMCPToolCall("sub-call", &core.PendingMCPToolCall{ToolCallID: "sub-call"})
	if pending := sub.GetPendingMCPToolCall("sub-call"); pending == nil || pending.ToolCallID != "sub-call" {
		t.Error("expected the sub-session to store its own pending tool call")
	}
	if pending := parent.GetPendingMCPToolCall("parent-call"); pending == nil || pending.ToolCallID != "parent-call" {
		t.Error("expected the parent pending tool call to be untouched")
	}

	// the tool config is independent, the sub-session does not inherit the parent's tool config
	tools := sub.GetContext().GetToolsConfig()
	if len(tools) != 0 {
		t.Errorf("expected empty sub-session tool config, got %v", tools)
	}
	if got := len(parent.GetContext().GetToolsConfig()); got != 2 {
		t.Errorf("expected the parent tool config to keep both entries, got %d", got)
	}
	if diag := sub.GetContext().SetToolsConfig([]core.ToolConfig{{Name: "SubTool"}}); diag != nil {
		t.Errorf("SetToolsConfig returned diagnostic: %s", diag.Message)
	}
	if got := len(parent.GetContext().GetToolsConfig()); got != 2 {
		t.Errorf("expected the parent tool config to stay at 2 entries, got %d", got)
	}
}

// TestNewSubSession_LoadedToolsMapNotShared covers the shared dedup map: the sub-session
// used to write into it, which made the parent skip the tool in SetLoadTools and then
// fail to find it by name during the reAct loop.
func TestNewSubSession_LoadedToolsMapNotShared(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	sub := parent.NewSubSession(nil)

	mockTool := testmock.NewMockTool(ctrl)
	mockTool.EXPECT().GetName().Return("SharedNameTool").AnyTimes()

	sub.SetLoadTools(mockTool)
	if got := len(*sub.GetLoadTools()); got != 1 {
		t.Fatalf("expected 1 tool loaded in the sub-session, got %d", got)
	}

	// same tool name loaded into the parent afterwards must not be deduplicated away
	parent.SetLoadTools(mockTool)
	if got := len(*parent.GetLoadTools()); got != 1 {
		t.Errorf("expected 1 tool loaded in the parent, got %d (dedup map is still shared)", got)
	}
}

// TestNewSubSession_OwnEventListener makes sure the sub-session drains its own event
// channels instead of silently dropping them, and that Emit does not block.
func TestNewSubSession_OwnEventListener(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	sub := parent.NewSubSession(nil)

	if got := len(sub.GetEventChans()); got == 0 {
		t.Fatal("expected the sub-session to register its own event channels")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		core.Emit(sub, core.CommonEvent[core.ReActMessage]{
			SourceType: core.SessionHistory,
			Data:       core.ReActMessage{Role: core.RoleUser, Content: "sub-session history"},
		})
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Emit blocked, the sub-session event listener is not draining its channels")
	}

	// OnEvent handles the event asynchronously and SaveMemory writes it into the session
	// memory file. Wait for that write instead of returning while it is still in flight.
	memoryFile := fmt.Sprintf(parent.GetConfigs().MemoryFilePathFormat, sub.GetID())
	if !waitForFile(t, memoryFile, 3*time.Second) {
		t.Errorf("expected the sub-session listener to save history into %s", memoryFile)
	}

	// stop before the test ends so the listener goroutine is not writing while
	// t.TempDir cleanup removes the directory
	if diagnostics := core.StopSession(parent, core.AgentCoreConfig{}); len(diagnostics) > 0 {
		t.Errorf("expected no diagnostics from StopSession, got %v", diagnostics)
	}
}

// waitForFile polls until the file exists and is not empty, or the timeout expires.
// auto-add: helper for asserting on files written by an async event listener.
func waitForFile(t *testing.T, filename string, timeout time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(filename); err == nil && info.Size() > 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// TestNewSubSession_RegisteredAndStoppedByParent covers registration into SubSessions
// and the shutdown chain: parent Stop must close the sub-session channels without
// panicking and must cancel its context.
func TestNewSubSession_RegisteredAndStoppedByParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	sub := parent.NewSubSession(nil)

	if got := len(parent.SubSessions); got != 1 {
		t.Fatalf("expected 1 registered sub-session, got %d", got)
	}
	if parent.SubSessions[0].GetID() != sub.GetID() {
		t.Error("expected the registered sub-session to be the one just created")
	}

	diagnostics := core.StopSession(parent, core.AgentCoreConfig{})
	if len(diagnostics) > 0 {
		t.Errorf("expected no diagnostics from StopSession, got %v", diagnostics)
	}

	// sub-session context derives from the parent one, so it is cancelled as well
	select {
	case <-sub.GetQueryCtx().Done():
		if err := sub.GetQueryCtx().Err(); err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("expected the sub-session context to be cancelled by the parent Stop")
	}
}

// TestSession_LockUsableAfterStop covers Stop no longer closing mu: Acquire used to panic
// with "send on closed channel" on a stopped session, because a send on a closed channel is
// always ready and the select in Acquire picks it over the timeout branch.
func TestSession_LockUsableAfterStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	if diagnostics := core.StopSession(parent, core.AgentCoreConfig{}); len(diagnostics) > 0 {
		t.Fatalf("expected no diagnostics from StopSession, got %v", diagnostics)
	}

	// must return normally instead of panicking on the closed lock channel
	if diag := parent.Acquire(); diag != nil {
		t.Errorf("expected Acquire to succeed on a stopped session, got %s", diag.Message)
	}
	if diag := parent.Release(); diag != nil {
		t.Errorf("expected Release to succeed on a stopped session, got %s", diag.Message)
	}
}

// TestNewSubSession_NotRegisteredAfterParentStop covers the ctx pre-check in NewSubSession:
// a sub-session created once the parent is stopping would never be shut down by it.
func TestNewSubSession_NotRegisteredAfterParentStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	if diagnostics := core.StopSession(parent, core.AgentCoreConfig{}); len(diagnostics) > 0 {
		t.Fatalf("expected no diagnostics from StopSession, got %v", diagnostics)
	}

	late := parent.NewSubSession(nil)
	if late == nil {
		t.Fatal("expected a sub-session, got nil")
	}
	if got := len(parent.SubSessions); got != 0 {
		t.Errorf("expected no sub-session registered after the parent stopped, got %d", got)
	}
	if late.GetQueryCtx().Err() != context.Canceled {
		t.Error("expected the late sub-session context to inherit the cancelled parent context")
	}
}

// =============================================================================
// RunSubSession tests
// =============================================================================

// TestRunSubSession_RejectsNestedSubSession covers the one level limit: a sub-session must not
// be able to create another one.
func TestRunSubSession_RejectsNestedSubSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)
	sub := parent.NewSubSession(nil)

	result, diag := simple.SimpleHarnessInstance.RunSubSession(sub, map[string]any{
		"query": "nested task",
	}, "")

	if diag == nil || diag.Level != core.SeverityError {
		t.Fatalf("expected an error diagnostic, got result %q and diag %v", result, diag)
	}
	if result != "" {
		t.Errorf("expected an empty result, got %q", result)
	}
	if got := len(parent.SubSessions); got != 1 {
		t.Errorf("expected no extra sub-session to be created, got %d registered", got)
	}
}

// TestRunSubSession_RequiresQuery covers the query argument validation, the field that carries
// the task the sub-session has to work on.
func TestRunSubSession_RequiresQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)

	result, diag := simple.SimpleHarnessInstance.RunSubSession(parent, map[string]any{}, "")

	if diag == nil || diag.Level != core.SeverityError {
		t.Fatalf("expected an error diagnostic, got result %q and diag %v", result, diag)
	}
	if got := len(parent.SubSessions); got != 0 {
		t.Errorf("expected no sub-session to be created, got %d registered", got)
	}
}

// TestRunSubSession_RunsSubSessionAndReturnsAnswer is the end to end path: the harness creates
// the sub-session, drives its own ProcessQuery against a mocked provider, and returns the
// answer text that becomes the tool result in the parent conversation.
func TestRunSubSession_RunsSubSessionAndReturnsAnswer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reasoning := "sub-session reasoning"
	mockProvider := testmock.NewMockProvider(ctrl)
	mockProvider.EXPECT().GetName().Return("mock").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"mock"}}).AnyTimes()
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message: &core.ResponseMessage{
					Role:             core.RoleAssistant,
					Content:          "sub-session answer",
					ReasoningContent: &reasoning,
				},
			},
		},
	}, []core.Diagnostic{}).AnyTimes()

	parent := newTestParentSession(t, ctrl, mockProvider)

	// "mock" matches the provider name, so this also proves the model string is used to pick
	// the provider instead of falling back to the first one
	// auto-add: the sub-session thinking is printed to the standard output, so capture it around
	// the call. RunSubSession waits for its drainer before returning, everything is written by
	// the time the call is back.
	stdout := os.Stdout
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create the stdout pipe: %v", err)
	}
	os.Stdout = pipeWriter

	result, diag := simple.SimpleHarnessInstance.RunSubSession(
		parent,
		map[string]any{
			"query": "do the sub task",
			// auto-add: a sub-session runs only with the tools explicitly specified, RunSubSession
			// now rejects an empty tool list. The provider answers with a stop on the first round,
			// so the tool is loaded but never executed.
			"tools": []any{map[string]any{"name": "SearchKnowledgeBase"}},
		},
		"mock",
	)

	// auto-add: restore the real stdout before asserting, so a failure still prints to the terminal
	os.Stdout = stdout
	if err := pipeWriter.Close(); err != nil {
		t.Fatalf("failed to close the stdout pipe: %v", err)
	}
	capturedStdout, err := io.ReadAll(pipeReader)
	if err != nil {
		t.Fatalf("failed to read the stdout pipe: %v", err)
	}

	if diag != nil {
		t.Fatalf("expected no diagnostic, got %s", diag.Message)
	}
	if result != "sub-session answer" {
		t.Errorf("expected the sub-session answer, got %q", result)
	}

	// the sub-session is registered so the parent Stop shuts it down
	if got := len(parent.SubSessions); got != 1 {
		t.Fatalf("expected 1 registered sub-session, got %d", got)
	}
	sub := parent.SubSessions[0]

	// it ran its own conversation instead of writing into the parent one
	if got := len(*sub.GetConversation()); got == 0 {
		t.Error("expected the sub-session to have its own conversation")
	}
	if got := len(*parent.GetConversation()); got != 0 {
		t.Errorf("expected the parent conversation to stay empty, got %d messages", got)
	}

	// CreateSubSession was filtered out, so the sub-session cannot create another one
	for _, toolConfig := range sub.GetContext().GetToolsConfig() {
		if toolConfig.Name == core.SubSessionToolName {
			t.Error("expected CreateSubSession to be filtered out of the sub-session tools")
		}
	}

	// the sub-session persisted its own history: system prompt, user query, final answer
	memoryFile := fmt.Sprintf(parent.GetConfigs().MemoryFilePathFormat, sub.GetID())
	if !waitForFileLines(t, memoryFile, 3, 3*time.Second) {
		t.Errorf("expected at least 3 history lines in %s", memoryFile)
	}

	// auto-add: thinking is always on for a sub-session and goes to the standard output, tagged
	// with the sub-session ID, never to the parent hint channel
	if !strings.Contains(string(capturedStdout), reasoning) {
		t.Errorf("expected the sub-session thinking %q in the standard output, got %q", reasoning, string(capturedStdout))
	}
	wantPrefix := fmt.Sprintf("[sub-session %s][thinking]", sub.GetID())
	if !strings.Contains(string(capturedStdout), wantPrefix) {
		t.Errorf("expected the thinking line to carry the prefix %q, got %q", wantPrefix, string(capturedStdout))
	}

	if diagnostics := core.StopSession(parent, core.AgentCoreConfig{}); len(diagnostics) > 0 {
		t.Errorf("expected no diagnostics from StopSession, got %v", diagnostics)
	}
}

// waitForFileLines polls until the file holds at least want lines, or the timeout expires.
// auto-add: the sub-session event listener saves history asynchronously, so tests have to wait
// for the writes instead of racing with the t.TempDir cleanup.
func waitForFileLines(t *testing.T, filename string, want int, timeout time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if content, err := os.ReadFile(filename); err == nil && strings.Count(string(content), "\n") >= want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// TestRunSubSession_FallsBackToDefaultAnswer covers the answer extraction fallback: with no
// provider available the sub-session exhausts its reAct rounds, and the harness default answer
// that ProcessQuery wraps carries its text in Response instead of Choices.
func TestRunSubSession_FallsBackToDefaultAnswer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// no provider on purpose, so AskQuestion returns nothing usable
	parent := newTestParentSession(t, ctrl)

	_, diag := simple.SimpleHarnessInstance.RunSubSession(
		parent,
		map[string]any{
			"query": "do the sub task",
			// auto-add: a sub-session runs only with the tools explicitly specified, RunSubSession
			// now rejects an empty tool list. No provider is registered, so the tool is never called.
			"tools": []any{map[string]any{"name": "SearchKnowledgeBase"}},
		},
		"",
	)
	// when no provider is available, AskQuestion returns a diagnostic error and RunSubSession
	// wraps it into a tool error. The test name is historical: it used to fall back to the
	// default answer without diagnostic, now it fails with one.
	if diag == nil {
		t.Fatalf("expected a diagnostic when no provider is available, got nil")
	}
	// the sub-session is still registered even though it failed

	if got := len(parent.SubSessions); got != 1 {
		t.Fatalf("expected 1 registered sub-session, got %d", got)
	}
	sub := parent.SubSessions[0]

	// when no provider is available, the sub-session writes only the system prompt and user query
	// to history before failing, not the full reAct round retries
	memoryFile := fmt.Sprintf(parent.GetConfigs().MemoryFilePathFormat, sub.GetID())
	if !waitForFileLines(t, memoryFile, 2, 5*time.Second) {
		t.Errorf("expected at least 2 history lines in %s", memoryFile)
	}

	if diagnostics := core.StopSession(parent, core.AgentCoreConfig{}); len(diagnostics) > 0 {
		t.Errorf("expected no diagnostics from StopSession, got %v", diagnostics)
	}
}

// repoRootPath resolves the project root from the test package directory, the schema files are
// read as path.Join(rootPath, "agent/simple/tools", name).
const repoRootPath = ".."

// TestCreateSubSessionTool_DeclaresIntentOnly covers the tool that triggers a sub-session: it
// must be registered so its schema reaches the model, and it must not create anything itself.
func TestCreateSubSessionTool_DeclaresIntentOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := newTestParentSession(t, ctrl)

	tool := core.CreateTool(core.SubSessionToolName, repoRootPath, parent)
	if tool == nil {
		t.Fatal("expected CreateSubSession to be registered")
	}

	// the name has to match the constant core.ProcessQuestion intercepts by
	if tool.GetName() != core.SubSessionToolName {
		t.Errorf("expected tool name %s, got %s", core.SubSessionToolName, tool.GetName())
	}
	if tool.IsDestructive() {
		t.Error("expected CreateSubSession not to be destructive")
	}
	if tool.GetDescription() == "" {
		t.Error("expected a description from the schema, the model relies on it")
	}

	// the schema must declare query as required, otherwise the model cannot know what to pass
	schema := tool.GetSchema()
	required, ok := schema.Function.Parameters["required"].([]any)
	if !ok {
		t.Fatalf("expected a required list in the schema, got %v", schema.Function.Parameters["required"])
	}
	found := false
	for _, name := range required {
		if name == "query" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected query to be required, got %v", required)
	}

	// Validate rejects a missing query and accepts a real one
	if diag := tool.Validate(map[string]any{}); diag == nil || diag.Level != core.SeverityError {
		t.Errorf("expected an error diagnostic for a missing query, got %v", diag)
	}
	if diag := tool.Validate(map[string]any{"query": "do the sub task"}); diag != nil {
		t.Errorf("expected no diagnostic for a valid query, got %s", diag.Message)
	}

	// the runner must not create a sub-session, the harness owns that
	result, diag := core.CallTool(tool, map[string]any{"query": "do the sub task"}, false)
	if diag == nil || diag.Level != core.SeverityError {
		t.Errorf("expected the runner to report that the harness handles it, got %q and %v", result, diag)
	}
	if result != "" {
		t.Errorf("expected an empty result from the runner, got %q", result)
	}
	if got := len(parent.SubSessions); got != 0 {
		t.Errorf("expected the runner not to create a sub-session, got %d registered", got)
	}

	// LoadTools has to pick it up from the tool config, otherwise its schema never reaches
	// the model and the interception in core.ProcessQuestion can never be hit
	simple.SimpleHarnessInstance.LoadTools("", parent)
	loadedTools := parent.GetLoadTools()
	if got := len(*loadedTools); got != 1 {
		t.Fatalf("expected 1 loaded tool, got %d", got)
	}
	if got := (*loadedTools)[0].GetName(); got != core.SubSessionToolName {
		t.Errorf("expected %s to be loaded, got %s", core.SubSessionToolName, got)
	}
}
