package core

type LanguageType string

const (
	Language_EN = "en_US"
	Language_ZH = "zh_CN"
)

type I18nConfig struct {
	DefaultLanguage string `json:"defaultLanguage"`
	FilePath        string `json:"filePath"`
}

// interface for UI message translate
type I18n interface {
	Translate(key MessageCode, targetLanguage LanguageType) string
}
