// test cases for core.Tool interface and related functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockTool satisfies core.Tool
var _ core.Tool = (*testmock.MockTool)(nil)

// =============================================================================
// Tool interface tests
// =============================================================================

func TestMockTool_GetSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	expectedSchema := core.ToolSchema{
		Type: "function",
		Function: core.FunctionSchema{
			Name:        "TestTool",
			Description: "A test tool",
			Parameters:  map[string]any{"type": "object"},
		},
	}

	mockTool.EXPECT().GetSchema().Return(expectedSchema)

	schema := mockTool.GetSchema()
	if schema.Function.Name != "TestTool" {
		t.Errorf("expected name TestTool, got %s", schema.Function.Name)
	}
}

func TestMockTool_GetName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	mockTool.EXPECT().GetName().Return("TestTool")

	name := mockTool.GetName()
	if name != "TestTool" {
		t.Errorf("expected name TestTool, got %s", name)
	}
}

func TestMockTool_GetDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	mockTool.EXPECT().GetDescription().Return("A test tool description")

	desc := mockTool.GetDescription()
	if desc != "A test tool description" {
		t.Errorf("expected description 'A test tool description', got %s", desc)
	}
}

func TestMockTool_Validate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	args := map[string]any{"key": "value"}

	mockTool.EXPECT().Validate(args).Return(nil)

	err := mockTool.Validate(args)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockTool_Validate_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	args := map[string]any{"key": "value"}
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeToolValidateError,
		Message: "validation failed",
	}

	mockTool.EXPECT().Validate(args).Return(expectedErr)

	err := mockTool.Validate(args)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %s", err.Message)
	}
}

func TestMockTool_GetRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	runner := func(args map[string]any) *core.Diagnostic { return nil }

	mockTool.EXPECT().GetRunner().Return(runner)

	gotRunner := mockTool.GetRunner()
	if gotRunner == nil {
		t.Error("expected runner function, got nil")
	}
}

// =============================================================================
// CallTool function tests
// =============================================================================

func TestCallTool_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	args := map[string]any{"path": "/tmp/test.txt", "content": "hello"}

	gomock.InOrder(
		mockTool.EXPECT().Validate(args).Return(nil),
		mockTool.EXPECT().GetRunner().Return(func(args map[string]any) *core.Diagnostic {
			return nil
		}),
	)

	err := core.CallTool(mockTool, args)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestCallTool_ValidateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	args := map[string]any{"path": "./invalid"}
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeToolValidateError,
		Message: "path traversal detected",
	}

	mockTool.EXPECT().Validate(args).Return(expectedErr)

	err := core.CallTool(mockTool, args)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Code != core.MessageCodeToolValidateError {
		t.Errorf("expected code %d, got %d", core.MessageCodeToolValidateError, err.Code)
	}
}

func TestCallTool_RunError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTool := testmock.NewMockTool(ctrl)
	args := map[string]any{"path": "/tmp/test.txt"}
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeToolRunError,
		Message: "failed to write file",
	}

	gomock.InOrder(
		mockTool.EXPECT().Validate(args).Return(nil),
		mockTool.EXPECT().GetRunner().Return(func(args map[string]any) *core.Diagnostic {
			return expectedErr
		}),
	)

	err := core.CallTool(mockTool, args)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Code != core.MessageCodeToolRunError {
		t.Errorf("expected code %d, got %d", core.MessageCodeToolRunError, err.Code)
	}
}

// =============================================================================
// MCPAvailable interface tests
// =============================================================================

func TestMockMCPAvailable_Transform(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMCP := testmock.NewMockMCPAvailable[string](ctrl)
	mockMCP.EXPECT().Transform("test data").Return("transformed")

	result := mockMCP.Transform("test data")
	if result != "transformed" {
		t.Errorf("expected 'transformed', got %s", result)
	}
}
