// auto-generated: test cases for core.Observable interface and benchmark functions
package test

import (
	"context"
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
	channels := map[string]chan core.StatEvent[any]{
		"session_run_time": make(chan core.StatEvent[any]),
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

	mockObservable := testmock.NewMockObservable(ctrl)
	ctx := context.Background()

	diag := core.StartBenchmark(ctx, mockObservable)
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

	mockObservable := testmock.NewMockObservable(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	mockObservable.EXPECT().CloseBenchmarkListeningChannels()

	diag := core.StopBenchmark(cancel, mockObservable)
	if diag != nil {
		t.Errorf("expected nil diagnostic, got %v", diag)
	}

	// verify context is cancelled
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("expected context to be cancelled")
	}
}
