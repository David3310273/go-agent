package test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

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

func TestHandleUserToolConfirmMessage_RunsApprovedBatchConcurrently(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := newTestParentSession(t, ctrl)
	session.SetQueryContext(context.Background(), nil)

	const toolSleep = 200 * time.Millisecond

	var running, maxRunning int32
	approvedTool := testmock.NewMockTool(ctrl)
	approvedTool.EXPECT().GetName().Return("approved_tool").AnyTimes()
	approvedTool.EXPECT().Validate(gomock.Any()).Return(nil).AnyTimes()
	approvedTool.EXPECT().GetSandboxEnv().Return(core.SandboxSpec{}).AnyTimes()
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

	declinedTool := testmock.NewMockTool(ctrl)
	declinedTool.EXPECT().GetName().Return("declined_tool").AnyTimes()

	session.SetPendingToolCall("call-approved-1", &core.ConfirmToolCall{
		Args:       map[string]any{"name": "one"},
		Tool:       approvedTool,
		ToolCallID: "call-approved-1",
	})
	session.SetPendingToolCall("call-declined", &core.ConfirmToolCall{
		Args:       map[string]any{"name": "two"},
		Tool:       declinedTool,
		ToolCallID: "call-declined",
	})
	session.SetPendingToolCall("call-approved-2", &core.ConfirmToolCall{
		Args:       map[string]any{"name": "three"},
		Tool:       approvedTool,
		ToolCallID: "call-approved-2",
	})

	question := &batchConfirmQuestion{
		SimpleQuestion: *simple.NewSimpleQuestion(
			"", session.GetID(), "", nil, nil, false, false, core.QuestionTypeToolConfirm, 0,
		),
		confirms: []core.ToolConfirmAnswer{
			{ToolCallID: "call-approved-1", Answer: "Yes"},
			{ToolCallID: "call-declined", Answer: "No"},
			{ToolCallID: "call-approved-2", Answer: "Yes"},
		},
	}

	confirmMsg, preparedCalls, diag := simple.SimpleHarnessInstance.HandleUserToolConfirmMessage(session, question)
	if diag != nil {
		t.Fatalf("HandleUserToolConfirmMessage returned unexpected diagnostic: %s", diag.Message)
	}
	if confirmMsg.Role != core.RoleUser {
		t.Errorf("expected confirm message role %v, got %v", core.RoleUser, confirmMsg.Role)
	}

	started := time.Now()
	toolMessages, runDiag := simple.SimpleHarnessInstance.RunToolCalls(session, question, preparedCalls)
	elapsed := time.Since(started)
	if runDiag != nil {
		t.Fatalf("RunToolCalls returned unexpected diagnostic: %s", runDiag.Message)
	}

	expected := []struct {
		toolCallID string
		content    string
	}{
		{"call-approved-1", "approved one"},
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

	if got := atomic.LoadInt32(&maxRunning); got != 2 {
		t.Errorf("expected the 2 approved calls to overlap, only %d ran at the same time", got)
	}
	if elapsed >= 2*toolSleep {
		t.Errorf("expected the approved calls to run concurrently, it took %v", elapsed)
	}

	for _, toolCallID := range []string{"call-approved-1", "call-declined", "call-approved-2"} {
		if pending := session.GetPendingToolCall(toolCallID); pending != nil {
			t.Errorf("expected the pending call %s to be deleted", toolCallID)
		}
	}
}

func TestHandleUserToolConfirmMessage_SkipsUnknownToolCallID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := newTestParentSession(t, ctrl)
	session.SetQueryContext(context.Background(), nil)

	question := &batchConfirmQuestion{
		SimpleQuestion: *simple.NewSimpleQuestion(
			"", session.GetID(), "", nil, nil, false, false, core.QuestionTypeToolConfirm, 0,
		),
		confirms: []core.ToolConfirmAnswer{
			{ToolCallID: "call-unknown", Answer: "Yes"},
		},
	}

	_, preparedCalls, diag := simple.SimpleHarnessInstance.HandleUserToolConfirmMessage(session, question)
	if diag == nil {
		t.Fatal("expected a diagnostic for an unknown tool call ID, got nil")
	}
	if diag.Code != core.MessageCodeToolAlreadyConfirmed {
		t.Errorf("expected the already confirmed code, got %v", diag.Code)
	}
	if len(preparedCalls) != 0 {
		t.Errorf("expected no prepared tool calls, got %d", len(preparedCalls))
	}

	messages := core.Conversation{}
	userDiag := simple.SimpleHarnessInstance.HandleUserQuestion(session, question, &messages)
	if userDiag == nil {
		t.Fatal("expected a diagnostic from HandleUserQuestion for an unknown answer, got nil")
	}
	if userDiag.Code != core.MessageCodeToolAlreadyConfirmed {
		t.Errorf("expected the already confirmed code from HandleUserQuestion, got %v", userDiag.Code)
	}
}
