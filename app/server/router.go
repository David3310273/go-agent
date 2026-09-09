package server

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
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

// SetupRouter initializes and returns gin router with all routes
func SetupRouter(appConfig *core.AppConfig, agent core.AgentCore) *gin.Engine {
	router := gin.Default()

	// middleware to inject agent into gin context
	router.Use(func(c *gin.Context) {
		c.Set("agent", agent)
		c.Next()
	})

	// v1 api group
	v1 := router.Group("/v1")
	{
		// POST /v1/ask - ask a question
		v1.POST("/ask", func(c *gin.Context) {
			handleAsk(c, appConfig)
		})
	}

	return router
}

// handles POST /v1/ask requests
func handleAsk(c *gin.Context, appConfig *core.AppConfig) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// get agent from context
	agentValue, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "agent not available",
		})
		return
	}

	agent, ok := agentValue.(*simple.SimpleAgent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid agent type",
		})
		return
	}

	// create response channel for receiving answer
	// larger buffer for streaming chunks
	responseChan := make(chan core.Answer, 64)
	hintChan := make(chan core.Answer, appConfig.RequestQueueLength)

	// create question object
	question := &SimpleQuestion{
		query:          req.Question,
		responseChan:   responseChan,
		hintChan:       hintChan,
		streaming:      req.Stream,
		enableThinking: req.EnableThinking,
	}

	if req.SessionID != nil {
		question.sessionID = *req.SessionID
	}

	if req.Model != nil {
		question.model = *req.Model
	}

	// rating limit for request
	requestWaitingTimeout := time.Duration(appConfig.MaxWaitingRequest) * time.Second
	go func() {
		select {
		case agent.Question <- question:
		case <-time.After(requestWaitingTimeout):
			// non-blocking send, abandon if handler already exited
			select {
			case responseChan <- question.GetDefaultAnswer():
			default:
			}
		}
	}()

	log.Printf("will process question %s in mode: enableThinking=%v, stream=%v", question.query, question.enableThinking, question.streaming)

	responseWaitingTimeout := time.Duration(appConfig.MaxWaitingSeconds) * time.Second
	if req.Stream {
		//  SSE streaming response
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// use local variable so we can nil it after hintChan closes
		hintReader := hintChan
		c.Stream(func(w io.Writer) bool {
			select {
			case hint, ok := <-hintReader:
				if !ok {
					// hintChan closed, stop listening but keep stream alive
					hintReader = nil
					return true
				}
				switch resp := hint.(type) {
				case core.AgentResponse:
					if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && resp.Choices[0].Message.ReasoningContent != nil {
						// mock method here: print it to log, should send to client socket in the real app
						agent.GetLogger().Printf("\n\n[handleAsk][stream][thought] %s", *resp.Choices[0].Message.ReasoningContent)
						c.SSEvent("thought", *resp.Choices[0].Message.ReasoningContent)
					}
				}
				return true
			case chunk, ok := <-responseChan:
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
					// send sessionID and usage as done event
					log.Printf("[handleAsk][stream] finished, sessionID=%s, usage=%+v", resp.SessionID, resp.Response.Usage)
					c.SSEvent("done", gin.H{
						"sessionID": resp.SessionID,
						"usage":     resp.Response.Usage,
					})
					return false
				}
				return true
			case <-time.After(responseWaitingTimeout):
				log.Printf("[handleAsk][stream] timeout after %v", responseWaitingTimeout)
				c.SSEvent("error", "request timeout")
				return false
			}
		})
		log.Printf("[handleAsk] streaming ended for question: %s", req.Question)
	} else {
		// non-streaming: wait for single complete response
		for {
			select {
			case hint := <-hintChan:
				// log hints in non-streaming mode, then continue waiting for answer
				if resp, ok := hint.(core.AgentResponse); ok {
					if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
						if resp.Choices[0].Message.ReasoningContent != nil {
							agent.GetLogger().Printf("\n\n\n[Session][hint] thinking: %s", *resp.Choices[0].Message.ReasoningContent)
						}
					}
				}
			case answer := <-responseChan:
				if answer == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"code":    core.MessageCodeSystemError,
						"message": "agent is not ready, please try again later",
						"level":   core.SeverityError,
					})
					return
				}
				sessionID := ""
				response := AskResponse{
					SessionID: sessionID,
					Answer:    question.GetDefaultAnswer().ToString(),
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
			case <-time.After(responseWaitingTimeout):
				c.JSON(http.StatusRequestTimeout, gin.H{
					"code":    core.MessageCodeSystemError,
					"message": "request timeout",
					"level":   core.SeverityError,
				})
				return
			}
		}
	}
}

// SimpleQuestion implements core.Question interface for HTTP requests
type SimpleQuestion struct {
	query          string
	sessionID      string
	model          string
	responseChan   chan core.Answer
	hintChan       chan core.Answer
	streaming      bool
	enableThinking bool
}

func (q *SimpleQuestion) GetID() string                     { return q.sessionID }
func (q *SimpleQuestion) GetProviderName() string           { return q.model }
func (q *SimpleQuestion) GetStreaming() bool                { return q.streaming }
func (q *SimpleQuestion) GetEnableThinking() bool           { return q.enableThinking }
func (q *SimpleQuestion) GetQuery() string                  { return q.query }
func (q *SimpleQuestion) SetQuery(query string)             { q.query = query }
func (q *SimpleQuestion) GetRetryQuery() string             { return q.query }
func (q *SimpleQuestion) GetSessionID() string              { return q.sessionID }
func (q *SimpleQuestion) GetResponseChan() chan core.Answer { return q.responseChan }
func (q *SimpleQuestion) GetHintChan() chan core.Answer     { return q.hintChan }
func (q *SimpleQuestion) GetDefaultAnswer() core.Answer {
	return &SimpleAnswer{Content: "I don't know how to do next, please try again later.", SessionID: q.sessionID}
}
func (q *SimpleQuestion) ToString() string { return q.query }

type SimpleAnswer struct {
	Content   string `json:"content"`
	SessionID string `json:"sessionID"`
}

func (s *SimpleAnswer) ToString() string     { return s.Content }
func (s *SimpleAnswer) GetSessionID() string { return s.SessionID }
