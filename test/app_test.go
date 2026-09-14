// test cases for core.AgentApp and related interfaces
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockAgentApp satisfies core.AgentApp
var _ core.AgentApp = (*testmock.MockAgentApp)(nil)

// =============================================================================
// ServiceProvider interface tests
// =============================================================================

func TestMockServiceProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// ServiceProvider is an empty interface, just verify mock creation
	_ = testmock.NewMockServiceProvider(ctrl)
}

// =============================================================================
// SingalManager interface tests
// =============================================================================

func TestMockSingalManager_GracefulQuit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSingal := testmock.NewMockSingalManager(ctrl)

	mockSingal.EXPECT().GracefulQuit()

	mockSingal.GracefulQuit()
}

// =============================================================================
// AgentServer interface tests
// =============================================================================

func TestMockAgentServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServer := testmock.NewMockAgentServer(ctrl)

	// AgentServer embeds SingalManager, ServiceProvider
	// Test a method from each embedded interface
	mockServer.EXPECT().GracefulQuit()

	mockServer.GracefulQuit()
}

// =============================================================================
// UserManager interface tests
// =============================================================================

func TestMockUserManager_AuthUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserMgr := testmock.NewMockUserManager(ctrl)
	mockUserMgr.EXPECT().AuthUser("valid-token").Return(nil)

	err := mockUserMgr.AuthUser("valid-token")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockUserManager_AuthUser_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserMgr := testmock.NewMockUserManager(ctrl)
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeProviderAuthError,
		Message: "invalid token",
	}

	mockUserMgr.EXPECT().AuthUser("invalid-token").Return(expectedErr)

	err := mockUserMgr.AuthUser("invalid-token")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMockUserManager_CheckUserPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserMgr := testmock.NewMockUserManager(ctrl)
	agentConfig := core.AgentConfig{Version: "1.0.0"}

	mockUserMgr.EXPECT().CheckUserPlan(agentConfig).Return(nil)

	err := mockUserMgr.CheckUserPlan(agentConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockUserManager_SetModelProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserMgr := testmock.NewMockUserManager(ctrl)
	modelConfigs := []core.ModelConfig{
		{Name: "qwen", BaseUrl: "https://api.qwen.com"},
	}
	appConfig := core.AppConfig{Language: core.Language_EN}

	mockUserMgr.EXPECT().SetModelProviders(modelConfigs, appConfig).Return(nil)

	err := mockUserMgr.SetModelProviders(modelConfigs, appConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// =============================================================================
// AgentApp interface tests
// =============================================================================

func TestMockAgentApp_Render(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().Render("Welcome to Agent App")

	mockApp.Render("Welcome to Agent App")
}

func TestMockAgentApp_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().Translate("hello", gomock.Eq(core.LanguageType("Chinese"))).Return("你好")

	result := mockApp.Translate("hello", core.LanguageType("Chinese"))
	if result != "你好" {
		t.Errorf("expected '你好', got %s", result)
	}
}
