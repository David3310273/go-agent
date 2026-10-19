package core

import (
	"encoding/json"
)

const (
	// DefaultBenchmarkChannelBufferSize 是 benchmark channel 的默认缓冲区大小，
	// 避免 benchmarker 未启动或消费慢时阻塞 agent/session 的发送。
	DefaultBenchmarkChannelBufferSize = 100
)

type MetricType string

const (
	// session metrics
	MetricSessionRunningTime MetricType = "sessionRunTime"
	MetricQueryFailedCount   MetricType = "sessionQueryFailedCount"
	MetricQuerySuccessCount  MetricType = "sessionQuerySuccessCount"
	MetricQueryTime          MetricType = "queryTime"
	MetricQueryTokenUsage    MetricType = "queryTokenUsage"
	MetricQueryReActRound    MetricType = "queryReActRound"

	// agent metrics
	MetricAgentRunningTime MetricType = "agentRunTime"
)

type Observable interface {
	// get listening channels where agent pushes data
	GetBenchmarkListeningChannels() map[MetricType]chan BenchmarkEvent[any]
	// get pull channels
	GetBenchmarkPullChannels() map[MetricType]chan BenchmarkEvent[any]
}

var _ Event[any] = (*BenchmarkEvent[any])(nil)

type BenchmarkEvent[T any] struct {
	TimeStamp int64
	Type      MetricType
	Data      T
}

func (e BenchmarkEvent[T]) GetSourceType() string {
	return string(e.Type)
}

func (e BenchmarkEvent[T]) GetData() T {
	return e.Data
}

func (e BenchmarkEvent[T]) ToString() string {
	msg, _ := json.Marshal(e)
	return string(msg)
}

// SendBenchmarkEvent 非阻塞地向 benchmark channel 发送事件。
// 如果 channel 已满或没有消费者（benchmarker 未启动），事件会被静默丢弃。
func SendBenchmarkEvent(ch chan BenchmarkEvent[any], event BenchmarkEvent[any]) {
	select {
	case ch <- event:
	default:
	}
}

func StartBenchmark(benchmarker WorkFlow) *Diagnostic {
	if errs := benchmarker.BeforeStart(); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	if errs := benchmarker.Start(); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	return nil
}

func StopBenchmark(benchmarker WorkFlow) *Diagnostic {
	if errs := benchmarker.BeforeStop(); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	if errs := benchmarker.Stop(); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	return nil
}
