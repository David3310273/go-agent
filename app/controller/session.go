package controller

import (
	"fmt"
	"net/http"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

type SessionInfo struct {
	SessionID string             `json:"sessionID"`
	Status    core.SessionStatus `json:"status"`
}

type SessionListResponse struct {
	SessionsInfo []services.SessionHistoryList `json:"sessionsInfo"`
}

type SessionDeleteResponse struct {
	SessionID string `json:"sessionID"`
	Message   string `json:"message"`
}

func extractAgent(c *gin.Context) (*simple.SimpleAgent, bool) {
	agentValue, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent not available"})
		return nil, false
	}
	agent, ok := agentValue.(*simple.SimpleAgent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid agent type"})
		return nil, false
	}
	return agent, true
}

func HandleDeleteSession(c *gin.Context) {
	agent, ok := extractAgent(c)
	if !ok {
		return
	}

	sessionID := c.Query("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "sessionID is required",
		})
		return
	}

	diag := agent.RemoveSession(sessionID)
	if diag != nil {
		c.JSON(http.StatusInternalServerError, diag)
		return
	}

	c.JSON(http.StatusOK, SessionDeleteResponse{
		SessionID: sessionID,
		Message:   fmt.Sprintf("session %s deleted successfully", sessionID),
	})
}

func HandleGetSession(c *gin.Context) {
	agent, ok := extractAgent(c)
	if !ok {
		return
	}

	sessionID := c.Query("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "sessionID is required",
		})
		return
	}

	c.JSON(http.StatusOK, services.GetSessionHistory(agent, sessionID))
}

func HandleListSessions(c *gin.Context) {
	agent, ok := extractAgent(c)
	if !ok {
		return
	}

	result := services.GetSessionList(agent)

	c.JSON(http.StatusOK, SessionListResponse{
		SessionsInfo: result,
	})
}
