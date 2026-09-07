package core

type LanguageType string

const (
	Language_EN = "English"
	Language_ZH = "Chinese"
)

type I18nConfig struct {
	DefaultLanguage string `json:"defaultLanguage"`
	FilePath        string `json:"filePath"`
}

// interface for UI message translate
type I18n interface {
	Translate(key string, targetLanguage LanguageType) string
	SetConfig(config I18nConfig) *Diagnostic
}
