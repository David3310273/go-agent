package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector registers all custom metrics with Prometheus and exposes the /metrics endpoint.
// It is completely decoupled from the core package; core introduces no Prometheus dependencies.
type PrometheusCollector struct {
	// ---- session level ----
	sessionRunningTime *prometheus.GaugeVec
	queryFailedCount   *prometheus.CounterVec
	querySuccessCount  *prometheus.CounterVec
	queryTime          *prometheus.HistogramVec
	queryTokenUsage    *prometheus.CounterVec
	queryReActRound    *prometheus.HistogramVec

	// ---- agent level ----
	agentRunningTime *prometheus.GaugeVec
}

// NewPrometheusCollector creates and registers all Prometheus metrics.
func NewPrometheusCollector() *PrometheusCollector {
	pc := &PrometheusCollector{
		// MetricSessionRunningTime
		sessionRunningTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "go_agent_session_running_time_seconds",
				Help: "Session total running time in seconds.",
			},
			[]string{"agent_id", "session_id"},
		),

		// MetricQueryFailedCount
		queryFailedCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "go_agent_session_query_failed_total",
				Help: "Total number of failed queries per session.",
			},
			[]string{"session_id"},
		),

		// MetricQuerySuccessCount
		querySuccessCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "go_agent_session_query_success_total",
				Help: "Total number of successful queries per session.",
			},
			[]string{"session_id"},
		),

		// MetricQueryTime
		queryTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "go_agent_query_time_seconds",
				Help:    "Per-query response time distribution.",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"session_id", "query_id"},
		),

		// MetricQueryTokenUsage
		queryTokenUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "go_agent_query_token_usage_total",
				Help: "Total token usage per query, partitioned by model.",
			},
			[]string{"session_id", "query_id", "model"},
		),

		// MetricQueryReActRound
		queryReActRound: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "go_agent_query_react_rounds",
				Help:    "ReAct rounds per query distribution.",
				Buckets: []float64{1, 2, 3, 5, 10, 20, 50},
			},
			[]string{"session_id", "query_id"},
		),

		// MetricAgentRunningTime
		agentRunningTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "go_agent_running_time_seconds",
				Help: "Agent total running time in seconds.",
			},
			[]string{"agent_id"},
		),
	}

	pc.register()
	return pc
}

func (pc *PrometheusCollector) register() {
	prometheus.MustRegister(
		pc.sessionRunningTime,
		pc.queryFailedCount,
		pc.querySuccessCount,
		pc.queryTime,
		pc.queryTokenUsage,
		pc.queryReActRound,
		pc.agentRunningTime,
	)
}

// Handler returns the http.Handler for the /metrics endpoint.
func (pc *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}

// ---- Record methods: receive domain structs and write to Prometheus ----

func (pc *PrometheusCollector) RecordSessionRunningTime(m SessionRunningTimeMetric) {
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	pc.sessionRunningTime.WithLabelValues(m.AgentID, m.SessionID).Set(duration)
}

func (pc *PrometheusCollector) RecordQueryFailedCount(m QueryFailedCountMetric) {
	pc.queryFailedCount.WithLabelValues(m.SessionID).Add(float64(m.Count))
}

func (pc *PrometheusCollector) RecordQuerySuccessCount(m QuerySuccessCountMetric) {
	pc.querySuccessCount.WithLabelValues(m.SessionID).Add(float64(m.Count))
}

func (pc *PrometheusCollector) RecordQueryTime(m QueryTimeMetric) {
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	pc.queryTime.WithLabelValues(m.SessionID, m.QueryID).Observe(duration)
}

func (pc *PrometheusCollector) RecordQueryTokenUsage(m QueryTokenUsageMetric) {
	pc.queryTokenUsage.WithLabelValues(m.SessionID, m.QueryID, m.Model).Add(float64(m.TokenUsage))
}

func (pc *PrometheusCollector) RecordQueryReActRound(m QueryReActRoundMetric) {
	pc.queryReActRound.WithLabelValues(m.SessionID, m.QueryID).Observe(float64(m.ReActRound))
}

func (pc *PrometheusCollector) RecordAgentRunningTime(m AgentRunningTimeMetric) {
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	pc.agentRunningTime.WithLabelValues(m.AgentID).Set(duration)
}
