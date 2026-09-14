// test cases for core.Context interface
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockContext satisfies core.Context
var _ core.Context = (*testmock.MockContext)(nil)

// =============================================================================
// Context interface tests
// =============================================================================

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
	kbConfig := []core.KnowledgeBaseConfig{{RootPath: "/path/to/kb"}}

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
	expectedKB := []core.KnowledgeBase[any]{}

	mockContext.EXPECT().GetKnowledgeBase().Return(expectedKB)

	kb := mockContext.GetKnowledgeBase()
	if kb == nil {
		t.Errorf("expected kb, got nil")
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

func TestMockContext_SetProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	mockContext.EXPECT().SetProviders(nil).Return(nil)

	err := mockContext.SetProviders(nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_SetToolsConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	toolsConfig := []core.ToolConfig{{Name: "test"}}

	mockContext.EXPECT().SetToolsConfig(toolsConfig).Return(nil)

	err := mockContext.SetToolsConfig(toolsConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetToolsConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedTools := []core.ToolConfig{
		{Name: "filewriter"},
	}

	mockContext.EXPECT().GetToolsConfig().Return(expectedTools)

	tools := mockContext.GetToolsConfig()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool config, got %d", len(tools))
	}
}
