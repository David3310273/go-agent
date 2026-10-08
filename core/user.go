package core

import "time"

type Plan struct {
	Name     PlanType `json:"name"`
	MaxUser  int      `json:"maxUser"`
	MaxToken int64    `json:"maxToken"`
	// start date
	StartUseDate time.Time `json:"startDate"`
	// expiry date
	ExpiryDate time.Time `json:"expiryDate"`
	// available models
	AvailableModels []string `json:"availableModels"`
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
