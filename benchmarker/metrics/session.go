package metrics

import "time"

// MetricSessionRunningTime  "session_run_time"
type SessionRunningTimeMetric struct {
	SessionID string
	StartTime time.Time
	EndTime   time.Time
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

// MetricQueryFailedCount  "sessionQueryFailedCount"
type QueryFailedCountMetric struct {
	SessionID string
	Count     int64
	TimeStamp int64
}

// MetricQuerySuccessCount  "sessionQuerySuccessCount"
type QuerySuccessCountMetric struct {
	SessionID string
	Count     int64
	TimeStamp int64
}
