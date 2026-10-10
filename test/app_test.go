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
// UserManager interface tests (via MockAgentApp)
// =============================================================================

func TestMockAgentApp_AuthUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().AuthUser("valid-token").Return(nil)

	err := mockApp.AuthUser("valid-token")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockAgentApp_AuthUser_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeProviderAuthError,
		Message: "invalid token",
	}

	mockApp.EXPECT().AuthUser("invalid-token").Return(expectedErr)

	err := mockApp.AuthUser("invalid-token")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMockAgentApp_CheckUserPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	userInfo := "user-123"
	plan := core.Plan{Name: core.PlanClassic}

	mockApp.EXPECT().CheckUserPlan(userInfo, plan).Return(nil)

	err := mockApp.CheckUserPlan(userInfo, plan)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockAgentApp_GetUserPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	expectedPlan := core.Plan{Name: core.PlanEnterprise}

	mockApp.EXPECT().GetUserPlan("user-123").Return(expectedPlan, nil)

	plan, err := mockApp.GetUserPlan("user-123")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if plan.Name != core.PlanEnterprise {
		t.Errorf("expected plan name 'PlanEnterprise', got %v", plan.Name)
	}
}

// =============================================================================
// AgentApp interface tests
// =============================================================================

func TestMockAgentApp_Render(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().Render(core.MessageCodeSuccess, gomock.Eq(core.LanguageType(core.Language_EN))).Return("Success")

	result := mockApp.Render(core.MessageCodeSuccess, core.Language_EN)
	if result != "Success" {
		t.Errorf("expected 'Success', got %s", result)
	}
}

func TestMockAgentApp_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().Translate(core.MessageCodeSuccess, gomock.Eq(core.LanguageType(core.Language_ZH))).Return("成功")

	result := mockApp.Translate(core.MessageCodeSuccess, core.Language_ZH)
	if result != "成功" {
		t.Errorf("expected '成功', got %s", result)
	}
}

func TestMockAgentApp_Welcome(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockApp := testmock.NewMockAgentApp(ctrl)
	mockApp.EXPECT().Welcome().Return("Welcome!")

	result := mockApp.Welcome()
	if result != "Welcome!" {
		t.Errorf("expected 'Welcome!', got %s", result)
	}
}
