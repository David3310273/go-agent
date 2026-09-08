// auto-generated: test cases for core.AgentCore interface and related functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockAgentCore satisfies core.AgentCore
var _ core.AgentCore = (*testmock.MockAgentCore)(nil)

// =============================================================================
// LockManager interface tests
// =============================================================================

func TestMockLockManager_Acquire(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLock := testmock.NewMockLockManager(ctrl)
	mockLock.EXPECT().Acquire().Return(nil)

	err := mockLock.Acquire()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockLockManager_Release(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLock := testmock.NewMockLockManager(ctrl)
	mockLock.EXPECT().Release().Return(nil)

	err := mockLock.Release()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// =============================================================================
// UI interface tests
// =============================================================================

func TestMockUI_Render(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUI := testmock.NewMockUI(ctrl)
	mockUI.EXPECT().Render("hello world")

	mockUI.Render("hello world")
}

func TestMockUI_RenderSystemMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUI := testmock.NewMockUI(ctrl)
	diagnostics := []core.Diagnostic{
		{Level: core.SeverityError, Message: "error occurred"},
	}

	mockUI.EXPECT().RenderSystemMessage(diagnostics)

	mockUI.RenderSystemMessage(diagnostics)
}

func TestMockUI_Welcome(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUI := testmock.NewMockUI(ctrl)
	mockUI.EXPECT().Welcome()

	mockUI.Welcome()
}

func TestMockUI_GetUserInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUI := testmock.NewMockUI(ctrl)
	mockUI.EXPECT().GetUserInput().Return("user input")

	input := mockUI.GetUserInput()
	if input != "user input" {
		t.Errorf("expected 'user input', got %s", input)
	}
}

// =============================================================================
// Configure interface tests
// =============================================================================

func TestMockConfigure_GetConfigPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := testmock.NewMockConfigure(ctrl)
	mockConfig.EXPECT().GetConfigPath().Return("/path/to/config.json")

	path := mockConfig.GetConfigPath()
	if path != "/path/to/config.json" {
		t.Errorf("expected path '/path/to/config.json', got %s", path)
	}
}

// =============================================================================
// WorkFlow interface tests
// =============================================================================

func TestMockWorkFlow_BeforeStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)
	config := core.AgentCoreConfig{}

	mockWorkFlow.EXPECT().BeforeStart(config).Return(nil)

	diagnostics := mockWorkFlow.BeforeStart(config)
	if diagnostics != nil {
		t.Errorf("expected nil diagnostics, got %v", diagnostics)
	}
}

func TestMockWorkFlow_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)
	config := core.AgentCoreConfig{}

	mockWorkFlow.EXPECT().Start(config).Return(nil)

	diagnostics := mockWorkFlow.Start(config)
	if diagnostics != nil {
		t.Errorf("expected nil diagnostics, got %v", diagnostics)
	}
}

func TestMockWorkFlow_BeforeStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)
	config := core.AgentCoreConfig{}

	mockWorkFlow.EXPECT().BeforeStop(config).Return(nil)

	diagnostics := mockWorkFlow.BeforeStop(config)
	if diagnostics != nil {
		t.Errorf("expected nil diagnostics, got %v", diagnostics)
	}
}

func TestMockWorkFlow_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)
	config := core.AgentCoreConfig{}

	mockWorkFlow.EXPECT().Stop(config).Return(nil)

	diagnostics := mockWorkFlow.Stop(config)
	if diagnostics != nil {
		t.Errorf("expected nil diagnostics, got %v", diagnostics)
	}
}

// =============================================================================
// SessionManager interface tests
// =============================================================================

func TestMockSessionManager_StopSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := testmock.NewMockSessionManager(ctrl)
	mockSessionMgr.EXPECT().StopSession("session-123").Return(nil)

	err := mockSessionMgr.StopSession("session-123")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockSessionManager_GetSessionOnCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionMgr := testmock.NewMockSessionManager(ctrl)
	mockSession := testmock.NewMockSession(ctrl)

	mockSessionMgr.EXPECT().GetSessionOnCreate("session-123", true).Return(mockSession, nil)

	session, err := mockSessionMgr.GetSessionOnCreate("session-123", true)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if session == nil {
		t.Error("expected session, got nil")
	}
}

// =============================================================================
// AgentCore interface tests
// =============================================================================

func TestMockAgentCore_GetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := testmock.NewMockAgentCore(ctrl)
	mockAgent.EXPECT().GetID().Return("agent-123")

	id := mockAgent.GetID()
	if id != "agent-123" {
		t.Errorf("expected id 'agent-123', got %s", id)
	}
}

func TestMockAgentCore_SetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := testmock.NewMockAgentCore(ctrl)
	mockAgent.EXPECT().SetID().Return(nil)

	err := mockAgent.SetID()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// =============================================================================
// StartAgentCore function tests
// =============================================================================

func TestStartAgentCore_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := testmock.NewMockAgentCore(ctrl)
	appConfig := core.AppConfig{Language: core.Language_EN}
	agentConfig := core.AgentCoreConfig{
		Agent: core.AgentConfig{
			History: core.HistoryConfig{BufferSize: 1024},
		},
	}

	// auto-generated: updated LoadConfigs to new no-arg signature
	gomock.InOrder(
		mockAgent.EXPECT().LoadConfigs().Return(agentConfig),
		mockAgent.EXPECT().SetID().Return(nil),
		mockAgent.EXPECT().SetLogger(agentConfig.Agent).Return(nil),
		mockAgent.EXPECT().SetHistory(agentConfig.Agent.History).Return(nil),
		mockAgent.EXPECT().SetSkills(agentConfig.Agent.Skill).Return(nil),
		mockAgent.EXPECT().SetPrompt(agentConfig.Agent.Prompt).Return(nil),
		mockAgent.EXPECT().SetKnowledgeBase(agentConfig.Agent.KnowledgeBase).Return(nil),
		mockAgent.EXPECT().SetToolsConfig(agentConfig.Agent.Tool).Return(nil),
		mockAgent.EXPECT().BeforeStart(agentConfig).Return(nil),
		mockAgent.EXPECT().Start(agentConfig).Return(nil),
	)

	diagnostics := core.StartAgentCore(mockAgent, appConfig)
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

// auto-generated: LoadConfigs no longer returns error in StartAgentCore,
// config loading errors are now handled in NewSimpleAgent.
// This test now verifies StartAgentCore with SetID error.
func TestStartAgentCore_SetIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := testmock.NewMockAgentCore(ctrl)
	appConfig := core.AppConfig{}
	agentConfig := core.AgentCoreConfig{}
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeAgentCoreConfigError,
		Message: "Failed to generate UUID",
	}

	// auto-generated: updated LoadConfigs to new no-arg signature
	mockAgent.EXPECT().LoadConfigs().Return(agentConfig).AnyTimes()
	mockAgent.EXPECT().SetID().Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetLogger(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetHistory(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetSkills(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetPrompt(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetKnowledgeBase(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().SetToolsConfig(gomock.Any()).Return(expectedErr).AnyTimes()
	mockAgent.EXPECT().BeforeStart(gomock.Any()).Return(nil).AnyTimes()
	mockAgent.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()

	diagnostics := core.StartAgentCore(mockAgent, appConfig)
	if len(diagnostics) == 0 {
		t.Error("expected diagnostics, got empty")
	}
}

// =============================================================================
// StopAgentCore function tests
// =============================================================================

func TestStopAgentCore_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := testmock.NewMockAgentCore(ctrl)
	agentConfig := core.AgentCoreConfig{}

	// auto-generated: updated LoadConfigs to new no-arg signature
	gomock.InOrder(
		mockAgent.EXPECT().LoadConfigs().Return(agentConfig),
		mockAgent.EXPECT().BeforeStop(agentConfig).Return(nil),
		mockAgent.EXPECT().Stop(agentConfig).Return(nil),
	)

	diagnostics := core.StopAgentCore(mockAgent)
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}
