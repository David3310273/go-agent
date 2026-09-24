package tools

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
)

func init() {
	// register SearchKnowledgebaseCall tool factory
	// updated to accept Session instead of Context.
	core.RegisterTool("SearchKnowledgeBase", func(rootPath string, session core.Session) core.Tool {
		return SearchKnowledgebaseCall{
			Name:     "SearchKnowledgeBase",
			Schema:   "searchkb.schema.json",
			RootPath: rootPath,
			Session:  session,
		}
	})
}

// SearchKnowledgebaseCall implements core.Tool interface for searching knowledge base.
// searches knowledge base by keyword and returns results as JSON.
type SearchKnowledgebaseCall struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	// project root path for resolving schema file path
	RootPath string
	// session for accessing context at runtime.
	Session core.Session
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f SearchKnowledgebaseCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f SearchKnowledgebaseCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	// use RootPath instead of hardcoded relative path
	content, err := os.ReadFile(path.Join(f.RootPath, SchemaPath, f.Schema))
	if err != nil {
		log.Printf("GetSchema: failed to read schema file: %v", err)
		return schema
	}

	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}

	return schema
}

// GetDescription returns the tool description from schema
func (f SearchKnowledgebaseCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

// GetContext returns the agent session runtime context.
// implements core.Tool interface.
func (f SearchKnowledgebaseCall) GetContext() core.Context {
	return f.Session.GetContext()
}

// Validate validates the tool configuration
// validates keyword is provided.
func (f SearchKnowledgebaseCall) Validate(args map[string]any) *core.Diagnostic {
	keyword, _ := args["keyword"].(string)
	if keyword == "" {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeToolRunError,
			Message: "keyword is required",
		}
	}

	return nil
}

func (f SearchKnowledgebaseCall) IsDestructive() bool {
	return false
}

// SearchResult represents a single search result from knowledge base.
// used for JSON serialization of search results.
type SearchResult struct {
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}

// SearchResponse represents the response from knowledge base search.
// used for JSON serialization of search response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}

// GetRunner returns the runner function for SearchKnowledgebaseCall.
// searches knowledge base and returns results as JSON.
func (f SearchKnowledgebaseCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		keyword, _ := args["keyword"].(string)

		// parse topK, default to 3
		topK := 3
		if topKVal, ok := args["topK"].(float64); ok {
			topK = int(topKVal)
		}

		// get knowledge bases from context
		knowledgeBases := f.Session.GetContext().GetKnowledgeBase()
		if len(knowledgeBases) == 0 {
			return "Failed", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "no knowledge base available",
			}
		}

		var allResults []SearchResult

		// search in each knowledge base
		for _, kbAny := range knowledgeBases {
			milvusKB, ok := kbAny.(*storage.MilvusKnowledgebase[any])
			if !ok {
				continue
			}

			searchResults, diag := milvusKB.Search(keyword, topK, "")
			if diag != nil {
				log.Printf("SearchKnowledgeBase: search error: %v", diag.Message)
				continue
			}

			for _, res := range searchResults {
				allResults = append(allResults, SearchResult{
					Content:    res.GetContent(),
					Confidence: res.GetConfidence(),
				})
			}
		}

		// build response
		response := SearchResponse{
			Results: allResults,
			Count:   len(allResults),
		}

		// serialize to JSON
		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
