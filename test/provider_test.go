// test cases for core.Provider interface and related functions
package test

import (
	"testing"

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

	mockProvider.EXPECT().Complete(messages, tools).Return(nil, nil)

	answer, diagnostics := mockProvider.Complete(messages, tools)
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
	config := core.ModelConfig{APIKey: core.APIKey{Key: "test-key"}}

	mockProvider.EXPECT().Auth(gomock.Any()).Return(nil)
	mockProvider.EXPECT().GetModelConfig().Return(config).Times(2)
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
	config := core.ModelConfig{APIKey: core.APIKey{Key: "invalid"}}
	authErr := &core.Diagnostic{
		Level: core.SeverityError,
		Code:  core.MessageCodeProviderAuthError,
	}

	mockProvider.EXPECT().Auth(config).Return(authErr)
	mockProvider.EXPECT().GetModelConfig().Return(config)

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

	messages := []core.ReActMessage{
		{Role: core.RoleSystem, Content: "system prompt"},
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockQuestion.EXPECT().GetProviderName().Return("qwen")
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools).Return(nil, nil)

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestAskQuestion_ProviderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)

	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockQuestion.EXPECT().GetProviderName().Return("unknown-provider")
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools).Return(nil, nil)

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

	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}

	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{})
	mockQuestion.EXPECT().GetProviderName().Return("qwen")

	answer, diagnostics := core.AskQuestion(mockSession, messages, tools, mockQuestion)
	if answer != nil {
		t.Errorf("expected nil answer, got %v", answer)
	}
	if len(diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diagnostics))
	}
}

func TestAskQuestion_WithDiagnostics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := testmock.NewMockSession(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	mockProvider := testmock.NewMockProvider(ctrl)

	messages := []core.ReActMessage{
		{Role: core.RoleUser, Content: "hello"},
	}
	tools := []core.Tool{}
	expectedDiag := []core.Diagnostic{
		{Level: core.SeverityWarn, Code: core.MessageCodeErrorFromLLM, Message: "LLM error"},
	}

	mockQuestion.EXPECT().GetProviderName().Return("qwen")
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	mockProvider.EXPECT().GetName().Return("qwen")
	mockProvider.EXPECT().Complete(messages, tools).Return(nil, expectedDiag)

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

	config := core.SessionConfig{
		ReActMaxRounds:     2,
		MemoryFileSplitter: "---",
	}

	mockSession.EXPECT().GetConfigs().Return(config).AnyTimes()
	mockSession.EXPECT().GenerateFinalContext(mockQuestion).Return("context")
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return([]core.Tool{}).AnyTimes()
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetRetryQuery().Return("retry query").AnyTimes()
	mockQuestion.EXPECT().SetQuery("retry query").AnyTimes()
	mockQuestion.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	emptyConversation := core.Conversation{}
	mockSession.EXPECT().GetConversation().Return(&emptyConversation)
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion)
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
	mockSession.EXPECT().GenerateFinalContext(mockQuestion).Return("context")
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return([]core.Tool{}).AnyTimes()
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider})
	emptyConversation := core.Conversation{}
	mockSession.EXPECT().GetConversation().Return(&emptyConversation)
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen")
	mockQuestion.EXPECT().GetProviderName().Return("qwen")
	mockQuestion.EXPECT().GetQuery().Return("test query")
	mockQuestion.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	//  mock GetEnableThinking for ProcessQuestion
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(expectedAnswer, nil)

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion)
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
	mockSession.EXPECT().GenerateFinalContext(mockQuestion).Return("context")
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return([]core.Tool{mockTool}).AnyTimes()
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	emptyConversation := core.Conversation{}
	mockSession.EXPECT().GetConversation().Return(&emptyConversation)
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	//  mock GetHintChan for ProcessQuestion hint sending
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockTool.EXPECT().GetName().Return("test_tool")
	mockTool.EXPECT().Validate(gomock.Any()).Return(nil)
	mockTool.EXPECT().GetRunner().Return(func(args map[string]any) (string, *core.Diagnostic) { return "Success", nil })
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, diagnostics := core.ProcessQuestion(mockSession, mockQuestion)
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
	mockSession.EXPECT().GenerateFinalContext(mockQuestion).Return("context")
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return([]core.Tool{}).AnyTimes()
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	emptyConversation := core.Conversation{}
	mockSession.EXPECT().GetConversation().Return(&emptyConversation)
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	//  mock GetHintChan for ProcessQuestion hint sending
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, _ := core.ProcessQuestion(mockSession, mockQuestion)
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
	mockSession.EXPECT().GenerateFinalContext(mockQuestion).Return("context")
	mockSession.EXPECT().SelectTools(mockQuestion, gomock.Any()).Return([]core.Tool{mockTool}).AnyTimes()
	mockSession.EXPECT().GetModelProviders().Return([]core.Provider{mockProvider}).AnyTimes()
	emptyConversation := core.Conversation{}
	mockSession.EXPECT().GetConversation().Return(&emptyConversation)
	mockSession.EXPECT().GetEventChans().Return(map[string]chan core.Event[any]{}).AnyTimes()
	mockProvider.EXPECT().GetName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetProviderName().Return("qwen").AnyTimes()
	mockQuestion.EXPECT().GetQuery().Return("test query").AnyTimes()
	mockQuestion.EXPECT().GetDefaultAnswer().Return(core.AgentResponse{Response: "default"}).AnyTimes()
	//  mock GetHintChan for ProcessQuestion hint sending
	mockQuestion.EXPECT().GetHintChan().Return(make(chan core.Answer, 10)).AnyTimes()
	mockQuestion.EXPECT().GetEnableThinking().Return(false).AnyTimes()

	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(firstResponse, nil)
	mockTool.EXPECT().GetName().Return("test_tool")
	mockProvider.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(finalResponse, nil)

	answer, _ := core.ProcessQuestion(mockSession, mockQuestion)
	if answer == nil {
		t.Error("expected answer, got nil")
	}
}
