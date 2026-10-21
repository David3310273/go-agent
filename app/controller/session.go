package controller

import (
	"fmt"
	"net/http"
	"strconv"

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

func extractApp(c *gin.Context) (core.AgentApp, bool) {
	appVal, exists := c.Get("app")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "app not available"})
		return nil, false
	}
	app, ok := appVal.(core.AgentApp)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid app type"})
		return nil, false
	}
	return app, true
}

func HandleDeleteSession(c *gin.Context) {
	app, ok := extractApp(c)
	if !ok {
		return
	}
	agent := app.GetAgentCore().(*simple.SimpleAgent)

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

	c.JSON(http.StatusOK, core.Diagnostic{
		Code: core.MessageCodeSuccess,
		Data: SessionDeleteResponse{
			SessionID: sessionID,
			Message:   fmt.Sprintf("session %s deleted successfully", sessionID),
		},
	})
}

func HandleGetSession(c *gin.Context) {
	app, ok := extractApp(c)
	if !ok {
		return
	}
	agent := app.GetAgentCore().(*simple.SimpleAgent)

	sessionID := c.Query("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "sessionID is required",
		})
		return
	}

	limit := 50

	limitStr := c.Query("limit")
	if limitStr != "" {
		if limitFromQuery, err := strconv.Atoi(limitStr); err == nil {
			limit = limitFromQuery
		} else {
			c.JSON(http.StatusBadRequest, core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: "sessionID is required",
			})
			return
		}
	}

	chats, diag := services.GetSessionHistory(agent, sessionID, limit)
	if diag != nil {
		c.JSON(http.StatusInternalServerError, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Message: diag.Message,
		})
		return
	}

	c.JSON(http.StatusOK, core.Diagnostic{
		Code: core.MessageCodeSuccess,
		Data: string(chats),
	})
}

func HandleListSessions(c *gin.Context) {
	app, ok := extractApp(c)
	if !ok {
		return
	}
	agent := app.GetAgentCore().(*simple.SimpleAgent)

	result := services.GetSessionList(agent)

	c.JSON(http.StatusOK, core.Diagnostic{
		Code: core.MessageCodeSuccess,
		Data: SessionListResponse{
			SessionsInfo: result,
		},
	})
}
