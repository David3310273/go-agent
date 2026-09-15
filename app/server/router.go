package server

import (
	"fmt"
	"net/http"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/app/controller"
	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/storage/milvus"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and returns gin router with all routes
func SetupRouter(appConfig *core.AppConfig, agent core.AgentCore) *gin.Engine {
	router := gin.Default()

	// middleware to inject agent into gin context
	router.Use(func(c *gin.Context) {
		c.Set("agent", agent)
		c.Next()
	})

	// v1 agent api group
	v1Agent := router.Group("/v1/agent")
	{
		// POST /v1/agent/ask - ask a question
		v1Agent.POST("/ask", func(c *gin.Context) {
			agentValue, exists := c.Get("agent")
			if !exists {
				c.JSON(500, gin.H{"error": "agent not available"})
				return
			}
			agent, ok := agentValue.(*simple.SimpleAgent)
			if !ok {
				c.JSON(500, gin.H{"error": "invalid agent type"})
				return
			}
			controller.HandleAsk(c, agent, appConfig)
		})
	}

	// internal knowledgebase api group (not exposed to external users when it's toC app)
	internalKnowledge := router.Group("/v1/internal/knowledgebase")
	{
		// middleware to check storageType and initialize milvus knowledge service
		// validates storageType before entering handlers.
		internalKnowledge.Use(func(c *gin.Context) {
			// get storageType from form-data (POST/PUT) or query (GET/DELETE)
			var storageType string
			if c.Request.Method == "POST" || c.Request.Method == "PUT" {
				storageType = c.PostForm("storageType")
			} else {
				storageType = c.Query("storageType")
			}

			// check storageType is provided and supported
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

			// initialize milvus with rootPath from agent
			rootPath := agent.GetRootPath()
			milvus.Init(rootPath)

			// create knowledge base config with default values
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

			// inject MilvusKnowledgeService with knowledge base instance
			c.Set("knowledgeService", services.NewMilvusKnowledgeService(kbConfig))
			c.Next()
		})

		// POST /v1/internal/knowledgebase - create
		internalKnowledge.POST("", controller.HandleCreateKnowledge)
		// GET /v1/internal/knowledgebase - search by keyword, filename, contentType
		internalKnowledge.GET("", controller.HandleGetKnowledge)
		// PUT /v1/internal/knowledgebase - update by filename, contentType (delete + create)
		internalKnowledge.PUT("", controller.HandleUpdateKnowledge)
		// DELETE /v1/internal/knowledgebase - delete by filename, contentType
		internalKnowledge.DELETE("", controller.HandleDeleteKnowledge)
	}

	return router
}
