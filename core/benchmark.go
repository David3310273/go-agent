package core

import (
	"encoding/json"
)

const (
	// DefaultBenchmarkChannelBufferSize is the default buffer size for benchmark channels,
	// preventing the agent/session from being blocked when the benchmarker is not started or is consuming slowly.
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

// SendBenchmarkEvent non-blockingly sends an event to the benchmark channel.
// If the channel is full or has no consumer (benchmarker not started), the event is silently dropped.
func SendBenchmarkEvent(ch chan BenchmarkEvent[any], event BenchmarkEvent[any]) {
	select {
	case ch <- event:
	default:
	}
}

func StartBenchmark(benchmarker WorkFlow) *Diagnostic {
	if errs := StartWorkFlow(benchmarker); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	return nil
}

func StopBenchmark(benchmarker WorkFlow) *Diagnostic {
	if errs := StopWorkFlow(benchmarker); len(errs) > 0 {
		return &Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeAgentCoreConfigError,
		}
	}

	return nil
}
