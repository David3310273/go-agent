package core

type UI interface {
	// show message
	Render(MessageCode, LanguageType) string
	// show error message
	RenderSystemMessage(*DiagnosticList, LanguageType) string
	// set welcome message
	Welcome() string
}
