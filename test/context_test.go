// test cases for core.Context and core.SessionContext interfaces
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockContext satisfies core.Context
var _ core.Context = (*testmock.MockContext)(nil)

// compile-time check: MockSessionContext satisfies core.SessionContext
var _ core.SessionContext = (*testmock.MockSessionContext)(nil)

// =============================================================================
// Context interface tests
// =============================================================================

func TestMockContext_SetLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	agentConfig := core.AgentConfig{LogPath: "/var/log/agent.log"}

	mockContext.EXPECT().SetLogger(agentConfig).Return(nil)

	err := mockContext.SetLogger(agentConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_SetLanguage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	mockContext.EXPECT().SetLanguage(gomock.Eq(core.LanguageType("Chinese")))

	mockContext.SetLanguage(core.LanguageType("Chinese"))
}

func TestMockContext_SetHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	historyConfig := core.HistoryConfig{BufferSize: 1024}

	mockContext.EXPECT().SetHistory(historyConfig).Return(nil)

	err := mockContext.SetHistory(historyConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedHistory := []byte(`{"history": "data"}`)

	mockContext.EXPECT().GetHistory().Return(expectedHistory)

	history := mockContext.GetHistory()
	if string(history) != string(expectedHistory) {
		t.Errorf("expected history %s, got %s", expectedHistory, history)
	}
}

// updated LoadConfigs to new no-arg signature
func TestMockContext_LoadConfigs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedConfig := core.AgentCoreConfig{
		Agent: core.AgentConfig{Version: "1.0.0"},
	}

	mockContext.EXPECT().LoadConfigs().Return(expectedConfig)

	config := mockContext.LoadConfigs()
	if config.Agent.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", config.Agent.Version)
	}
}

func TestMockContext_SetPrompt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	promptConfig := core.PromptConfig{Paths: []string{"/path/to/prompt.md"}}

	mockContext.EXPECT().SetPrompt(promptConfig).Return(nil)

	err := mockContext.SetPrompt(promptConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetPrompt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedPrompt := []byte("You are a helpful assistant.")

	mockContext.EXPECT().GetPrompt().Return(expectedPrompt)

	prompt := mockContext.GetPrompt()
	if string(prompt) != string(expectedPrompt) {
		t.Errorf("expected prompt %s, got %s", expectedPrompt, prompt)
	}
}

func TestMockContext_SetKnowledgeBase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	kbConfig := core.KnowledgeBaseConfig{Paths: []string{"/path/to/kb"}}

	mockContext.EXPECT().SetKnowledgeBase(kbConfig).Return(nil)

	err := mockContext.SetKnowledgeBase(kbConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetKnowledgeBase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedKB := []byte(`{"knowledge": "base"}`)

	mockContext.EXPECT().GetKnowledgeBase().Return(expectedKB)

	kb := mockContext.GetKnowledgeBase()
	if string(kb) != string(expectedKB) {
		t.Errorf("expected kb %s, got %s", expectedKB, kb)
	}
}

func TestMockContext_SetSkills(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	skillConfig := core.SkillConfig{Paths: []string{"/path/to/skills"}}

	mockContext.EXPECT().SetSkills(skillConfig).Return(nil)

	err := mockContext.SetSkills(skillConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetSkills(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedSkills := []byte(`{"skills": []}`)

	mockContext.EXPECT().GetSkills().Return(expectedSkills)

	skills := mockContext.GetSkills()
	if string(skills) != string(expectedSkills) {
		t.Errorf("expected skills %s, got %s", expectedSkills, skills)
	}
}

func TestMockContext_GetSessionConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedConfig := core.SessionConfig{ReActMaxRounds: 10}

	mockContext.EXPECT().GetSessionConfig().Return(expectedConfig)

	config := mockContext.GetSessionConfig()
	if config.ReActMaxRounds != 10 {
		t.Errorf("expected ReActMaxRounds 10, got %d", config.ReActMaxRounds)
	}
}

func TestMockContext_GetModelProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	mockContext.EXPECT().GetModelProviders().Return(nil)

	providers := mockContext.GetModelProviders()
	if providers != nil {
		t.Errorf("expected nil providers, got %v", providers)
	}
}

func TestMockContext_GetToolsConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedTools := []core.ToolConfig{
		{Name: "filewriter", Schema: "filewriter.schema.json"},
	}

	mockContext.EXPECT().GetToolsConfig().Return(expectedTools)

	tools := mockContext.GetToolsConfig()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool config, got %d", len(tools))
	}
}

// =============================================================================
// SessionContext interface tests
// =============================================================================

func TestMockSessionContext_GenerateFinalContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionCtx := testmock.NewMockSessionContext(ctrl)
	mockQuestion := testmock.NewMockQuestion(ctrl)
	expectedContext := "This is the final context."

	mockSessionCtx.EXPECT().GenerateFinalContext(mockQuestion).Return(expectedContext)

	context := mockSessionCtx.GenerateFinalContext(mockQuestion)
	if context != expectedContext {
		t.Errorf("expected context '%s', got %s", expectedContext, context)
	}
}
