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

	mockContext.EXPECT().GetKnowledgeBase().Return(&expectedKB)

	kb := mockContext.GetKnowledgeBase()
	if kb == nil {
		t.Errorf("expected kb, got nil")
	}
}

func TestMockContext_AddSkill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	skillDef := core.SkillDefinition{Name: "test", Description: "test skill", Tools: []string{"tool1"}}

	mockContext.EXPECT().AddSkill(skillDef).Return(nil)

	err := mockContext.AddSkill(skillDef)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetSkills(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedSkills := &[]core.Skill{}

	mockContext.EXPECT().GetSkills().Return(expectedSkills)

	skills := mockContext.GetSkills()
	if skills == nil {
		t.Errorf("expected skills, got nil")
	}
	if len(*skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(*skills))
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

func TestMockContext_LoadTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	toolsConfig := []core.ToolConfig{{Name: "test"}}

	mockContext.EXPECT().LoadTools(toolsConfig).Return(nil)

	err := mockContext.LoadTools(toolsConfig)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockContext_GetTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContext := testmock.NewMockContext(ctrl)
	expectedTools := &[]core.Tool{}

	mockContext.EXPECT().GetTools().Return(expectedTools)

	tools := mockContext.GetTools()
	if tools == nil {
		t.Errorf("expected tools, got nil")
	}
	if len(*tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(*tools))
	}
}
