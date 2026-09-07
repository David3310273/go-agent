package simple

import (
	"time"

	"github.com/David3310273/go-agent/core"
)

// only related data type for event

type AgentEventData struct {
	SessionID string        `json:"sessionID"`
	Question  core.Question `json:"question"`
	AgentID   string        `json:"agentID"`
	Message   string        `json:"message"`
}

type AgentEventTimeData struct {
	SnapshotTime time.Time `json:"snapshotTime"`
	AgentID      string    `json:"agentID"`
	Message      string    `json:"message"`
}

type SessionEventTimeData struct {
	SnapshotTime time.Time `json:"snapshotTime"`
	SessionID    string    `json:"sessionID"`
	Message      string    `json:"message"`
}
