package controller

import (
	"io"
	"net/http"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

// ResponseType constants for different interaction types
const (
	ResponseTypeNormal      = "normal"       // normal response with answer
	ResponseTypeToolConfirm = "tool_confirm" // destructive tool needs user confirmation
	// future types can be added here, e.g.:
	// ResponseTypePermission = "permission"    // permission request
	// ResponseTypeInput      = "input"         // request for user input
)

// ToolConfirmAnswerRequest is one answer of a tool_confirm batch.
// the frontend answers every pending destructive tool call in a single request, each
// item carries the toolCallID it got in the tool_confirm response.
type ToolConfirmAnswerRequest struct {
	ToolCallID string `json:"toolCallID"`           // identifies the pending call
	ServerName string `json:"serverName,omitempty"` // MCP server name, echoed back
	ToolName   string `json:"toolName,omitempty"`   // outer tool name, echoed back
	Answer     string `json:"answer"`               // Yes or No
}

// AskRequest represents the request body for /v1/ask endpoint
type AskRequest struct {
	SessionID *string `json:"sessionID,omitempty"` // optional
	// no longer binding required, a tool_confirm request carries confirms instead of a
	// question. HandleAsk validates the field that its type needs.
	Question string            `json:"question,omitempty"`
	Type     core.QuestionType `json:"type" binding:"required"`
	Model    *string           `json:"model,omitempty"`
	// option from request
	Stream         bool `json:"stream,omitempty"`
	EnableThinking bool `json:"enableThinking,omitempty"`
	// option for tool_confirm type
	// a batch of answers replaces the single toolName/serverName pair
	Confirms []ToolConfirmAnswerRequest `json:"confirms,omitempty"`
}

// AskResponse represents the response body for /v1/ask endpoint
type AskResponse struct {
	Type      string     `json:"type"` // response type: normal, tool_confirm, etc.
	SessionID string     `json:"sessionID"`
	Answer    string     `json:"answer"`
	Thought   string     `json:"thought"`
	Usage     core.Usage `json:"usage"`
	// only for tool_confirm type, the whole batch waiting for the user answers
	Confirms []simple.ToolConfirmItem `json:"confirms,omitempty"`
}

// CancelRequest represents the request body for /v1/agent/cancel endpoint
type CancelRequest struct {
	SessionID string `json:"sessionID" binding:"required"`
}

// CancelResponse represents the response body for /v1/agent/cancel endpoint
type CancelResponse struct {
	SessionID string `json:"sessionID"`
	Message   string `json:"message"`
}

// HandleCancel handles POST /v1/agent/cancel requests
func HandleCancel(c *gin.Context, agent *simple.SimpleAgent) {
	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// get session
	session, diag := agent.GetSessionOnCreate(req.SessionID, false)
	if diag != nil || session == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    core.MessageCodeSessionStopError,
			"message": "session not found: " + req.SessionID,
			"level":   core.SeverityError,
		})
		return
	}

	// async cancel current query processing (triggers queryCtx.Done())
	session.CancelQuery()

	c.JSON(http.StatusOK, CancelResponse{
		SessionID: req.SessionID,
		Message:   "Session query cancelled successfully",
	})
}

// HandleAsk handles POST /v1/ask requests
func HandleAsk(c *gin.Context, agent *simple.SimpleAgent, appConfig *core.AppConfig) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// validate the field the request type actually carries. A tool_confirm request
	// answers a batch of pending destructive calls, every item needs its toolCallID and Yes or No.
	if req.Type == core.QuestionTypeToolConfirm {
		if len(req.Confirms) == 0 {
			c.JSON(http.StatusBadRequest, core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: "tool_confirm requires at least one item in confirms",
			})
			return
		}
		for _, confirm := range req.Confirms {
			if confirm.ToolCallID == "" || (confirm.Answer != "Yes" && confirm.Answer != "No") {
				c.JSON(http.StatusBadRequest, core.Diagnostic{
					Code:    core.MessageCodeSystemError,
					Level:   core.SeverityError,
					Message: "each confirms item requires a toolCallID and answer 'Yes' or 'No'",
					Data:    confirm.ToolCallID + ":" + confirm.Answer,
				})
				return
			}
		}
	} else if req.Question == "" {
		// the binding tag is gone, so the normal type is checked here instead
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "question is required",
		})
		return
	}

	// build params
	params := &services.AskParams{
		Question:       req.Question,
		Type:           req.Type,
		Stream:         req.Stream,
		EnableThinking: req.EnableThinking,
	}

	if req.SessionID != nil {
		params.SessionID = *req.SessionID
	} else {
		id, _ := uuid.NewV4()
		params.SessionID = id.String()
	}

	if req.Model != nil {
		params.Model = *req.Model
	}

	var finalParams any = params
	if params.Type == core.QuestionTypeToolConfirm {
		// map the batch of answers into the confirm question
		confirms := make([]core.ToolConfirmAnswer, 0, len(req.Confirms))
		for _, confirm := range req.Confirms {
			confirms = append(confirms, core.ToolConfirmAnswer{
				ToolCallID: confirm.ToolCallID,
				ServerName: confirm.ServerName,
				ToolName:   confirm.ToolName,
				Answer:     confirm.Answer,
			})
		}
		finalParams = &services.ToolConfirmAskParams{
			AskParams: *params,
			Confirms:  confirms,
		}
	}

	// call service
	result, diag := services.Ask(agent, appConfig, finalParams)
	if diag != nil {
		c.JSON(http.StatusBadRequest, diag)
		return
	}

	responseWaitingTimeout := time.Duration(appConfig.MaxWaitingSeconds) * time.Second
	if req.Stream {
		handleStreamResponse(c, agent, result, responseWaitingTimeout)
	} else {
		handleNonStreamResponse(c, result, responseWaitingTimeout)
	}
}

// handleStreamResponse handles SSE streaming response
func handleStreamResponse(c *gin.Context, agent *simple.SimpleAgent, result *services.AskResult, timeout time.Duration) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	hintReader := result.HintChan
	c.Stream(func(w io.Writer) bool {
		select {
		case hint, ok := <-hintReader:
			if !ok {
				hintReader = nil
				return true
			}
			switch resp := hint.(type) {
			case core.AgentResponse:
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && resp.Choices[0].Message.ReasoningContent != nil {
					core.LogDebug("", "\n\n[handleAsk][stream][thought] %s", *resp.Choices[0].Message.ReasoningContent)
					c.SSEvent("thought", *resp.Choices[0].Message.ReasoningContent)
				}
			}
			return true
		case chunk, ok := <-result.ResponseChan:
			if !ok {
				core.LogStd(core.LogLevelDebug, "response channel closed, ending stream")
				return false
			}
			switch resp := chunk.(type) {
			// event stream chunk, original response
			case core.AgentResponse:
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && resp.Choices[0].Message.Content != "" {
					core.LogStd(core.LogLevelDebug, "stream chunk: %s", resp.Choices[0].Message.Content)
					c.SSEvent("message", resp.Choices[0].Message.Content)
				}
			case simple.SimpleNormalResponse:
				core.LogStd(core.LogLevelInfo, "stream finished: session_id=%s, prompt_tokens=%d, completion_tokens=%d, total_tokens=%d",
					resp.SessionID, resp.Response.Usage.PromptTokens, resp.Response.Usage.CompletionTokens, resp.Response.Usage.TotalTokens)
				// send response message if available (e.g., tool already confirmed)
				if resp.Response.Response != "" {
					c.SSEvent("message", resp.Response.Response)
				}
				c.SSEvent("done", gin.H{
					"type":      ResponseTypeNormal,
					"sessionID": resp.SessionID,
					"usage":     resp.Response.Usage,
				})
				return false
			case simple.SimpleToolConfirmResponse:
				// send tool confirmation event for the destructive tool batch
				// the payload carries the whole batch, the frontend answers every item
				core.LogStd(core.LogLevelInfo, "tool confirm needed: count=%d, session_id=%s", len(resp.Confirms), resp.SessionID)
				c.SSEvent("tool_confirm", gin.H{
					"type":      ResponseTypeToolConfirm,
					"confirms":  resp.Confirms,
					"message":   resp.Response.Response,
					"sessionID": resp.SessionID,
					"usage":     resp.Response.Usage,
				})
				return false
			}
			return true
		case <-time.After(timeout):
			core.LogStd(core.LogLevelWarn, "stream timeout after %v", timeout)
			c.SSEvent("error", "request timeout")
			return false
		}
	})
	core.LogStd(core.LogLevelDebug, "streaming ended for question: %s", result.Question.GetQuery())
}

// handleNonStreamResponse handles non-streaming JSON response
func handleNonStreamResponse(c *gin.Context, result *services.AskResult, timeout time.Duration) {
	for {
		select {
		case hint := <-result.HintChan:
			if resp, ok := hint.(core.AgentResponse); ok {
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
					if resp.Choices[0].Message.ReasoningContent != nil {
						core.LogStd(core.LogLevelDebug, "hint thinking: %s", *resp.Choices[0].Message.ReasoningContent)
					}
				}
			}
		case answer := <-result.ResponseChan:
			if answer == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"code":    core.MessageCodeSystemError,
					"message": "agent is not ready, please try again later",
					"level":   core.SeverityError,
				})
				return
			}
			// handle different response types
			switch resp := answer.(type) {
			case simple.SimpleNormalResponse:
				agentResponse := resp.Response
				response := AskResponse{
					Type:      ResponseTypeNormal,
					SessionID: resp.SessionID,
					Answer:    agentResponse.Response,
					Thought:   agentResponse.Thought,
					Usage:     agentResponse.Usage,
				}
				if len(agentResponse.Choices) > 0 && agentResponse.Choices[0].Message != nil {
					response.Answer = agentResponse.Choices[0].Message.Content
				}
				c.JSON(http.StatusOK, response)
			case simple.SimpleToolConfirmResponse:
				// return confirmation response for the destructive tool batch
				// confirms carries every pending call, the frontend answers them in one
				// request with a Yes or No per toolCallID
				response := AskResponse{
					Type:      ResponseTypeToolConfirm,
					SessionID: resp.SessionID,
					Answer:    resp.Response.Response,
					Usage:     resp.Response.Usage,
					Confirms:  resp.Confirms,
				}
				c.JSON(http.StatusOK, response)
			default:
				// fallback for other response types
				c.JSON(http.StatusOK, AskResponse{
					Type:   ResponseTypeNormal,
					Answer: answer.ToString(),
				})
			}
			return
		case <-time.After(timeout):
			c.JSON(http.StatusRequestTimeout, gin.H{
				"code":    core.MessageCodeSystemError,
				"message": "request timeout",
				"level":   core.SeverityError,
			})
			return
		}
	}
}
