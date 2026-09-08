// auto-generated: test cases for core.I18n interface
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockI18n satisfies core.I18n
var _ core.I18n = (*testmock.MockI18n)(nil)

// =============================================================================
// I18n interface tests
// =============================================================================

func TestMockI18n_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockI18n := testmock.NewMockI18n(ctrl)
	mockI18n.EXPECT().Translate("hello", gomock.Eq(core.LanguageType("Chinese"))).Return("你好")

	result := mockI18n.Translate("hello", core.LanguageType("Chinese"))
	if result != "你好" {
		t.Errorf("expected '你好', got %s", result)
	}
}

func TestMockI18n_Translate_English(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockI18n := testmock.NewMockI18n(ctrl)
	mockI18n.EXPECT().Translate("你好", gomock.Eq(core.LanguageType("English"))).Return("hello")

	result := mockI18n.Translate("你好", core.LanguageType("English"))
	if result != "hello" {
		t.Errorf("expected 'hello', got %s", result)
	}
}

func TestMockI18n_SetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockI18n := testmock.NewMockI18n(ctrl)
	config := core.I18nConfig{
		DefaultLanguage: "English",
		FilePath:        "/path/to/i18n.json",
	}

	mockI18n.EXPECT().SetConfig(config).Return(nil)

	err := mockI18n.SetConfig(config)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMockI18n_SetConfig_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockI18n := testmock.NewMockI18n(ctrl)
	config := core.I18nConfig{
		FilePath: "/invalid/path",
	}
	expectedErr := &core.Diagnostic{
		Level:   core.SeverityError,
		Code:    core.MessageCodeConfigFileNotFound,
		Message: "i18n file not found",
	}

	mockI18n.EXPECT().SetConfig(config).Return(expectedErr)

	err := mockI18n.SetConfig(config)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
