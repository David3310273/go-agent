// auto-add: test cases for the core.Harness tool confirm batch.
// A batch of destructive tool calls is confirmed in one request now, so HandleUserQuestion records
// every answer at once and HandleUserToolConfirm replays the approved calls concurrently while a
// declined call only produces its tool response.
package test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// batchConfirmQuestion is a tool_confirm question carrying a batch of answers.
// auto-add: the app layer builds the same shape out of the HTTP request, the harness only needs
// the core.ToolConfirmable half of it.
type batchConfirmQuestion struct {
	simple.SimpleQuestion
	confirms []core.ToolConfirmAnswer
}

var _ core.Question = (*batchConfirmQuestion)(nil)
var _ core.ToolConfirmable = (*batchConfirmQuestion)(nil)

func (q *batchConfirmQuestion) GetType() core.QuestionType { return core.QuestionTypeToolConfirm }

func (q *batchConfirmQuestion) GetConfirmAnswers() []core.ToolConfirmAnswer { return q.confirms }

func (q *batchConfirmQuestion) ValidateConfirmAnswers() bool {
	if len(q.confirms) == 0 {
		return false
	}
	for _, confirm := range q.confirms {
		if confirm.ToolCallID == "" {
			return false
		}
		if confirm.Answer != "Yes" && confirm.Answer != "No" {
			return false
		}
	}
	return true
}

// TestHandleUserToolConfirm_RunsApprovedBatchConcurrently covers the replay of a confirmed batch:
// the calls answered Yes run concurrently, a call answered No only produces a declined tool
// response and never runs, the messages come back in the order the user answered, and the pending
// state of the whole batch is cleared afterwards.
func TestHandleUserToolConfirm_RunsApprovedBatchConcurrently(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := newTestParentSession(t, ctrl)

	const toolSleep = 200 * time.Millisecond

	// running counts the runners currently inside a tool, maxRunning is the highest overlap seen
	var running, maxRunning int32
	approvedTool := testmock.NewMockTool(ctrl)
	approvedTool.EXPECT().GetName().Return("approved_tool").AnyTimes()
	approvedTool.EXPECT().Validate(gomock.Any()).Return(nil).AnyTimes()
	approvedTool.EXPECT().GetRunner().Return(func(args map[string]any) (string, *core.Diagnostic) {
		current := atomic.AddInt32(&running, 1)
		for {
			previous := atomic.LoadInt32(&maxRunning)
			if current <= previous || atomic.CompareAndSwapInt32(&maxRunning, previous, current) {
				break
			}
		}
		time.Sleep(toolSleep)
		atomic.AddInt32(&running, -1)
		return fmt.Sprintf("approved %v", args["name"]), nil
	}).AnyTimes()

	// the declined call must never run, so no Validate and no GetRunner expectation is registered
	declinedTool := testmock.NewMockTool(ctrl)
	declinedTool.EXPECT().GetName().Return("declined_tool").AnyTimes()

	// the batch as the reAct loop stored it, keyed by the tool call ID
	session.SetPendingMCPToolCall("call-approved-1", &core.PendingMCPToolCall{
		Args:       map[string]any{"name": "one"},
		Tool:       approvedTool,
		ToolCallID: "call-approved-1",
	})
	session.SetPendingMCPToolCall("call-declined", &core.PendingMCPToolCall{
		Args:       map[string]any{"name": "two"},
		Tool:       declinedTool,
		ToolCallID: "call-declined",
	})
	session.SetPendingMCPToolCall("call-approved-2", &core.PendingMCPToolCall{
		Args:       map[string]any{"name": "three"},
		Tool:       approvedTool,
		ToolCallID: "call-approved-2",
	})

	question := &batchConfirmQuestion{
		SimpleQuestion: *simple.NewSimpleQuestion(
			"", session.GetID(), "", nil, nil, false, false, core.QuestionTypeToolConfirm,
		),
		confirms: []core.ToolConfirmAnswer{
			{ToolCallID: "call-approved-1", Answer: "Yes"},
			{ToolCallID: "call-declined", Answer: "No"},
			{ToolCallID: "call-approved-2", Answer: "Yes"},
		},
	}

	// the whole batch is recorded before any tool runs
	userMessage, userDiag := simple.SimpleHarnessInstance.HandleUserQuestion(session, question)
	if userMessage == nil {
		t.Fatalf("expected the confirmation user message, got nil (diagnostic: %s)", userDiag.Message)
	}
	for _, toolCallID := range []string{"call-approved-1", "call-declined", "call-approved-2"} {
		if !session.IsToolConfirmed(toolCallID) {
			t.Errorf("expected the answer for %s to be recorded", toolCallID)
		}
	}

	started := time.Now()
	toolMessages := simple.SimpleHarnessInstance.HandleUserToolConfirm(session, question)
	elapsed := time.Since(started)

	expected := []struct {
		toolCallID string
		content    string
	}{
		{"call-approved-1", "approved one"},
		{"call-declined", "Tool call call-declined has been declined by user"},
		{"call-approved-2", "approved three"},
	}
	if len(toolMessages) != len(expected) {
		t.Fatalf("expected %d tool messages, got %d", len(expected), len(toolMessages))
	}
	for i, want := range expected {
		if toolMessages[i].Role != core.RoleTool {
			t.Errorf("tool message %d: expected role %v, got %v", i, core.RoleTool, toolMessages[i].Role)
		}
		if toolMessages[i].ToolCallID != want.toolCallID {
			t.Errorf("tool message %d: expected tool call ID %s, got %s", i, want.toolCallID, toolMessages[i].ToolCallID)
		}
		if toolMessages[i].Content != want.content {
			t.Errorf("tool message %d: expected content %q, got %q", i, want.content, toolMessages[i].Content)
		}
	}

	// the two approved calls have to be inside their tool at the same time
	if got := atomic.LoadInt32(&maxRunning); got != 2 {
		t.Errorf("expected the 2 approved calls to overlap, only %d ran at the same time", got)
	}
	// a serial replay would take 2 * toolSleep
	if elapsed >= 2*toolSleep {
		t.Errorf("expected the approved calls to run concurrently, it took %v", elapsed)
	}

	// the pending state and the recorded answers of the whole batch are gone
	for _, toolCallID := range []string{"call-approved-1", "call-declined", "call-approved-2"} {
		if pending := session.GetPendingMCPToolCall(toolCallID); pending != nil {
			t.Errorf("expected the pending call %s to be deleted", toolCallID)
		}
		if session.IsToolConfirmed(toolCallID) {
			t.Errorf("expected the confirmed answer %s to be cleared", toolCallID)
		}
	}
}

// TestHandleUserToolConfirm_SkipsUnknownToolCallID covers an answer whose pending call is gone, it
// is skipped instead of producing a tool response with an empty tool_call ID.
func TestHandleUserToolConfirm_SkipsUnknownToolCallID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := newTestParentSession(t, ctrl)

	question := &batchConfirmQuestion{
		SimpleQuestion: *simple.NewSimpleQuestion(
			"", session.GetID(), "", nil, nil, false, false, core.QuestionTypeToolConfirm,
		),
		confirms: []core.ToolConfirmAnswer{
			{ToolCallID: "call-unknown", Answer: "Yes"},
		},
	}

	if toolMessages := simple.SimpleHarnessInstance.HandleUserToolConfirm(session, question); len(toolMessages) != 0 {
		t.Errorf("expected no tool message for an unknown tool call ID, got %d", len(toolMessages))
	}

	// with nothing pending the batch is reported as already confirmed, so the reAct loop is skipped
	userMessage, userDiag := simple.SimpleHarnessInstance.HandleUserQuestion(session, question)
	if userMessage != nil {
		t.Error("expected no user message when every answer of the batch is unknown")
	}
	if userDiag.Code != core.MessageCodeToolAlreadyConfirmed {
		t.Errorf("expected the already confirmed code, got %v", userDiag.Code)
	}
}
