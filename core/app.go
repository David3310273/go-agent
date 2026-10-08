package core

type ServiceProvider interface {
	// rpc or http service collection...
}

type SingalManager interface {
	// graceful quit
	GracefulQuit()
}

// agent app, with UI, user identity and backend server
type AgentApp interface {
	UI
	I18n
	AgentServer
	// responsible for user identity, privileges and plan management
	UserManager
}
