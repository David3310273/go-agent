package core

import "time"

type Plan struct {
	Name     PlanType `json:"name"`
	MaxToken uint64   `json:"maxToken"`
	// start date
	StartUseDate time.Time `json:"startDate"`
	// expiry date
	ExpiryDate time.Time `json:"expiryDate"`
	// available models
	AvailableModels []string `json:"availableModels"`
	// max rounds of reAct
	MaxRoundsReAct int `json:"maxRoundsReAct"`
	// max memory size
	MemoryWindowSize uint64 `json:"memoryWindowSize"`
}

// user info manager
type UserManager interface {
	// auth user from token
	AuthUser(token string) *Diagnostic
	// check user plan
	GetUserPlan(userID string) (Plan, *Diagnostic)
	// app config: rules that what plan can use what model
	// load model providers depending on:
	// app config:system provided model
	// user plan
	CheckUserPlan(userInfo any, plan Plan) *Diagnostic
}
