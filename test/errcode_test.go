// auto-generated: test cases for core.Serializable interface and Diagnostic
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockSerializable satisfies core.Serializable
var _ core.Serializable = (*testmock.MockSerializable)(nil)

// =============================================================================
// Serializable interface tests
// =============================================================================

func TestMockSerializable_ToString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSerializable := testmock.NewMockSerializable(ctrl)
	mockSerializable.EXPECT().ToString().Return(`{"data":"test"}`)

	result := mockSerializable.ToString()
	if result != `{"data":"test"}` {
		t.Errorf("expected '{\"data\":\"test\"}', got %s", result)
	}
}

// =============================================================================
// Diagnostic tests
// =============================================================================

func TestDiagnostic_ToString(t *testing.T) {
	diag := core.Diagnostic{
		Code:    core.MessageCodeSuccess,
		Level:   core.SeverityInfo,
		Message: "operation successful",
		Data:    "extra data",
	}

	result := diag.ToString()
	if result == "" {
		t.Error("expected non-empty string")
	}
	// Should contain the message
	if !contains(result, "operation successful") {
		t.Errorf("expected result to contain 'operation successful', got %s", result)
	}
}

func TestDiagnostic_ToString_InvalidJSON(t *testing.T) {
	// Diagnostic with valid fields should always marshal correctly
	diag := core.Diagnostic{
		Code:    core.MessageCodeSuccess,
		Level:   core.SeverityInfo,
		Message: "test",
	}

	result := diag.ToString()
	if result == "" {
		t.Error("expected non-empty string")
	}
}

// =============================================================================
// DiagnosticList tests
// =============================================================================

func TestDiagnosticList_ToString(t *testing.T) {
	diagList := core.DiagnosticList{
		{Code: core.MessageCodeSuccess, Level: core.SeverityInfo, Message: "first"},
		{Code: core.MessageCodeConfigFileFormatError, Level: core.SeverityError, Message: "second"},
	}

	result := diagList.ToString()
	if result == "" {
		t.Error("expected non-empty string")
	}
}

func TestDiagnosticList_ToString_Empty(t *testing.T) {
	diagList := core.DiagnosticList{}

	result := diagList.ToString()
	// Empty list should still marshal to "[]"
	if result != "[]" {
		t.Errorf("expected '[]', got %s", result)
	}
}

// =============================================================================
// Severity constants tests
// =============================================================================

func TestSeverity_Constants(t *testing.T) {
	if core.SeverityInfo != 0 {
		t.Errorf("expected SeverityInfo to be 0, got %d", core.SeverityInfo)
	}
	if core.SeverityWarn != 1 {
		t.Errorf("expected SeverityWarn to be 1, got %d", core.SeverityWarn)
	}
	if core.SeverityError != 2 {
		t.Errorf("expected SeverityError to be 2, got %d", core.SeverityError)
	}
}

// =============================================================================
// MessageCode constants tests
// =============================================================================

func TestMessageCode_Constants(t *testing.T) {
	if core.MessageCodeSuccess != 0 {
		t.Errorf("expected MessageCodeSuccess to be 0, got %d", core.MessageCodeSuccess)
	}
	if core.MessageCodeConfigFileNotFound != 100 {
		t.Errorf("expected MessageCodeConfigFileNotFound to be 100, got %d", core.MessageCodeConfigFileNotFound)
	}
	if core.MessageCodeProviderCreateError != 300 {
		t.Errorf("expected MessageCodeProviderCreateError to be 300, got %d", core.MessageCodeProviderCreateError)
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
