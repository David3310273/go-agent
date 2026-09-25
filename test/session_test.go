// test cases for core.Session interface and related functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockSession satisfies core.Session
var _ core.Session = (*testmock.MockSession)(nil)

// =============================================================================
// Session interface tests
// =============================================================================

func TestMockSession_GetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSession.EXPECT().GetID().Return("session-123")

	id := mockSession.GetID()
	if id != "session-123" {
		t.Errorf("expected id 'session-123', got %s", id)
	}
}

func TestMockSession_GetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSession.EXPECT().GetStatus().Return(core.SessionStatusRunning)

	status := mockSession.GetStatus()
	if status != core.SessionStatusRunning {
		t.Errorf("expected status Running, got %d", status)
	}
}

func TestMockSession_SetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSession.EXPECT().SetStatus(core.SessionStatusIdle).Return(nil)

	err := mockSession.SetStatus(core.SessionStatusIdle)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockSession_GetConfigs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	expectedConfig := core.SessionConfig{
		ReActMaxRounds: 5,
	}

	mockSession.EXPECT().GetConfigs().Return(expectedConfig)

	config := mockSession.GetConfigs()
	if config.ReActMaxRounds != 5 {
		t.Errorf("expected ReActMaxRounds 5, got %d", config.ReActMaxRounds)
	}
}

func TestMockSession_GetConversation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	conversation := core.Conversation{
		{Role: core.RoleUser, Content: "hello"},
		{Role: core.RoleAssistant, Content: "hi"},
	}

	mockSession.EXPECT().GetConversation().Return(&conversation)

	result := mockSession.GetConversation()
	if result == nil {
		t.Fatal("expected conversation, got nil")
	}
	if len(*result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(*result))
	}
	if (*result)[0].Content != "hello" {
		t.Errorf("expected first message 'hello', got '%s'", (*result)[0].Content)
	}
}

func TestMockSession_ProcessQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)

	mockSession.EXPECT().ProcessQuery(mockQuestion)

	mockSession.ProcessQuery(mockQuestion)
}

func TestMockSession_NewSubSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSubSession := testmock.NewMockSession(ctrl)

	mockSession.EXPECT().NewSubSession(gomock.Any()).Return(mockSubSession)

	subSession := mockSession.NewSubSession([]core.Tool{})
	if subSession == nil {
		t.Error("expected sub session, got nil")
	}
}

// =============================================================================
// StartSession function tests
// =============================================================================

func TestStartSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	config := core.AgentCoreConfig{}

	gomock.InOrder(
		mockSession.EXPECT().BeforeStart(config).Return(nil),
		mockSession.EXPECT().Start(config).Return(nil),
	)

	diagnostics := core.StartSession(mockSession, config)
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestStartSession_BeforeStartError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	config := core.AgentCoreConfig{}
	expectedErr := []core.Diagnostic{
		{Level: core.SeverityError, Code: core.MessageCodeSessionCreateError},
	}

	mockSession.EXPECT().BeforeStart(config).Return(expectedErr)

	diagnostics := core.StartSession(mockSession, config)
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

// =============================================================================
// StopSession function tests
// =============================================================================

func TestStopSession_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	config := core.AgentCoreConfig{}

	gomock.InOrder(
		mockSession.EXPECT().BeforeStop(config).Return(nil),
		mockSession.EXPECT().Stop(config).Return(nil),
	)

	diagnostics := core.StopSession(mockSession, config)
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestStopSession_BeforeStopError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	config := core.AgentCoreConfig{}
	expectedErr := []core.Diagnostic{
		{Level: core.SeverityError, Code: core.MessageCodeSessionStopError},
	}

	mockSession.EXPECT().BeforeStop(config).Return(expectedErr)

	diagnostics := core.StopSession(mockSession, config)
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

// =============================================================================
// Config path resolution tests
// =============================================================================

// TestResolveConfigPath covers the root path resolution shared by the session log, the session
// memory file, the agent log and the conversation recovery.
// auto-add: an absolute configured path used to be joined with RootPath, and path.Join turns
// ".." + "/tmp/x/memories" into the relative "../tmp/x/memories", so the file was written
// outside the configured directory and nothing was ever found at the configured path.
func TestResolveConfigPath(t *testing.T) {
	cases := []struct {
		name       string
		rootPath   string
		configPath string
		want       string
	}{
		{
			name:       "relative config path is joined with the root path",
			rootPath:   "/app",
			configPath: "app/logs/session_%s.log",
			want:       "/app/app/logs/session_%s.log",
		},
		{
			name:       "relative config path with a relative root path",
			rootPath:   "..",
			configPath: "memories/%s.jsonl",
			want:       "../memories/%s.jsonl",
		},
		{
			name:       "absolute config path is kept as it is",
			rootPath:   "/app",
			configPath: "/tmp/dir/logs/session_%s.log",
			want:       "/tmp/dir/logs/session_%s.log",
		},
		{
			name:       "absolute config path survives a relative root path",
			rootPath:   "..",
			configPath: "/tmp/dir/memories/%s.jsonl",
			want:       "/tmp/dir/memories/%s.jsonl",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := utils.ResolvePath(testCase.rootPath, testCase.configPath); got != testCase.want {
				t.Errorf("expected %q, got %q", testCase.want, got)
			}
		})
	}
}
