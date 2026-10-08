// test cases for core.I18n interface
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
	mockI18n.EXPECT().Translate(core.MessageCodeSuccess, gomock.Eq(core.LanguageType(core.Language_ZH))).Return("成功")

	result := mockI18n.Translate(core.MessageCodeSuccess, core.Language_ZH)
	if result != "成功" {
		t.Errorf("expected '成功', got %s", result)
	}
}

func TestMockI18n_Translate_English(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockI18n := testmock.NewMockI18n(ctrl)
	mockI18n.EXPECT().Translate(core.MessageCodeSystemError, gomock.Eq(core.LanguageType(core.Language_EN))).Return("System Error")

	result := mockI18n.Translate(core.MessageCodeSystemError, core.Language_EN)
	if result != "System Error" {
		t.Errorf("expected 'System Error', got %s", result)
	}
}
