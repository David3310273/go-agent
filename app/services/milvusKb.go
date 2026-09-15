package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/app/models"
	"github.com/David3310273/go-agent/core"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
	"github.com/David3310273/go-agent/storage/milvus"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageTypeMilvus StorageType = "milvus"
)

// MilvusKnowledgeService handles milvus knowledge base operations
// uses KnowledgeBase[any] for document processing and storage.
type MilvusKnowledgeService struct {
	kb *storage.MilvusKnowledgebase[any]
}

// NewMilvusKnowledgeService creates a new MilvusKnowledgeService instance
// creates KnowledgeBase using NewSimpleKnowledgeBase factory.
func NewMilvusKnowledgeService(config core.KnowledgeBaseConfig) *MilvusKnowledgeService {
	kb := simple.NewSimpleKnowledgeBase(config)
	return &MilvusKnowledgeService{kb: kb}
}

// MilvusKnowledgeCreateRequest represents the request for creating a knowledge base entry
// accepts binary file data and filename for document processing.
type MilvusKnowledgeCreateRequest struct {
	StorageType StorageType `json:"storageType"`
	ContentType string      `json:"contentType"`
	Domain      string      `json:"domain"`
	Data        []byte      `json:"-"`
	Filename    string      `json:"-"`
}

// MilvusKnowledgeCreateResponse represents the response for creating a knowledge base entry
type MilvusKnowledgeCreateResponse struct {
	InsertCount int `json:"insertCount"`
}

// MilvusKnowledgeSearchRequest represents the request for searching milvus knowledge base
type MilvusKnowledgeSearchRequest struct {
	DBName        string      `json:"dbName,omitempty"`
	Collection    string      `json:"collection"`
	PartitionName string      `json:"partitionName,omitempty"`
	VectorField   string      `json:"vectorField"`
	Vectors       [][]float32 `json:"vectors"`
	TopK          int         `json:"topK,omitempty"`
	MetricType    string      `json:"metricType,omitempty"`
	OutputFields  []string    `json:"outputFields,omitempty"`
	Filter        string      `json:"filter,omitempty"`
}

// MilvusKnowledgeSearchResponse represents the response for searching milvus knowledge base
type MilvusKnowledgeSearchResponse struct {
	Results []map[string]any `json:"results"`
}

// MilvusKnowledgeDeleteRequest represents the request for deleting milvus knowledge base entries
// supports delete by filename and contentType with domain filter.
type MilvusKnowledgeDeleteRequest struct {
	StorageType StorageType `json:"storageType"`
	Domain      string      `json:"domain"`
	ContentType string      `json:"contentType"`
	Filename    string      `json:"filename"`
}

// MilvusKnowledgeDeleteResponse represents the response for deleting milvus knowledge base entries
type MilvusKnowledgeDeleteResponse struct {
	Success bool `json:"success"`
}

// MilvusKnowledgeGetRequest represents the request for searching milvus knowledge base entries
// supports search by keyword, filename, contentType with domain filter.
type MilvusKnowledgeGetRequest struct {
	StorageType StorageType `json:"storageType"`
	Domain      string      `json:"domain"`
	ContentType string      `json:"contentType"`
	Filename    string      `json:"filename"`
	Keyword     string      `json:"keyword"`
	TopK        int         `json:"topK,omitempty"`
}

// MilvusKnowledgeGetResponse represents the response for searching milvus knowledge base entries
type MilvusKnowledgeGetResponse struct {
	Results []core.Readable `json:"results"`
}

// CreateMilvusKnowledge processes document and stores chunks into milvus knowledge base
// calls KnowledgeBase.Process to handle document, then Save to store entities.
func (s *MilvusKnowledgeService) CreateMilvusKnowledge(req *MilvusKnowledgeCreateRequest) (*MilvusKnowledgeCreateResponse, *core.Diagnostic) {
	if s.kb == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "knowledge base not initialized",
		}
	}

	// hardcode for simplicity, check if domain is supported, only "technology" is supported for now.
	// given domain, should know the collection name
	if req.Domain != "technology" {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "collection not found for domain: " + req.Domain,
		}
	}

	// process document: load, chunk, and embed
	embedResult, diag := s.kb.Process(req.Data, req.Filename)
	if diag != nil {
		return nil, diag
	}

	if embedResult == nil || len(embedResult.Chunks) == 0 {
		return &MilvusKnowledgeCreateResponse{InsertCount: 0}, nil
	}

	// convert embedding results to MarkdownCollection entities
	entities := make([]any, 0, len(embedResult.Chunks))
	for i, chunk := range embedResult.Chunks {
		// convert []float64 to []float32 for embedding
		var embedding []float32
		if i < len(embedResult.Vectors) && len(embedResult.Vectors[i]) > 0 {
			embedding = make([]float32, len(embedResult.Vectors[i]))
			for j, v := range embedResult.Vectors[i] {
				embedding[j] = float32(v)
			}
		}

		collection := models.MarkdownCollection{
			ChunkID:    int64(i),
			Privacy:    string(core.PublicKB),
			Filename:   req.Filename,
			DocumentID: req.Filename,
			CreateAt:   time.Now().Unix(),
			Domain:     req.ContentType,
			Embedding:  embedding,
			Content:    chunk,
		}
		entities = append(entities, collection)
	}

	// save entities to milvus
	diag = s.kb.Save(entities)
	if diag != nil {
		return nil, diag
	}

	return &MilvusKnowledgeCreateResponse{InsertCount: len(entities)}, nil
}

// SearchMilvusKnowledge searches for milvus knowledge base entries
func (s *MilvusKnowledgeService) SearchMilvusKnowledge(req *MilvusKnowledgeSearchRequest) (*MilvusKnowledgeSearchResponse, *core.Diagnostic) {
	if s.kb == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "knowledge base not initialized",
		}
	}

	searchReq := milvus.SearchRequest{
		MilvusOptions: milvus.MilvusOptions{
			DBName:        req.DBName,
			Collection:    req.Collection,
			PartitionName: req.PartitionName,
		},
		VectorField:  req.VectorField,
		Vectors:      req.Vectors,
		TopK:         req.TopK,
		MetricType:   req.MetricType,
		OutputFields: req.OutputFields,
		Filter:       req.Filter,
	}

	results, diag := milvus.MilvusClientInstance.Search(context.TODO(), searchReq)
	if diag != nil {
		return nil, diag
	}

	return &MilvusKnowledgeSearchResponse{Results: results}, nil
}

// DeleteMilvusKnowledge deletes milvus knowledge base entries by filename and contentType
// supports delete by filename and contentType with domain filter.
func (s *MilvusKnowledgeService) DeleteMilvusKnowledge(req *MilvusKnowledgeDeleteRequest) (*MilvusKnowledgeDeleteResponse, *core.Diagnostic) {
	if s.kb == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "knowledge base not initialized",
		}
	}

	// check if domain is supported, only "technology" is supported for now.
	if req.Domain != "technology" {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "collection not found for domain: " + req.Domain,
		}
	}

	// at least one of filename or contentType is required
	if req.Filename == "" && req.ContentType == "" {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "at least one of filename or contentType is required",
		}
	}

	// build filter string for milvus delete
	filter := buildFilterString(req.ContentType, req.Filename)

	// get options from kb
	opts := s.kb.GetOptions()

	diag := milvus.MilvusClientInstance.Delete(context.TODO(), opts, filter)
	if diag != nil {
		return nil, diag
	}

	return &MilvusKnowledgeDeleteResponse{Success: true}, nil
}

// GetMilvusKnowledge searches milvus knowledge base entries by keyword, filename, or contentType
// supports semantic search by keyword with server-side filtering by domain/contentType/filename.
func (s *MilvusKnowledgeService) GetMilvusKnowledge(req *MilvusKnowledgeGetRequest) (*MilvusKnowledgeGetResponse, *core.Diagnostic) {
	if s.kb == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "knowledge base not initialized",
		}
	}

	// check if domain is supported, only "technology" is supported for now.
	if req.Domain != "technology" {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "collection not found for domain: " + req.Domain,
		}
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	// build filter string for milvus query
	filter := buildFilterString(req.ContentType, req.Filename)

	// if keyword is provided, use semantic search with filter
	if req.Keyword != "" {
		results, diag := s.kb.Search(req.Keyword, topK, filter)
		if diag != nil {
			return nil, diag
		}

		return &MilvusKnowledgeGetResponse{Results: results}, nil
	}

	// if no keyword, return empty results (could implement filter-only search later)
	return &MilvusKnowledgeGetResponse{Results: []core.Readable{}}, nil
}

// buildFilterString builds a milvus filter expression from domain, contentType, and filename
// helper function to construct filter string for server-side filtering.
func buildFilterString(contentType, filename string) string {
	var conditions []string

	if contentType != "" {
		conditions = append(conditions, fmt.Sprintf(`domain == "%s"`, contentType))
	}
	if filename != "" {
		conditions = append(conditions, fmt.Sprintf(`filename == "%s"`, filename))
	}

	if len(conditions) == 0 {
		return ""
	}

	return strings.Join(conditions, " and ")
}
