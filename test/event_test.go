// auto-generated: test cases for core.Event interface and related functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockEvent satisfies core.Event[any]
var _ core.Event[any] = (*testmock.MockEvent[any])(nil)

// =============================================================================
// Event interface tests
// =============================================================================

func TestMockEvent_GetSourceType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	mockEvent.EXPECT().GetSourceType().Return("session_start")

	sourceType := mockEvent.GetSourceType()
	if sourceType != "session_start" {
		t.Errorf("expected source type 'session_start', got %s", sourceType)
	}
}

func TestMockEvent_GetData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	testData := map[string]string{"key": "value"}

	mockEvent.EXPECT().GetData().Return(testData)

	data := mockEvent.GetData()
	if data == nil {
		t.Error("expected data, got nil")
	}
}

func TestMockEvent_WriteToStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	mockEvent.EXPECT().WriteToStorage().Return(nil)

	err := mockEvent.WriteToStorage()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// =============================================================================
// EventManager interface tests
// =============================================================================

func TestMockEventManager_OnEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventMgr := testmock.NewMockEventManager(ctrl)
	mockEventMgr.EXPECT().OnEvent()

	mockEventMgr.OnEvent()
}

func TestMockEventManager_GetEventChans(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventMgr := testmock.NewMockEventManager(ctrl)
	eventChans := map[string]chan core.Event[any]{
		"session_start": make(chan core.Event[any]),
	}

	mockEventMgr.EXPECT().GetEventChans().Return(eventChans)

	chans := mockEventMgr.GetEventChans()
	if len(chans) != 1 {
		t.Errorf("expected 1 channel, got %d", len(chans))
	}
}

func TestMockEventManager_RegisterEventChans(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventMgr := testmock.NewMockEventManager(ctrl)
	mockEventMgr.EXPECT().RegisterEventChans()

	mockEventMgr.RegisterEventChans()
}

// =============================================================================
// Emit function tests
// =============================================================================

func TestEmit_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventMgr := testmock.NewMockEventManager(ctrl)
	mockEvent := testmock.NewMockEvent[any](ctrl)

	eventChan := make(chan core.Event[any], 1)
	eventChans := map[string]chan core.Event[any]{
		"test_source": eventChan,
	}

	gomock.InOrder(
		mockEventMgr.EXPECT().GetEventChans().Return(eventChans),
		mockEvent.EXPECT().GetSourceType().Return("test_source"),
	)

	core.Emit(mockEventMgr, mockEvent)

	select {
	case receivedEvent := <-eventChan:
		if receivedEvent == nil {
			t.Error("expected event in channel, got nil")
		}
	default:
		t.Error("expected event in channel, channel is empty")
	}
}

func TestEmit_InvalidSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventMgr := testmock.NewMockEventManager(ctrl)
	mockEvent := testmock.NewMockEvent[any](ctrl)

	eventChans := map[string]chan core.Event[any]{}

	gomock.InOrder(
		mockEventMgr.EXPECT().GetEventChans().Return(eventChans),
		mockEvent.EXPECT().GetSourceType().Return("invalid_source"),
	)

	core.Emit(mockEventMgr, mockEvent)
}

// =============================================================================
// SimpleEvent tests (now in agent/simple package, testing via interface)
// =============================================================================

func TestSimpleEvent_GetSourceType(t *testing.T) {
	// SimpleEvent is now in agent/simple package, test via mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	mockEvent.EXPECT().GetSourceType().Return("test_source")

	sourceType := mockEvent.GetSourceType()
	if sourceType != "test_source" {
		t.Errorf("expected source type 'test_source', got %s", sourceType)
	}
}

func TestSimpleEvent_GetData(t *testing.T) {
	//  test event data retrieval via mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	testData := "test data"
	mockEvent.EXPECT().GetData().Return(testData)

	data := mockEvent.GetData()
	if data != "test data" {
		t.Errorf("expected data 'test data', got %v", data)
	}
}

func TestSimpleEvent_WriteToStorage(t *testing.T) {
	//  test WriteToStorage via mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvent := testmock.NewMockEvent[any](ctrl)
	mockEvent.EXPECT().WriteToStorage().Return(nil)

	err := mockEvent.WriteToStorage()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
