package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector 将所有自定义 metric 注册到 Prometheus，并提供 /metrics 端点。
// 它与 core 包完全解耦，core 中不引入任何 Prometheus 依赖。
type PrometheusCollector struct {
	// ---- session 级别 ----
	sessionRunningTime *prometheus.GaugeVec
	sessionTokenUsage  *prometheus.CounterVec
	queryCount         *prometheus.GaugeVec
	queryTime          *prometheus.HistogramVec
	queryTokenUsage    *prometheus.CounterVec
	queryReActRound    *prometheus.HistogramVec

	// ---- agent 级别 ----
	agentRunningTime *prometheus.GaugeVec
}

// NewPrometheusCollector 创建并注册所有 Prometheus 指标。
func NewPrometheusCollector() *PrometheusCollector {
	pc := &PrometheusCollector{
		// MetricSessionRunningTime
		sessionRunningTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "go_agent_session_running_time_seconds",
				Help: "Session total running time in seconds.",
			},
			[]string{"session_id"},
		),

		// MetricSessionTokenUsage
		sessionTokenUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "go_agent_session_token_usage_total",
				Help: "Total token usage per session, partitioned by model.",
			},
			[]string{"session_id", "model"},
		),

		// MetricQueryCount
		queryCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "go_agent_session_query_count",
				Help: "Current number of queries in a session.",
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
		pc.sessionTokenUsage,
		pc.queryCount,
		pc.queryTime,
		pc.queryTokenUsage,
		pc.queryReActRound,
		pc.agentRunningTime,
	)
}

// Handler 返回 /metrics 端点使用的 http.Handler。
func (pc *PrometheusCollector) Handler() http.Handler {
	return promhttp.Handler()
}

// ---- Record 方法：接收领域结构体，写入 Prometheus ----

func (pc *PrometheusCollector) RecordSessionRunningTime(m SessionRunningTimeMetric) {
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	pc.sessionRunningTime.WithLabelValues(m.SessionID).Set(duration)
}

func (pc *PrometheusCollector) RecordSessionTokenUsage(m SessionTokenUsageMetric) {
	pc.sessionTokenUsage.WithLabelValues(m.SessionID, m.Model).Add(float64(m.TokenUsage))
}

func (pc *PrometheusCollector) RecordQueryCount(m QueryCountMetric) {
	pc.queryCount.WithLabelValues(m.SessionID).Set(float64(m.QueryCount))
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
