// auto-generated: test cases for core.Session interface and related functions
package test

import (
	"testing"

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

func TestMockSession_GetModelProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSession.EXPECT().GetModelProviders().Return(nil)

	providers := mockSession.GetModelProviders()
	if providers != nil {
		t.Errorf("expected nil providers, got %v", providers)
	}
}

func TestMockSession_GetConversations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockConversations := map[string]core.Conversation{
		"conv-1": nil,
	}

	mockSession.EXPECT().GetConversations().Return(mockConversations)

	conversations := mockSession.GetConversations()
	if len(conversations) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(conversations))
	}
}

func TestMockSession_SelectTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return(nil)

	tools := mockSession.SelectTools(mockQuestion, core.ReActMessage{})
	if tools != nil {
		t.Errorf("expected nil tools, got %v", tools)
	}
}

// test for ProcessQuery method
func TestMockSession_ProcessQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)

	mockSession.EXPECT().ProcessQuery(mockQuestion)

	mockSession.ProcessQuery(mockQuestion)
}

// test for SelectLocalKB method
func TestMockSession_SelectLocalKB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)

	mockSession.EXPECT().SelectLocalKB(mockQuestion).Return("local_kb_content")

	result := mockSession.SelectLocalKB(mockQuestion)
	if result != "local_kb_content" {
		t.Errorf("expected 'local_kb_content', got %s", result)
	}
}

func TestMockSession_NewSubSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockSubSession := testmock.NewMockSession(ctrl)

	mockSession.EXPECT().NewSubSession().Return(mockSubSession)

	subSession := mockSession.NewSubSession()
	if subSession == nil {
		t.Error("expected sub session, got nil")
	}
}

// =============================================================================
// Conversation interface tests
// =============================================================================

func TestMockConversation_GetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConv := testmock.NewMockConversation(ctrl)
	mockConv.EXPECT().GetID().Return("conv-123")

	id := mockConv.GetID()
	if id != "conv-123" {
		t.Errorf("expected id 'conv-123', got %s", id)
	}
}

func TestMockConversation_GetContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConv := testmock.NewMockConversation(ctrl)
	mockContent := testmock.NewMockSerializable(ctrl)
	mockContent.EXPECT().ToString().Return("test content")

	mockConv.EXPECT().GetContent().Return(mockContent)

	content := mockConv.GetContent()
	if content.ToString() != "test content" {
		t.Errorf("expected content 'test content', got %s", content.ToString())
	}
}

func TestMockConversation_SaveConversation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConv := testmock.NewMockConversation(ctrl)
	mockContent := testmock.NewMockSerializable(ctrl)

	mockConv.EXPECT().SaveConversation(mockContent).Return(nil)

	err := mockConv.SaveConversation(mockContent)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
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
