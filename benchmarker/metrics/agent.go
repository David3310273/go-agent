package metrics

import "time"

// MetricAgentRunningTime  "agent_run_time"
type AgentRunningTimeMetric struct {
	AgentID   string
	StartTime time.Time
	EndTime   time.Time
}

// MetricAgentTokenUsage  "agent_token_usage"
type AgentTokenUsageMetric struct {
	AgentID    string
	Model      string
	TokenUsage uint64
	TimeStamp  int64
}

// MetricAgentSessionCount  "agent_session_count"
type AgentSessionCountMetric struct {
	AgentID      string
	SessionCount int64
	TimeStamp    int64
}
