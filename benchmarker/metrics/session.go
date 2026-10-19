package metrics

import "time"

// MetricSessionRunningTime  "session_run_time"
type SessionRunningTimeMetric struct {
	SessionID string
	StartTime time.Time
	EndTime   time.Time
}

// MetricSessionTokenUsage  "session_token_usage"
type SessionTokenUsageMetric struct {
	SessionID  string
	Model      string
	TokenUsage uint64
	TimeStamp  int64
}

// MetricQueryCount  "query_count"
type QueryCountMetric struct {
	SessionID  string
	QueryCount int64
	TimeStamp  int64
}

// MetricQueryTime  "query_time"
type QueryTimeMetric struct {
	SessionID string
	QueryID   string
	StartTime time.Time
	EndTime   time.Time
}

// MetricQueryTokenUsage  "query_token_usage"
type QueryTokenUsageMetric struct {
	SessionID  string
	QueryID    string
	Model      string
	TokenUsage uint64
	TimeStamp  int64
}

// MetricQueryReActRound  "query_round"
type QueryReActRoundMetric struct {
	SessionID  string
	QueryID    string
	ReActRound int
	TimeStamp  int64
}
