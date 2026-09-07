package server

import (
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
	responseChan := make(chan core.Answer, 1)
	hintChan := make(chan core.Answer, appConfig.RequestQueueLength)

	// create question object
	question := &SimpleQuestion{
		query:        req.Question,
		responseChan: responseChan,
		hintChan:     hintChan,
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
			responseChan <- question.GetDefaultAnswer()
		}
	}()

	responseWaitingTimeout := time.Duration(appConfig.MaxWaitingSeconds) * time.Second
	// wait for response with timeout using select
	select {
	// TODO: receive hint from hint chan if streaming mode supported
	case answer := <-responseChan:
		if answer == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    core.MessageCodeSystemError,
				"message": "agent is not ready, please try again later",
				"level":   core.SeverityError,
			})
			return
		}
		// extract sessionID and AgentResponse via type assertion
		sessionID := ""
		response := AskResponse{
			SessionID: sessionID,
			Answer:    question.GetDefaultAnswer().ToString(),
		}
		if sessionResponse, ok := answer.(simple.SimpleSessionResponse); ok {
			agentResponse := sessionResponse.Response
			// use choices[0].message.content as the clean text answer
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
	case <-time.After(responseWaitingTimeout):
		c.JSON(http.StatusRequestTimeout, gin.H{
			"code":    core.MessageCodeSystemError,
			"message": "request timeout",
			"level":   core.SeverityError,
		})
	}
}

// SimpleQuestion implements core.Question interface for HTTP requests
type SimpleQuestion struct {
	query        string
	sessionID    string
	model        string
	responseChan chan core.Answer
	hintChan     chan core.Answer
}

func (q *SimpleQuestion) GetID() string                     { return q.sessionID }
func (q *SimpleQuestion) GetProviderName() string           { return q.model }
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
