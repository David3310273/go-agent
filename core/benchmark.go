package core

import "context"

type StatEventType string

const (
	StatEventSessionRunningTime StatEventType = "session_run_time"
	StatEventSessionTokenUsage  StatEventType = "session_token_usage"
)

type Observable interface {
	// get listening channels
	GetBenchmarkListeningChannels() map[string]chan StatEvent[any]
	// close listening channels
	CloseBenchmarkListeningChannels()
}

func onEvent(ctx context.Context, obj Observable) *Diagnostic {
	return nil
}

func StartBenchmark(ctx context.Context, obj Observable) *Diagnostic {
	go onEvent(ctx, obj)
	return nil
}

// stop benchmark
func StopBenchmark(cancel context.CancelFunc, obj Observable) *Diagnostic {
	// inform obj to close benchmark channels
	obj.CloseBenchmarkListeningChannels()
	// stop benchmark goroutine
	cancel()
	return nil
}

type StatEvent[T any] struct {
	Event[T]
	StartTime uint64
	EndTime   uint64
	Type      StatEventType
	Data      T
}

type AgentStatisticsInventory struct {
	// total token usage
	TokenUsage uint64
	// total sessions
	ModelUseCount map[string]uint64
	// total sub sessions
	SessionCount uint64
	// total session time cost
	SesstionTime uint64
}

/*
	benchmark is a independent component, should not be a part of agent/session,
	using observer pattern instead
*/

type Benchmarker struct {
	Configs           BenchmarkerConfig
	ListeningChannels map[string]chan StatEvent[any]
}

func NewBenchmarker(obj Observable) *Benchmarker {
	return nil
}
