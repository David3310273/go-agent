package core

import (
	json "encoding/json"
)

type EventType int

// unitformed event type, feel free to add more if needed
const (
	// session operation event
	SessionEventStart             = "session_start"
	SessionEventStop              = "session_stop"
	SessionEventSaveHistoryFailed = "session_save_history_failed"
	// session answer event
	SessionStartProcessQuestion = "session_start_process_question"
	SessionHistory              = "session_answer"
	SessionFinishQuestion       = "session_finish_question"
	// agent operation event
	AgentEventStart               = "agent_start"
	AgentEventStop                = "agent_stop"
	AgentEventPromptLoadFailed    = "prompt_load_failed"
	AgentEventDeleteSessionFailed = "session_delete_failed"
	AgentEventCreateSessionFailed = "session_create_failed"
)

type Event[T any] interface {
	Serializable

	GetSourceType() string
	GetData() T
}

type CommonEvent[T any] struct {
	SourceType string `json:"source_type"`
	Data       T      `json:"data"`
}

func (s CommonEvent[T]) GetSourceType() string {
	return s.SourceType
}

func (s CommonEvent[T]) GetData() any {
	return s.Data
}

// ToString returns JSON representation of the event
func (s CommonEvent[T]) ToString() string {
	data, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(data)
}

type EventManager interface {
	// OnEvent
	OnEvent()
	GetEventChans() map[string]chan Event[any]
	RegisterEventChans()
}

func Emit(manager EventManager, event Event[any]) {
	eventChans := manager.GetEventChans()
	if eventChan, ok := eventChans[event.GetSourceType()]; ok {
		eventChan <- event
	}
}
