// test cases for core.Provider interface and related functions
package test

import (
	"context"

	// auto-add: the tool call batch tests assert the concurrency and the ordering of a batch
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockProvider satisfies core.Provider
var _ core.Provider = (*testmock.MockProvider)(nil)

// =============================================================================
// Provider interface tests
// =============================================================================

func TestMockProvider_GetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	mockProvider.EXPECT().GetID().Return("test-provider")

	id := mockProvider.GetID()
	if id != "test-provider" {
		t.Errorf("expected id 'test-provider', got %s", id)
	}
}

func TestMockProvider_GetName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	mockProvider.EXPECT().GetName().Return("qwen3.8-max")

	name := mockProvider.GetName()
	if name != "qwen3.8-max" {
		t.Errorf("expected name 'qwen3.8-max', got %s", name)
	}
}

func TestMockProvider_Auth_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	config := core.ModelConfig{
		Name:    "qwen",
		BaseUrl: "https://api.example.com",
		APIKey:  core.APIKey{Key: "test-key"},
	}

	mockProvider.EXPECT().Auth(config).Return(nil)

	err := mockProvider.Auth(config)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockProvider_Auth_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	config := core.ModelConfig{
		APIKey: core.APIKey{Key: "invalid-key"},
	}
	expectedErr := &core.Diagnostic{
		Level: core.SeverityWarn,
		Code:  core.MessageCodeProviderAuthError,
	}

	mockProvider.EXPECT().Auth(config).Return(expectedErr)

	err := mockProvider.Auth(config)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMockProvider_Init(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	config := core.ModelConfig{
		APIKey: core.APIKey{Key: "test-key"},
	}

	mockProvider.EXPECT().Init(config).Return(nil)

	err := mockProvider.Init(config)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockProvider_GetModelConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	expectedConfig := core.ModelConfig{
		Name:          "qwen",
		MaxTokenUsage: 4096,
		BaseUrl:       "https://api.example.com",
	}

	mockProvider.EXPECT().GetModelConfig().Return(expectedConfig)

	config := mockProvider.GetModelConfig()
	if config.MaxTokenUsage != 4096 {
		t.Errorf("expected MaxTokenUsage 4096, got %d", config.MaxTokenUsage)
	}
}

func TestMockProvider_Complete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockProvider.EXPECT().Complete(messages, tools, "test-model").Return(nil, nil)

	answer, diagnostics := mockProvider.Complete(messages, tools, "test-model")
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

// =============================================================================
// Question interface tests
// =============================================================================

func TestMockQuestion_GetID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().GetID().Return("q-123")

	id := mockQuestion.GetID()
	if id != "q-123" {
		t.Errorf("expected id 'q-123', got %s", id)
	}
}

func TestMockQuestion_GetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().GetQuery().Return("What is Go?")

	query := mockQuestion.GetQuery()
	if query != "What is Go?" {
		t.Errorf("expected query 'What is Go?', got %s", query)
	}
}

func TestMockQuestion_SetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().SetQuery("new query")

	mockQuestion.SetQuery("new query")
}

func TestMockQuestion_GetProviderName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().GetProviderName().Return("qwen")

	name := mockQuestion.GetProviderName()
	if name != "qwen" {
		t.Errorf("expected provider name 'qwen', got %s", name)
	}
}

func TestMockQuestion_GetSessionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().GetSessionID().Return("session-123")

	sessionID := mockQuestion.GetSessionID()
	if sessionID != "session-123" {
		t.Errorf("expected session id 'session-123', got %s", sessionID)
	}
}

// =============================================================================
// Answer interface tests
// =============================================================================

func TestMockAnswer_ToString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnswer := testmock.NewMockAnswer(ctrl)
	mockAnswer.EXPECT().ToString().Return(`{"response":"hello"}`)

	str := mockAnswer.ToString()
	if str != `{"response":"hello"}` {
		t.Errorf("expected string representation, got %s", str)
	}
}

// =============================================================================
// ValidateProviders function tests
// =============================================================================

func TestValidateProviders_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	config := core.ModelConfig{APIKey: core.APIKey{Key: "test-key"}, Models: []string{"test-model"}}

	mockProvider.EXPECT().GetModelConfig().Return(config)
	mockProvider.EXPECT().Auth(config).Return(nil)
	mockProvider.EXPECT().Init(config).Return(nil)

	diagnostics := core.ValidateProviders([]core.Provider{mockProvider})
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %v", len(diagnostics), diagnostics)
	}
}

func TestValidateProviders_AuthError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := testmock.NewMockProvider(ctrl)
	config := core.ModelConfig{APIKey: core.APIKey{Key: "invalid"}, Models: []string{"test-model"}}
	authErr := &core.Diagnostic{
		Level: core.SeverityError,
		Code:  core.MessageCodeProviderAuthError,
	}

	mockProvider.EXPECT().GetModelConfig().Return(config)
	mockProvider.EXPECT().Auth(config).Return(authErr)

	diagnostics := core.ValidateProviders([]core.Provider{mockProvider})
	if len(diagnostics) == 0 {
		t.Error("expected diagnostics, got empty")
	}
}

func TestValidateProviders_NoAvailableProvider(t *testing.T) {
	diagnostics := core.ValidateProviders([]core.Provider{})
	if len(diagnostics) == 0 {
		t.Error("expected diagnostics for empty providers")
	}

	hasNoProviderError := false
	for _, d := range diagnostics {
		if d.Code == core.MessageCodeNoAvailableProvider {
			hasNoProviderError = true
			break
		}
	}
	if !hasNoProviderError {
		t.Error("expected MessageCodeNoAvailableProvider error")
	}
}

// =============================================================================
// AskQuestion function tests
// =============================================================================

func TestAskQuestion_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	messages := []core.ReActMessage{
		{Role: core.RoleSystem, Content: "system prompt"},
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}})
	mockQuestion.EXPECT().GetModelName().Return("test-model")
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools, "test-model").Return(nil, nil)

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestAskQuestion_UnknownProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	messages := []core.ReActMessage{{Role: core.RoleUser, Content: "hello"}}
	tools := []core.Tool{}

	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()

	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockQuestion.EXPECT().GetModelName().Return("test-model")
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}})
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools, "test-model").Return(nil, nil)

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestAskQuestion_NoProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{})
	mockQuestion.EXPECT().GetModelName().Return("test-model")

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

func TestAskQuestion_WithDiagnostics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}
	expectedDiag := []core.Diagnostic{
		{Level: core.SeverityWarn, Code: core.MessageCodeErrorFromLLM, Message: "LLM error"},
	}

	mockQuestion.EXPECT().GetModelName().Return("test-model")
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}})
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools, "test-model").Return(nil, expectedDiag)

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(diagnostics))
	}
}

// =============================================================================
// ProcessQuestion function tests
// =============================================================================

func TestProcessQuestion_InvalidResponse_Retry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockHarness := testmock.NewMockHarness(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	config := core.SessionConfig{
		ReActMaxRounds:     2,
		MemoryFileSplitter: "---",
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	initialConversation := core.Conversation{{Role: core.RoleUser, Content: "test"}}
	mockSession.EXPECT().GetConversation().Return(&initialConversation).AnyTimes()
	mockSession.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	mockSession.EXPECT().CancelQuery().AnyTimes()
	mockSession.EXPECT().GetLoadTools().Return(&[]core.Tool{}).AnyTimes()
	mockSession.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	mockSession.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	mockSession.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	mockSession.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()
	mockContext.EXPECT().GetMCPServerConfigs().Return(nil).AnyTimes()
	mockContext.EXPECT().GetToolsConfig().Return(nil).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetModelName().Return("test-model").AnyTimes()
	mockQuestion.EXPECT().GetRetryQuery().Return("retry query").AnyTimes()
	mockQuestion.EXPECT().SetQuery(gomock.Any()).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockQuestion.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	mockQuestion.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	mockHarness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	mockHarness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()

	mockHarness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	// auto-add: the reAct loop runs a tool call through the harness now, delegate back to CallTool
	// so the mocked tool runner is still the one being exercised
	mockHarness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion, mockHarness)
	if answer == nil {
		t.Error("expected default answer after max rounds, got nil")
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestProcessQuestion_StopReason(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockHarness := testmock.NewMockHarness(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	config := core.SessionConfig{
		ReActMaxRounds:     3,
		MemoryFileSplitter: "---",
	}

	expectedAnswer := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message: &core.ResponseMessage{
					Role:    core.RoleAssistant,
					Content: "final answer",
				},
			},
		},
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	initialConversation := core.Conversation{{Role: core.RoleUser, Content: "test"}}
	mockSession.EXPECT().GetConversation().Return(&initialConversation).AnyTimes()
	mockSession.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	mockSession.EXPECT().CancelQuery().AnyTimes()
	mockSession.EXPECT().GetLoadTools().Return(&[]core.Tool{}).AnyTimes()
	mockSession.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	mockSession.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	mockSession.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	mockSession.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()
	mockContext.EXPECT().GetMCPServerConfigs().Return(nil).AnyTimes()
	mockContext.EXPECT().GetToolsConfig().Return(nil).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetModelName().Return("test-model").AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockQuestion.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	mockQuestion.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	mockHarness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	mockHarness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	mockHarness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	mockHarness.EXPECT().GetConfirmDestructiveToolAnswer(gomock.Any()).Return("").AnyTimes()
	// auto-add: the reAct loop runs a tool call through the harness now, delegate back to CallTool
	// so the mocked tool runner is still the one being exercised
	mockHarness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedAnswer, nil)

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion, mockHarness)
	if answer == nil {
		t.Error("expected answer, got nil")
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestProcessQuestion_ToolCall_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockTool := testmock.NewMockTool(ctrl)
	mockHarness := testmock.NewMockHarness(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	config := core.SessionConfig{
		ReActMaxRounds:     3,
		MemoryFileSplitter: "---",
	}

	toolCalls := []core.ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: core.Function{
				Name:      "test_tool",
				Arguments: `{"key":"value"}`,
			},
		},
	}

	firstResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonFunction,
				Message: &core.ResponseMessage{
					Role:      core.RoleAssistant,
					Content:   "calling tool",
					ToolCalls: &toolCalls,
				},
			},
		},
	}

	finalResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message: &core.ResponseMessage{
					Role:    core.RoleAssistant,
					Content: "done",
				},
			},
		},
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	initialConversation := core.Conversation{{Role: core.RoleUser, Content: "test"}}
	mockSession.EXPECT().GetConversation().Return(&initialConversation).AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockSession.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	mockSession.EXPECT().CancelQuery().AnyTimes()
	mockSession.EXPECT().GetLoadTools().Return(&[]core.Tool{mockTool}).AnyTimes()
	mockSession.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	mockSession.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	mockSession.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	mockSession.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetModelName().Return("test-model").AnyTimes()
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockQuestion.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	mockQuestion.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	mockHarness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	mockHarness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	mockHarness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	mockHarness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockTool.EXPECT().GetName().Return("test_tool").AnyTimes()
	mockTool.EXPECT().Validate(gomock.Any()).Return(nil).AnyTimes()
	mockTool.EXPECT().GetRunner().Return(func(args map[string]any) (string, *core.Diagnostic) { return "Success", nil }).AnyTimes()
	mockTool.EXPECT().IsDestructive().Return(false).AnyTimes()
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion, mockHarness)
	if answer == nil {
		t.Error("expected answer, got nil")
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestProcessQuestion_ToolCall_ToolNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockHarness := testmock.NewMockHarness(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	config := core.SessionConfig{
		ReActMaxRounds:     3,
		MemoryFileSplitter: "---",
	}

	toolCalls := []core.ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: core.Function{
				Name:      "nonexistent_tool",
				Arguments: `{}`,
			},
		},
	}

	firstResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonFunction,
				Message: &core.ResponseMessage{
					Role:      core.RoleAssistant,
					Content:   "calling tool",
					ToolCalls: &toolCalls,
				},
			},
		},
	}

	finalResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message: &core.ResponseMessage{
					Role:    core.RoleAssistant,
					Content: "done",
				},
			},
		},
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	initialConversation := core.Conversation{{Role: core.RoleUser, Content: "test"}}
	mockSession.EXPECT().GetConversation().Return(&initialConversation).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockSession.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	mockSession.EXPECT().CancelQuery().AnyTimes()
	mockSession.EXPECT().GetLoadTools().Return(&[]core.Tool{}).AnyTimes()
	mockSession.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	mockSession.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	mockSession.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	mockSession.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()
	mockContext.EXPECT().GetMCPServerConfigs().Return(nil).AnyTimes()
	mockContext.EXPECT().GetToolsConfig().Return(nil).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetModelName().Return("test-model").AnyTimes()
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockQuestion.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	mockQuestion.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	mockHarness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	mockHarness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	mockHarness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	// auto-add: the reAct loop runs a tool call through the harness now, delegate back to CallTool
	// so the mocked tool runner is still the one being exercised
	mockHarness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, _ := core.ProcessQuestion(mockSession, mockQuestion, mockHarness)
	if answer == nil {
		t.Error("expected answer, got nil")
	}
}

func TestProcessQuestion_ToolCall_InvalidArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)
	mockTool := testmock.NewMockTool(ctrl)
	mockHarness := testmock.NewMockHarness(ctrl)
	mockContext := testmock.NewMockContext(ctrl)

	config := core.SessionConfig{
		ReActMaxRounds:     3,
		MemoryFileSplitter: "---",
	}

	toolCalls := []core.ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: core.Function{
				Name:      "test_tool",
				Arguments: `invalid json`,
			},
		},
	}

	firstResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonFunction,
				Message: &core.ResponseMessage{
					Role:      core.RoleAssistant,
					Content:   "calling tool",
					ToolCalls: &toolCalls,
				},
			},
		},
	}

	finalResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message: &core.ResponseMessage{
					Role:    core.RoleAssistant,
					Content: "done",
				},
			},
		},
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GetID().Return("test-session").AnyTimes()
	mockSession.EXPECT().GetContext().Return(mockContext).AnyTimes()
	mockContext.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	initialConversation := core.Conversation{{Role: core.RoleUser, Content: "test"}}
	mockSession.EXPECT().GetConversation().Return(&initialConversation).AnyTimes()
	mockSession.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	mockSession.EXPECT().CancelQuery().AnyTimes()
	mockSession.EXPECT().GetLoadTools().Return(&[]core.Tool{mockTool}).AnyTimes()
	mockSession.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	mockSession.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	mockSession.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	mockSession.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	mockSession.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()
	mockContext.EXPECT().GetMCPServerConfigs().Return(nil).AnyTimes()
	mockContext.EXPECT().GetToolsConfig().Return(nil).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockProvider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetModelName().Return("test-model").AnyTimes()
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockQuestion.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	mockQuestion.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	mockHarness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	mockHarness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockHarness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	mockHarness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	mockHarness.EXPECT().GetConfirmDestructiveToolAnswer(gomock.Any()).Return("").AnyTimes()
	// auto-add: the reAct loop runs a tool call through the harness now, delegate back to CallTool
	// so the mocked tool runner is still the one being exercised
	mockHarness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockTool.EXPECT().GetName().Return("test_tool")
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, _ := core.ProcessQuestion(mockSession, mockQuestion, mockHarness)
	if answer == nil {
		t.Error("expected answer, got nil")
	}
}

// =============================================================================
// Tool call batch tests
// =============================================================================

// toolCallBatchFixture carries the mocks shared by the tool call batch tests.
// auto-add: the batch tests only differ in the tools they load, the responses the provider
// returns and the few harness calls they assert on.
type toolCallBatchFixture struct {
	session  *testmock.MockSession
	question *testmock.MockQuestion
	provider *testmock.MockProvider
	harness  *testmock.MockHarness
	// mockContext is the session runtime context
	mockContext *testmock.MockContext
	// conversation is the session conversation, ProcessQuestion appends into it in place
	conversation core.Conversation
}

// newToolCallBatchFixture wires the expectations every reAct round needs, each one AnyTimes so a
// test only adds what it asserts on. The responses are returned by Complete in order.
func newToolCallBatchFixture(ctrl *gomock.Controller, loadedTools []core.Tool, responses ...core.AgentResponse) *toolCallBatchFixture {
	fixture := &toolCallBatchFixture{
		session:      testmock.NewMockSession(ctrl),
		question:     testmock.NewMockQuestion(ctrl),
		provider:     testmock.NewMockProvider(ctrl),
		harness:      testmock.NewMockHarness(ctrl),
		mockContext:  testmock.NewMockContext(ctrl),
		conversation: core.Conversation{{Role: core.RoleUser, Content: "test"}},
	}

	fixture.session.EXPECT().GetConfigs().Return(core.SessionConfig{
		ReActMaxRounds:     3,
		MemoryFileSplitter: "---",
	}).AnyTimes()
	fixture.session.EXPECT().GetContext().Return(fixture.mockContext).AnyTimes()
	fixture.session.EXPECT().GetConversation().Return(&fixture.conversation).AnyTimes()
	fixture.session.EXPECT().GetQueryCtx().Return(context.Background()).AnyTimes()
	fixture.session.EXPECT().CancelQuery().AnyTimes()
	fixture.session.EXPECT().GetLoadTools().Return(&loadedTools).AnyTimes()
	fixture.session.EXPECT().SetLoadTools(gomock.Any()).AnyTimes()
	fixture.session.EXPECT().IsToolConfirmed(gomock.Any()).Return(false).AnyTimes()
	fixture.session.EXPECT().SetToolConfirmed(gomock.Any(), gomock.Any()).AnyTimes()
	fixture.session.EXPECT().ClearToolConfirmed(gomock.Any()).AnyTimes()
	fixture.session.EXPECT().GetPendingMCPToolCall(gomock.Any()).Return(nil).AnyTimes()
	fixture.session.EXPECT().SetPendingMCPToolCall(gomock.Any(), gomock.Any()).AnyTimes()
	fixture.session.EXPECT().DeletePendingMCPToolCall(gomock.Any()).AnyTimes()
	fixture.session.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	fixture.session.EXPECT().GetID().Return("test-session-id").AnyTimes()

	fixture.mockContext.EXPECT().GetModelProviders().Return([]core.Provider{fixture.provider}).AnyTimes()
	fixture.mockContext.EXPECT().GetPrompt().Return([]byte("system prompt")).AnyTimes()
	fixture.mockContext.EXPECT().GetSkills().Return([]core.SkillDefinition{}).AnyTimes()
	fixture.mockContext.EXPECT().GetHistory().Return([]byte("")).AnyTimes()

	fixture.provider.EXPECT().GetName().Return("qwen").AnyTimes()
	fixture.provider.EXPECT().GetModelConfig().Return(core.ModelConfig{Models: []string{"test-model"}}).AnyTimes()
	for _, response := range responses {
		fixture.provider.EXPECT().Complete(gomock.Any(), gomock.Any(), gomock.Any()).Return(response, nil)
	}

	fixture.question.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	fixture.question.EXPECT().GetQuery().Return("test query").AnyTimes()
	fixture.question.EXPECT().GetModelName().Return("test-model").AnyTimes()
	fixture.question.EXPECT().GetRetryQuery().Return("retry query").AnyTimes()
	fixture.question.EXPECT().SetQuery(gomock.Any()).AnyTimes()
	fixture.question.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	fixture.question.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	fixture.question.EXPECT().GetType().Return(core.QuestionType("normal")).AnyTimes()
	fixture.question.EXPECT().GetResponseChan().Return(make(chan core.Answer, 10)).AnyTimes()

	fixture.harness.EXPECT().GenerateFinalPrompt(gomock.Any(), gomock.Any()).Return("final prompt").AnyTimes()
	fixture.harness.EXPECT().LoadTools(gomock.Any(), gomock.Any()).AnyTimes()
	fixture.harness.EXPECT().SetCurrRoundMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	fixture.harness.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	fixture.harness.EXPECT().HandleUserQuestion(gomock.Any(), gomock.Any()).
		Return(&core.ReActMessage{Role: core.RoleUser, Content: "test query"}, core.Diagnostic{}).AnyTimes()
	// the reAct loop runs a tool call through the harness, delegate back to CallTool so the mocked
	// tool runner is still the one being exercised
	fixture.harness.EXPECT().RunToolCall(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, toolCall core.ToolCall, targetTool core.Tool, args map[string]any, model string) (string, *core.Diagnostic) {
			return core.CallTool(targetTool, args)
		}).AnyTimes()

	return fixture
}

// toolCallBatchResponse builds the assistant message that asks for the given tool calls
func toolCallBatchResponse(toolCalls *[]core.ToolCall) core.AgentResponse {
	return core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonFunction,
				Message: &core.ResponseMessage{
					Role:      core.RoleAssistant,
					Content:   "calling tools",
					ToolCalls: toolCalls,
				},
			},
		},
	}
}

// toolMessagesOf returns the tool responses appended to the conversation, in conversation order
func toolMessagesOf(conversation core.Conversation) []core.ReActMessage {
	toolMessages := []core.ReActMessage{}
	for _, message := range conversation {
		if message.Role == core.RoleTool {
			toolMessages = append(toolMessages, message)
		}
	}
	return toolMessages
}

// TestProcessQuestion_ToolCallBatch_RunsConcurrently covers the batch execution: the tool calls of
// one assistant message run concurrently, and their responses are still appended in tool call
// order so each of them stays matched with its tool_call ID.
func TestProcessQuestion_ToolCallBatch_RunsConcurrently(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const toolCallCount = 3
	const toolSleep = 200 * time.Millisecond

	toolCalls := make([]core.ToolCall, 0, toolCallCount)
	for i := 0; i < toolCallCount; i++ {
		toolCalls = append(toolCalls, core.ToolCall{
			ID:   fmt.Sprintf("call-%d", i),
			Type: "function",
			Function: core.Function{
				Name:      "test_tool",
				Arguments: fmt.Sprintf(`{"index":%d}`, i),
			},
		})
	}

	// running counts the runners currently inside a tool, maxRunning is the highest overlap seen
	var running, maxRunning int32
	mockTool := testmock.NewMockTool(ctrl)
	mockTool.EXPECT().GetName().Return("test_tool").AnyTimes()
	mockTool.EXPECT().IsDestructive().Return(false).AnyTimes()
	mockTool.EXPECT().Validate(gomock.Any()).Return(nil).AnyTimes()
	mockTool.EXPECT().GetRunner().Return(func(args map[string]any) (string, *core.Diagnostic) {
		current := atomic.AddInt32(&running, 1)
		for {
			previous := atomic.LoadInt32(&maxRunning)
			if current <= previous || atomic.CompareAndSwapInt32(&maxRunning, previous, current) {
				break
			}
		}
		time.Sleep(toolSleep)
		atomic.AddInt32(&running, -1)
		return fmt.Sprintf("result-%d", int(args["index"].(float64))), nil
	}).AnyTimes()

	finalResponse := core.AgentResponse{
		Choices: []core.Choices{
			{
				FinishReason: core.FinishReasonStop,
				Message:      &core.ResponseMessage{Role: core.RoleAssistant, Content: "done"},
			},
		},
	}

	fixture := newToolCallBatchFixture(ctrl, []core.Tool{mockTool}, toolCallBatchResponse(&toolCalls), finalResponse)

	started := time.Now()
	answer, diagnostics := core.ProcessQuestion(fixture.session, fixture.question, fixture.harness)
	elapsed := time.Since(started)

	if answer == nil {
		t.Fatal("expected answer, got nil")
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
	// every call of the batch has to be inside its tool at the same time
	if got := atomic.LoadInt32(&maxRunning); got != toolCallCount {
		t.Errorf("expected %d tool calls to overlap, only %d ran at the same time", toolCallCount, got)
	}
	// a serial batch would take toolCallCount * toolSleep
	if elapsed >= time.Duration(toolCallCount)*toolSleep {
		t.Errorf("expected the batch to run concurrently, it took %v", elapsed)
	}

	// the responses keep the tool call order, not the completion order
	toolMessages := toolMessagesOf(fixture.conversation)
	if len(toolMessages) != toolCallCount {
		t.Fatalf("expected %d tool messages, got %d", toolCallCount, len(toolMessages))
	}
	for i, message := range toolMessages {
		if expected := fmt.Sprintf("call-%d", i); message.ToolCallID != expected {
			t.Errorf("tool message %d: expected tool call ID %s, got %s", i, expected, message.ToolCallID)
		}
		if expected := fmt.Sprintf("result-%d", i); message.Content != expected {
			t.Errorf("tool message %d: expected content %s, got %s", i, expected, message.Content)
		}
	}
}

// TestProcessQuestion_DestructiveBatch_ConfirmsWholeBatch covers the batch confirmation: every
// destructive call of one assistant message is collected into a single confirmation response, the
// other calls of the batch still run, and each tool_call keeps its own response.
func TestProcessQuestion_DestructiveBatch_ConfirmsWholeBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	toolCalls := []core.ToolCall{
		{ID: "call-destructive-1", Type: "function", Function: core.Function{Name: "destructive_tool", Arguments: `{}`}},
		{ID: "call-normal", Type: "function", Function: core.Function{Name: "test_tool", Arguments: `{}`}},
		{ID: "call-destructive-2", Type: "function", Function: core.Function{Name: "destructive_tool", Arguments: `{}`}},
	}

	mockDestructiveTool := testmock.NewMockTool(ctrl)
	mockDestructiveTool.EXPECT().GetName().Return("destructive_tool").AnyTimes()
	mockDestructiveTool.EXPECT().IsDestructive().Return(true).AnyTimes()

	var normalRuns int32
	mockTool := testmock.NewMockTool(ctrl)
	mockTool.EXPECT().GetName().Return("test_tool").AnyTimes()
	mockTool.EXPECT().IsDestructive().Return(false).AnyTimes()
	mockTool.EXPECT().Validate(gomock.Any()).Return(nil).AnyTimes()
	mockTool.EXPECT().GetRunner().Return(func(args map[string]any) (string, *core.Diagnostic) {
		atomic.AddInt32(&normalRuns, 1)
		return "normal result", nil
	}).AnyTimes()

	fixture := newToolCallBatchFixture(ctrl, []core.Tool{mockDestructiveTool, mockTool}, toolCallBatchResponse(&toolCalls))

	var confirmCalls []core.ToolConfirmCall
	fixture.harness.EXPECT().GetConfirmDestructiveToolAnswer(gomock.Any()).Return("waiting for confirmation").AnyTimes()
	fixture.harness.EXPECT().GenerateToolConfirmResponse(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(session core.Session, calls []core.ToolConfirmCall, usage core.Usage) core.Answer {
			confirmCalls = calls
			return core.AgentResponse{Response: "confirm the batch"}
		}).Times(1)

	answer, _ := core.ProcessQuestion(fixture.session, fixture.question, fixture.harness)

	confirmAnswer, ok := answer.(core.AgentResponse)
	if !ok || confirmAnswer.Response != "confirm the batch" {
		t.Fatalf("expected the confirmation response to end the round, got %#v", answer)
	}

	// both destructive calls are asked about at once, in tool call order
	if len(confirmCalls) != 2 {
		t.Fatalf("expected 2 calls waiting for confirmation, got %d", len(confirmCalls))
	}
	if confirmCalls[0].ToolCallID != "call-destructive-1" || confirmCalls[1].ToolCallID != "call-destructive-2" {
		t.Errorf("expected the destructive calls in tool call order, got %s and %s",
			confirmCalls[0].ToolCallID, confirmCalls[1].ToolCallID)
	}

	// the non destructive call of the batch is not held back by the confirmation
	if got := atomic.LoadInt32(&normalRuns); got != 1 {
		t.Errorf("expected the normal tool call to run once, got %d", got)
	}

	// every tool_call of the assistant message keeps its response, in tool call order
	toolMessages := toolMessagesOf(fixture.conversation)
	expectedContents := []string{"waiting for confirmation", "normal result", "waiting for confirmation"}
	if len(toolMessages) != len(expectedContents) {
		t.Fatalf("expected %d tool messages, got %d", len(expectedContents), len(toolMessages))
	}
	for i, message := range toolMessages {
		if message.ToolCallID != toolCalls[i].ID {
			t.Errorf("tool message %d: expected tool call ID %s, got %s", i, toolCalls[i].ID, message.ToolCallID)
		}
		if message.Content != expectedContents[i] {
			t.Errorf("tool message %d: expected content %q, got %q", i, expectedContents[i], message.Content)
		}
	}
}
