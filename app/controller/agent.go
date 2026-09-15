package controller

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

// AskRequest represents the request body for /v1/ask endpoint
type AskRequest struct {
	SessionID *string `json:"sessionID,omitempty"` // optional
	Question  string  `json:"question" binding:"required"`
	Model     *string `json:"model,omitempty"`
	// option from request
	Stream         bool `json:"stream,omitempty"`
	EnableThinking bool `json:"enableThinking,omitempty"`
}

// AskResponse represents the response body for /v1/ask endpoint
type AskResponse struct {
	SessionID string     `json:"sessionID"`
	Answer    string     `json:"answer"`
	Thought   string     `json:"thought"`
	Usage     core.Usage `json:"usage"`
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

	// build params
	params := &services.AskParams{
		Question:       req.Question,
		Stream:         req.Stream,
		EnableThinking: req.EnableThinking,
	}
	if req.SessionID != nil {
		params.SessionID = *req.SessionID
	}
	if req.Model != nil {
		params.Model = *req.Model
	}

	// call service
	result := services.Ask(agent, appConfig, params)

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
					agent.GetLogger().Printf("\n\n[handleAsk][stream][thought] %s", *resp.Choices[0].Message.ReasoningContent)
					c.SSEvent("thought", *resp.Choices[0].Message.ReasoningContent)
				}
			}
			return true
		case chunk, ok := <-result.ResponseChan:
			if !ok {
				log.Printf("[handleAsk][stream] responseChan closed, ending stream")
				return false
			}
			switch resp := chunk.(type) {
			case core.AgentResponse:
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && resp.Choices[0].Message.Content != "" {
					log.Printf("[handleAsk][stream][chunk] %s", resp.Choices[0].Message.Content)
					c.SSEvent("message", resp.Choices[0].Message.Content)
				}
			case simple.SimpleSessionResponse:
				log.Printf("[handleAsk][stream] finished, sessionID=%s, usage=%+v", resp.SessionID, resp.Response.Usage)
				c.SSEvent("done", gin.H{
					"sessionID": resp.SessionID,
					"usage":     resp.Response.Usage,
				})
				return false
			}
			return true
		case <-time.After(timeout):
			log.Printf("[handleAsk][stream] timeout after %v", timeout)
			c.SSEvent("error", "request timeout")
			return false
		}
	})
	log.Printf("[handleAsk] streaming ended for question: %s", result.Question.GetQuery())
}

// handleNonStreamResponse handles non-streaming JSON response
func handleNonStreamResponse(c *gin.Context, result *services.AskResult, timeout time.Duration) {
	for {
		select {
		case hint := <-result.HintChan:
			if resp, ok := hint.(core.AgentResponse); ok {
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
					if resp.Choices[0].Message.ReasoningContent != nil {
						log.Printf("\n\n\n[Session][hint] thinking: %s", *resp.Choices[0].Message.ReasoningContent)
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
			response := AskResponse{
				Answer: result.Question.GetDefaultAnswer().ToString(),
			}
			if sessionResponse, ok := answer.(simple.SimpleSessionResponse); ok {
				agentResponse := sessionResponse.Response
				if len(agentResponse.Choices) > 0 && agentResponse.Choices[0].Message != nil {
					response.Answer = agentResponse.Choices[0].Message.Content
				} else {
					response.Answer = agentResponse.Response
				}
				response.SessionID = sessionResponse.SessionID
				response.Thought = agentResponse.Thought
				response.Usage = agentResponse.Usage
			}
			c.JSON(http.StatusOK, response)
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
