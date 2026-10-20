// test cases for core.Observable interface and benchmark functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockObservable satisfies core.Observable
var _ core.Observable = (*testmock.MockObservable)(nil)

// =============================================================================
// Observable interface tests
// =============================================================================

func TestMockObservable_GetBenchmarkListeningChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockObservable := testmock.NewMockObservable(ctrl)
	channels := map[core.MetricType]chan core.BenchmarkEvent[any]{
		core.MetricSessionRunningTime: make(chan core.BenchmarkEvent[any]),
	}

	mockObservable.EXPECT().GetBenchmarkListeningChannels().Return(channels)

	chans := mockObservable.GetBenchmarkListeningChannels()
	if len(chans) != 1 {
		t.Errorf("expected 1 channel, got %d", len(chans))
	}
}

func TestMockObservable_CloseBenchmarkListeningChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockObservable := testmock.NewMockObservable(ctrl)
	mockObservable.EXPECT().CloseBenchmarkListeningChannels()

	mockObservable.CloseBenchmarkListeningChannels()
}

// =============================================================================
// StartBenchmark function tests
// =============================================================================

func TestStartBenchmark(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)

	mockWorkFlow.EXPECT().BeforeStart().Return(nil)
	mockWorkFlow.EXPECT().Start().Return(nil)

	diag := core.StartBenchmark(mockWorkFlow)
	if diag != nil {
		t.Errorf("expected nil diagnostic, got %v", diag)
	}
}

// =============================================================================
// StopBenchmark function tests
// =============================================================================

func TestStopBenchmark(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkFlow := testmock.NewMockWorkFlow(ctrl)

	mockWorkFlow.EXPECT().BeforeStop().Return(nil)
	mockWorkFlow.EXPECT().Stop().Return(nil)

	diag := core.StopBenchmark(mockWorkFlow)
	if diag != nil {
		t.Errorf("expected nil diagnostic, got %v", diag)
	}
}
