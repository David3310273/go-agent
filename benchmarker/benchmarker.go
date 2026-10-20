package simple

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/David3310273/go-agent/benchmarker/metrics"
	"github.com/David3310273/go-agent/benchmarker/server"
	core "github.com/David3310273/go-agent/core"
)

const (
	// defaultPullInterval is the default interval when not configured in config.intervals.
	defaultPullInterval = 15 * time.Second
)

// pullRequest is sent periodically by the benchmarker to the agent's pull channel.
// The agent calculates the current runtime on receipt and pushes it back via the listening channel.
type pullRequest struct {
	MetricType core.MetricType
	TimeStamp  int64
}

// pullMetric defines the metric types that need periodic collection.
var pullMetrics = []core.MetricType{
	core.MetricSessionRunningTime,
	core.MetricAgentRunningTime,
}

type ModelUsage struct {
	Model      string `json:"model"`
	TokenUsage uint64 `json:"tokenUsage"`
}

type SessionStatistics struct {
	QuestionCount int64  `json:"questionCount"`
	Usage         uint64 `json:"usage"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
}

type AgentStatistics struct {
	// total token usage
	UsageInfo map[string]ModelUsage
	// total sub sessions
	SessionInfo map[string]SessionStatistics
	// online time
	OnlineTime int64
}

var _ core.WorkFlow = (*Benchmarker)(nil)

/*
benchmark is a independent component, should not be a part of agent/session,
one benchmarker only can monitor one agent core, but with multiple metrics.
using observer pattern instead
*/
type Benchmarker struct {
	Configs       core.BenchmarkerConfig
	context       context.Context
	cancel        context.CancelFunc
	metricsServer *server.MetricsServer

	// pullChans stores pull channels grouped by observable ID, then by metric type.
	// Outer key: objID (string), value: *sync.Map[core.MetricType]chan core.BenchmarkEvent[any].
	pullChans sync.Map

	// cleanupCh receives observable IDs (sessionID/agentID) whose pull channels should be removed.
	cleanupCh chan string
}

// OnEvent：
//   - Starts a goroutine for each listening channel to consume events and write to Prometheus.
//   - Saves pull channels for periodicCollect to send pull requests periodically.
func (b *Benchmarker) OnEvent(obj core.Observable) *core.Diagnostic {
	listeningChans := obj.GetBenchmarkListeningChannels()
	pullChans := obj.GetBenchmarkPullChannels()
	objID := obj.GetObjectID()

	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "[benchmarker] OnEvent: id=%s listening channels=%d, pull channels=%d",
		objID, len(listeningChans), len(pullChans))

	// Start listening channel consumers.
	for metricType, ch := range listeningChans {
		core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] OnEvent: subscribing to listening channel metric=%s buffer=%d",
			metricType, cap(ch))
		go b.consumeChannel(metricType, ch)
	}

	// Merge pull channels for periodic collection, grouped by the observable's ID.
	inner, _ := b.pullChans.LoadOrStore(objID, &sync.Map{})
	innerMap := inner.(*sync.Map)
	for mt, ch := range pullChans {
		if _, loaded := innerMap.LoadOrStore(mt, ch); loaded {
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, mt)), "[benchmarker] OnEvent: pull channel already registered for objID=%s metric=%s, skipping", objID, mt)
		} else {
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, mt)), "[benchmarker] OnEvent: registered pull channel objID=%s metric=%s", objID, mt)
		}
	}

	return nil
}

// consumeChannel continuously reads events from the listening channel and dispatches them to the corresponding Record* methods.
func (b *Benchmarker) consumeChannel(metricType core.MetricType, ch chan core.BenchmarkEvent[any]) {
	collector := b.metricsServer.Collector()
	timeout := time.Duration(b.Configs.MaxTimeout) * time.Second

	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] consumeChannel: started goroutine for metric=%s timeout=%s",
		metricType, timeout)

	for {
		select {
		case <-b.context.Done():
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] consumeChannel: goroutine stopped for metric=%s (context done)", metricType)
			return
		case event, ok := <-ch:
			if !ok {
				core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] consumeChannel: channel closed for metric=%s", metricType)
				return
			}
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] consumeChannel: received event metric=%s timestamp=%d",
				metricType, event.TimeStamp)
			b.dispatch(collector, metricType, event.Data)
		case <-time.After(timeout):
			core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] consumeChannel: timeout for metric=%s after %s", metricType, timeout)
		}
	}
}

// dispatch routes event data to the appropriate Record* method based on MetricType.
func (b *Benchmarker) dispatch(collector *metrics.PrometheusCollector, metricType core.MetricType, data any) {
	switch metricType {
	case core.MetricSessionRunningTime:
		if m, ok := data.(metrics.SessionRunningTimeMetric); ok {
			core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: recording MetricSessionRunningTime sessionID=%s startTime=%v endTime=%v",
				m.SessionID, m.StartTime, m.EndTime)
			collector.RecordSessionRunningTime(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricSessionRunningTime, got type=%T", data)
		}
	case core.MetricQueryFailedCount:
		if m, ok := data.(metrics.QueryFailedCountMetric); ok {
			collector.RecordQueryFailedCount(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricQueryFailedCount, got type=%T", data)
		}
	case core.MetricQuerySuccessCount:
		if m, ok := data.(metrics.QuerySuccessCountMetric); ok {
			collector.RecordQuerySuccessCount(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricQuerySuccessCount, got type=%T", data)
		}
	case core.MetricQueryTime:
		if m, ok := data.(metrics.QueryTimeMetric); ok {
			collector.RecordQueryTime(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricQueryTime, got type=%T", data)
		}
	case core.MetricQueryTokenUsage:
		if m, ok := data.(metrics.QueryTokenUsageMetric); ok {
			collector.RecordQueryTokenUsage(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricQueryTokenUsage, got type=%T", data)
		}
	case core.MetricQueryReActRound:
		if m, ok := data.(metrics.QueryReActRoundMetric); ok {
			collector.RecordQueryReActRound(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricQueryReActRound, got type=%T", data)
		}
	case core.MetricAgentRunningTime:
		if m, ok := data.(metrics.AgentRunningTimeMetric); ok {
			core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: recording MetricAgentRunningTime agentID=%s startTime=%v endTime=%v",
				m.AgentID, m.StartTime, m.EndTime)
			collector.RecordAgentRunningTime(m)
		} else {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: type assertion failed for MetricAgentRunningTime, got type=%T", data)
		}
	default:
		core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] dispatch: unknown metric type=%s", metricType)
	}
}

func (b *Benchmarker) BeforeStart() []core.Diagnostic {
	return nil
}

func (b *Benchmarker) Start() []core.Diagnostic {
	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "[benchmarker] Start: starting metrics server on port %s, pull metrics=%v",
		b.metricsServer.Port(), pullMetrics)

	// Start the metrics HTTP server.
	go func() {
		core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "Metrics server listening on %s", b.metricsServer.Port())
		if err := b.metricsServer.Start(); err != nil {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "Metrics server error: %v", err)
		}
	}()

	// Start cleanup goroutine: receives observable IDs to unregister from pullChans.
	go b.cleanupLoop()

	// Start periodic collection: each pull metric uses its own interval to pull runtime data from the agent.
	for _, mt := range pullMetrics {
		go b.periodicCollect(mt)
	}

	return nil
}

// periodicCollect periodically sends pull requests to the agent's pull channel at the metric's interval,
// triggering the agent to compute the current runtime and push it back via the listening channel.
func (b *Benchmarker) periodicCollect(metricType core.MetricType) {
	interval := defaultPullInterval

	if v, ok := b.Configs.Intervals[string(metricType)]; ok && v > 0 {
		interval = time.Duration(v) * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "Periodic collector for %s started (interval=%s)", metricType, interval)

	for {
		select {
		case <-b.context.Done():
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "Periodic collector for %s stopped", metricType)
			return
		case now := <-ticker.C:
			b.pullRunningTime(metricType, now)
		}
	}
}

// pullRunningTime sends a pull request to all pull channels matching the given metric type.
// The agent/session computes the runtime on receipt and pushes it back via the listening channel.
// Uses SendBenchmarkEvent for non-blocking send; silently drops if the receiver is not consuming the channel.
func (b *Benchmarker) pullRunningTime(metricType core.MetricType, now time.Time) {
	b.pullChans.Range(func(objID, innerVal any) bool {
		innerMap := innerVal.(*sync.Map)
		if val, ok := innerMap.Load(metricType); ok {
			ch := val.(chan core.BenchmarkEvent[any])
			core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] pullRunningTime: sending pull request to objID=%s metric=%s", objID, metricType)
			core.SendBenchmarkEvent(ch, core.BenchmarkEvent[any]{
				Type:      metricType,
				TimeStamp: now.Unix(),
				Data:      pullRequest{MetricType: metricType, TimeStamp: now.Unix()},
			})
		}
		return true
	})
}

func (b *Benchmarker) BeforeStop() []core.Diagnostic {
	// drain and clear all remaining pull channel references.
	var remaining int
	b.pullChans.Range(func(key, value any) bool {
		b.pullChans.Delete(key)
		remaining++
		return true
	})

	if remaining > 0 {
		core.LogWarn(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "[benchmarker] BeforeStop: cleared %d remaining pull channel entries", remaining)
	}
	return nil
}

func (b *Benchmarker) Stop() []core.Diagnostic {
	b.cancel()
	return nil
}

// Unregister sends an observable ID to the cleanup goroutine so that all pull channels
// belonging to that ID are removed from pullChans. Non-blocking; if the cleanup channel is
// full the unregister request is silently dropped.
func (b *Benchmarker) Unregister(id string) {
	select {
	case b.cleanupCh <- id:
	default:
	}
}

// cleanupLoop listens on cleanupCh and removes all pullChans entries whose key
// suffix matches the received ID.
func (b *Benchmarker) cleanupLoop() {
	for {
		select {
		case <-b.context.Done():
			return
		case id := <-b.cleanupCh:
			b.removePullChansByID(id)
		}
	}
}

// removePullChansByID deletes all pull channels for the given observable ID.
func (b *Benchmarker) removePullChansByID(id string) {
	b.pullChans.Delete(id)
	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "[benchmarker] cleanupLoop: removed pull channels for objID=%s", id)
}

func NewBenchmarker(configs core.BenchmarkerConfig) *Benchmarker {
	ctx, cancel := context.WithCancel(context.Background())

	port := fmt.Sprintf(":%d", configs.Server.Port)
	metricsServer := server.NewMetricsServer(&configs, port)

	return &Benchmarker{
		Configs:       configs,
		context:       ctx,
		cancel:        cancel,
		metricsServer: metricsServer,
		cleanupCh:     make(chan string, 256),
	}
}

// MetricsServer returns the internal MetricsServer for external metric writes.
func (b *Benchmarker) MetricsServer() *server.MetricsServer {
	return b.metricsServer
}
