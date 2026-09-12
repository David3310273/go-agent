package core

type ServiceProvider interface {
	// rpc or http service collection...
}

type UI interface {
	// show message
	Render(string)
	// show error message
	RenderSystemMessage([]Diagnostic)
	// display welcome message
	Welcome()
	// get user input
	GetUserInput() string
}

type SingalManager interface {
	// graceful quit
	GracefulQuit()
}

// user info manager
type UserManager interface {
	// auth user from token
	AuthUser(token string) *Diagnostic
	// check user plan
	CheckUserPlan(AgentConfig) *Diagnostic
	// app config: rules that what plan can use what model
	// load model providers depending on:
	// app config:system provided model
	// user plan
	SetModelProviders(modelConfigs []ModelConfig, appConfig AppConfig) *Diagnostic
}

// agent app, with UI, user identity and backend server
type AgentApp interface {
	UI
	I18n
	AgentServer
	// responsible for user identity, privileges and plan management
	UserManager
}
