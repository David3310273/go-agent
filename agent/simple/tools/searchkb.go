package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/David3310273/go-agent/core"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
)

func init() {
	// register SearchKnowledgebaseCall tool factory
	// updated to accept Context instead of Session.
	core.RegisterTool("SearchKnowledgeBase", func(ctx core.Context) core.Tool {
		config := LoadToolConfig(ctx.GetRootPath(), SearchKnowledgebaseSchemaPath, "searchkb.config.json")
		return SearchKnowledgebaseCall{
			Name:     config.Name,
			Schema:   config.Schema,
			RootPath: ctx.GetRootPath(),
			Context:  ctx,
		}
	})

	// register FetchAllDocInfo tool factory
	core.RegisterTool("FetchAllDocInfo", func(ctx core.Context) core.Tool {
		config := LoadToolConfig(ctx.GetRootPath(), SearchKnowledgebaseSchemaPath, "fetchalldoc.config.json")
		return FetchAllDocInfoCall{
			Name:     config.Name,
			Schema:   config.Schema,
			RootPath: ctx.GetRootPath(),
			Context:  ctx,
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
	// context for accessing runtime resources.
	Context core.Context
}

// GetName returns the function name from schema for matching with LLM tool calls
func (f SearchKnowledgebaseCall) GetName() string {
	return f.GetSchema().Function.Name
}

// implement core.Tool interface, returns provider-agnostic ToolSchema
func (f SearchKnowledgebaseCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema

	// use RootPath instead of hardcoded relative path
	content, err := os.ReadFile(path.Join(f.RootPath, SearchKnowledgebaseSchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "searchkb: failed to read schema: %v", err)
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
	return f.Context
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

func (f SearchKnowledgebaseCall) ToShellScript(args map[string]any) string {
	return ""
}

func (f SearchKnowledgebaseCall) GetSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
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
		knowledgeBases := f.Context.GetKnowledgeBase()
		if len(*knowledgeBases) == 0 {
			return "Failed", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "no knowledge base available",
			}
		}

		var allResults []SearchResult

		// search in each knowledge base
		for _, kbAny := range *knowledgeBases {
			milvusKB, ok := kbAny.(*storage.MilvusKnowledgebase[any])
			if !ok {
				continue
			}

			filters := []string{}
			if filename, ok := args["filename"].(string); ok && filename != "" {
				filters = append(filters, fmt.Sprintf("filename == '%s'", filename))
			}
			if domain, ok := args["domain"].(string); ok && domain != "" {
				filters = append(filters, fmt.Sprintf("domain == '%s'", domain))
			}

			filter := strings.Join(filters, " && ")

			searchResults, diag := milvusKB.Search(keyword, topK, filter)
			if diag != nil {
				core.LogStd(core.LogLevelError, "searchkb: search error: %v", diag.Message)
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

// =============================================================================
// FetchAllDocInfo
// =============================================================================

// interface assertion
var _ core.Tool = (*FetchAllDocInfoCall)(nil)

// FetchAllDocInfoCall implements core.Tool interface for fetching all document information from a knowledge base.
type FetchAllDocInfoCall struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	RootPath string
	Context  core.Context
}

func (f FetchAllDocInfoCall) GetName() string {
	return f.GetSchema().Function.Name
}

func (f FetchAllDocInfoCall) IsDestructive() bool {
	return false
}

func (f FetchAllDocInfoCall) ToShellScript(args map[string]any) string {
	return ""
}

func (f FetchAllDocInfoCall) GetSandboxEnv() core.SandboxSpec {
	return core.SandboxSpec{}
}

func (f FetchAllDocInfoCall) GetSchema() core.ToolSchema {
	var schema core.ToolSchema
	content, err := os.ReadFile(path.Join(f.RootPath, SearchKnowledgebaseSchemaPath, f.Schema))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "searchkb: failed to read schema: %v", err)
		return schema
	}
	if err := json.Unmarshal(content, &schema); err != nil {
		return schema
	}
	return schema
}

func (f FetchAllDocInfoCall) GetDescription() string {
	return f.GetSchema().Function.Description
}

func (f FetchAllDocInfoCall) GetContext() core.Context {
	return f.Context
}

func (f FetchAllDocInfoCall) Validate(args map[string]any) *core.Diagnostic {
	// kbName is optional, empty means fetch all knowledge bases
	return nil
}

// DocInfo represents document information in knowledge base
type DocInfo struct {
	Domain      string `json:"domain"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
}

// KBDocInfoResponse represents the response from fetching all doc info
type KBDocInfoResponse struct {
	KBName        string    `json:"kbName"`
	KBDescription string    `json:"kbDescription"`
	Documents     []DocInfo `json:"documents"`
}

func (f FetchAllDocInfoCall) GetRunner() func(args map[string]any) (string, *core.Diagnostic) {
	return func(args map[string]any) (string, *core.Diagnostic) {
		kbName, _ := args["kbName"].(string)

		// get knowledge bases from context
		knowledgeBases := f.Context.GetKnowledgeBase()
		if len(*knowledgeBases) == 0 {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "no knowledge base available",
			}
		}

		allKBs := make([]KBDocInfoResponse, 0, len(*knowledgeBases))
		totalDocs := 0

		for _, kb := range *knowledgeBases {
			if kbName != "" && kbName != kb.GetName() {
				continue
			}

			docDesc := kb.GetDocDescription("", "")
			documents := make([]DocInfo, 0)

			if len(docDesc) > 0 {
				for domain, desc := range docDesc {
					if descMap, ok := desc.(map[string]any); ok && descMap != nil {
						for filename, description := range descMap {
							descStr := ""
							if desc, ok := description.(string); ok {
								descStr = desc
							}
							documents = append(documents, DocInfo{
								Domain:      domain,
								Filename:    filename,
								Description: descStr,
							})
						}
					}
				}
			}

			allKBs = append(allKBs, KBDocInfoResponse{
				KBName:        kb.GetName(),
				KBDescription: kb.GetDescription(),
				Documents:     documents,
			})
			totalDocs += len(documents)
		}

		jsonResult, err := json.Marshal(allKBs)
		if err != nil {
			return "", &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeToolRunError,
				Message: "failed to serialize results",
			}
		}

		core.LogStd(core.LogLevelDebug, "fetchalldocinfo: fetched %d knowledge bases with %d total documents", len(allKBs), totalDocs)

		return string(jsonResult), &core.Diagnostic{
			Level:   core.SeverityInfo,
			Code:    core.MessageCodeSuccess,
			Message: "Success",
		}
	}
}
