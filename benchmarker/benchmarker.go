package simple

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/David3310273/go-agent/benchmarker/metrics"
	"github.com/David3310273/go-agent/benchmarker/server"
	core "github.com/David3310273/go-agent/core"
)

const (
	// defaultPullInterval 未在 config.intervals 中配置时的默认拉取间隔
	defaultPullInterval = 15 * time.Second
)

// pullRequest 拉取请求，benchmarker 定时发送到 agent 的 pull channel，
// agent 收到后计算当前 runtime 并通过 listening channel 推回。
type pullRequest struct {
	MetricType core.MetricType
	TimeStamp  int64
}

// pullMetric 定义需要定时拉取的指标类型。
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

	// pullChans 由 agent 提供，benchmarker 定时往里面发 pullRequest
	pullChans map[core.MetricType]chan core.BenchmarkEvent[any]
}

// OnEvent：
//   - 对每个 listening channel 启动 goroutine 消费事件 → 写入 Prometheus
//   - 保存 pull channel，供 periodicCollect 定时发送拉取请求
func (b *Benchmarker) OnEvent(obj core.Observable) *core.Diagnostic {
	listeningChans := obj.GetBenchmarkListeningChannels()
	pullChans := obj.GetBenchmarkPullChannels()

	core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "[benchmarker] OnEvent: listening channels=%d, pull channels=%d",
		len(listeningChans), len(pullChans))

	// 启动 listening channel 消费者
	for metricType, ch := range listeningChans {
		core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] OnEvent: subscribing to listening channel metric=%s buffer=%d",
			metricType, cap(ch))
		go b.consumeChannel(metricType, ch)
	}

	// 保存 pull channel，供定时采集使用
	b.pullChans = pullChans
	if len(pullChans) > 0 {
		for mt := range pullChans {
			core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, mt)), "[benchmarker] OnEvent: registered pull channel metric=%s", mt)
		}
	}

	return nil
}

// consumeChannel 持续从 listening channel 读取事件，分发给对应的 Record* 方法。
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

// dispatch 根据 MetricType 将事件数据路由到对应的 Record* 方法。
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

	// 启动 metrics HTTP server
	go func() {
		core.LogInfo(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "Metrics server listening on %s", b.metricsServer.Port())
		if err := b.metricsServer.Start(); err != nil {
			core.LogError(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, "benchmarker")), "Metrics server error: %v", err)
		}
	}()

	// 启动定时采集：每个 pull metric 用各自的 interval 向 agent 拉取 runtime 数据
	for _, mt := range pullMetrics {
		go b.periodicCollect(mt)
	}

	return nil
}

// periodicCollect 以 metric 独立的 interval 定时向 agent 的 pull channel 发送拉取请求，
// 触发 agent 计算当前 runtime 并通过 listening channel 推回。
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

// pullRunningTime 向指定 metric 的 pull channel 发送拉取请求，
// agent 收到后计算 runtime 并推回 listening channel。
// 使用 SendBenchmarkEvent 非阻塞发送，若 agent 未消费该 channel 则静默丢弃。
func (b *Benchmarker) pullRunningTime(metricType core.MetricType, now time.Time) {
	ch, ok := b.pullChans[metricType]
	if !ok {
		core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] pullRunningTime: no pull channel for metric=%s", metricType)
		return
	}

	core.LogDebug(filepath.Join(b.Configs.RootPath, b.Configs.OutputPath, fmt.Sprintf(b.Configs.FilenameFormat, string(metricType))), "[benchmarker] pullRunningTime: sending pull request to metric=%s", metricType)
	core.SendBenchmarkEvent(ch, core.BenchmarkEvent[any]{
		Type:      metricType,
		TimeStamp: now.Unix(),
		Data:      pullRequest{MetricType: metricType, TimeStamp: now.Unix()},
	})
}

func (b *Benchmarker) BeforeStop() []core.Diagnostic {
	return nil
}

func (b *Benchmarker) Stop() []core.Diagnostic {
	b.cancel()
	return nil
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
	}
}

// MetricsServer 返回内部的 MetricsServer，供外部写入指标数据。
func (b *Benchmarker) MetricsServer() *server.MetricsServer {
	return b.metricsServer
}
