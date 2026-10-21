package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/David3310273/go-agent/app/controller"
	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/storage/milvus"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and returns gin router with all routes
func SetupRouter(app core.AgentApp) *gin.Engine {
	router := gin.Default()

	// serve static frontend assets (React/Vite build output)
	assets := router.Group("/assets")
	assets.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Next()
	})
	assets.Static("", "./app/frontend/web/dist/assets")
	router.StaticFile("/favicon.svg", "./app/frontend/web/dist/favicon.svg")

	// SPA fallback
	router.NoRoute(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("./app/frontend/web/dist/index.html")
	})

	// allow CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// JWT auth middleware: parse X-User-Identity header, get plan, apply plan
	router.Use(authMiddleware(app))

	// middleware to inject app into gin context
	router.Use(func(c *gin.Context) {
		c.Set("app", app)
		c.Next()
	})

	v1Session := router.Group("/v1/session")
	{
		v1Session.DELETE("", controller.HandleDeleteSession)
		v1Session.GET("", controller.HandleGetSession)
		v1Session.GET("/list", controller.HandleListSessions)
	}

	// v1 agent api group
	v1Agent := router.Group("/v1/agent")
	{
		v1Agent.POST("/ask", func(c *gin.Context) {
			controller.HandleAsk(c)
		})

		v1Agent.POST("/cancel", func(c *gin.Context) {
			controller.HandleCancel(c)
		})
	}

	// internal knowledgebase api group
	internalKnowledge := router.Group("/v1/internal/knowledgebase")
	{
		internalKnowledge.Use(func(c *gin.Context) {
			var storageType string
			if c.Request.Method == "POST" || c.Request.Method == "PUT" {
				storageType = c.PostForm("storageType")
			} else {
				storageType = c.Query("storageType")
			}

			if storageType == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, core.Diagnostic{
					Code:    core.MessageCodeSystemError,
					Level:   core.SeverityError,
					Message: "storageType is required",
				})
				return
			}

			if services.StorageType(storageType) != services.StorageTypeMilvus {
				c.AbortWithStatusJSON(http.StatusBadRequest, core.Diagnostic{
					Code:    core.MessageCodeSystemError,
					Level:   core.SeverityError,
					Message: fmt.Sprintf("storage type %s not implemented", storageType),
					Data:    storageType,
				})
				return
			}

			rootPath := app.GetAgentCore().GetRootPath()
			milvus.Init(rootPath)

			kbConfig := core.KnowledgeBaseConfig{
				RootPath: rootPath,
				Domain:   "technology",
				StorageOptions: map[string]string{
					"type":          "milvus",
					"dbName":        "",
					"collection":    "markdown_documents",
					"partitionName": "",
				},
			}

			c.Set("knowledgeService", services.NewMilvusKnowledgeService(kbConfig))
			c.Next()
		})

		internalKnowledge.POST("", controller.HandleCreateKnowledge)
		internalKnowledge.GET("", controller.HandleGetKnowledge)
		internalKnowledge.PUT("", controller.HandleUpdateKnowledge)
		internalKnowledge.DELETE("", controller.HandleDeleteKnowledge)
	}

	return router
}

// authMiddleware parses JWT from Authorization header,
// authenticates the user, retrieves and applies the user plan.
func authMiddleware(app core.AgentApp) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// extract Bearer token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: "invalid Authorization header format, expected Bearer token",
			})
			return
		}

		// parse JWT payload to extract user ID
		userID, err := parseJWTUserID(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, core.Diagnostic{
				Code:    core.MessageCodeSystemError,
				Level:   core.SeverityError,
				Message: fmt.Sprintf("invalid token: %v", err),
			})
			return
		}

		// auth user
		if diag := app.AuthUser(token); diag != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, diag)
			return
		}

		// get user plan
		plan, diag := app.GetUserPlan(userID)
		if diag != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, diag)
			return
		}

		// inject plan overrides into request body so downstream handlers
		// receive them transparently without the client sending them.
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			injectPlanIntoBody(c, plan)
		}

		// store user info in context for downstream handlers
		c.Set("userID", userID)
		c.Next()
	}
}

// parseJWTUserID extracts the user ID from a JWT token payload.
// It decodes the base64 payload section without verifying the signature.
func parseJWTUserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("missing or invalid 'sub' claim in JWT")
	}

	return userID, nil
}

// injectPlanIntoBody reads the request body, injects plan-level overrides,
// and writes the modified body back so downstream ShouldBindJSON picks them up.
func injectPlanIntoBody(c *gin.Context, plan core.Plan) {
	if c.Request.Body == nil {
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	if err != nil {
		return
	}

	var bodyMap map[string]any
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		// body is not valid JSON or empty, restore original
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return
	}

	bodyMap["planReActMaxRounds"] = plan.MaxRoundsReAct
	bodyMap["planMaxToken"] = plan.MaxToken
	bodyMap["planAvailableModels"] = plan.AvailableModels

	newBody, err := json.Marshal(bodyMap)
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(newBody))
	c.Request.ContentLength = int64(len(newBody))
}
