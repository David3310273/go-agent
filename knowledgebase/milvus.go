package knowledgebase

import (
	"context"
	"encoding/json"
	"log"

	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/storage/milvus"
)

// MilvusKnowledgebase contains many same format docs with the uniform domain
// Use this struct directly in your app for agent-level knowledge management.
// auto-added: generic knowledgebase that uses user-defined entity type.
type MilvusKnowledgebase[T any] struct {
	loader   core.DocLoader
	embedder core.Embedder
	Config   core.KnowledgeBaseConfig
	Options  milvus.MilvusOptions
}

// MilvusEntity represents a single entity to be inserted into Milvus.
// auto-added: default entity structure for Milvus RESTful API insert.
type MilvusEntity struct {
	ChunkID    int64     `json:"chunk_id"`
	Privacy    string    `json:"privacy"`
	Filename   string    `json:"filename"`
	DocumentID string    `json:"document_id"`
	CreateAt   int64     `json:"create_at"`
	Domain     string    `json:"domain"`
	Embedding  []float32 `json:"embedding"`
	Content    string    `json:"content"`
}

var _ core.KnowledgeBase[MilvusEntity] = (*MilvusKnowledgebase[MilvusEntity])(nil)

// NewMilvusKnowledgebase creates a MilvusKnowledgebase with the given configuration.
// auto-added: constructor for generic MilvusKnowledgebase with entity type injection.
func NewMilvusKnowledgebase[T any](loader core.DocLoader, embedder core.Embedder, config core.KnowledgeBaseConfig, options milvus.MilvusOptions) *MilvusKnowledgebase[T] {
	return &MilvusKnowledgebase[T]{
		loader:   loader,
		embedder: embedder,
		Options:  options,
		Config:   config,
	}
}

// GetLoader returns the loader.
// auto-added: implements core.KnowledgeBase interface.
func (kb *MilvusKnowledgebase[T]) GetLoader(path string) core.DocLoader {
	return kb.loader
}

// GetEmbedder returns the embedder instance.
// auto-added: getter for embedder, implements core.KnowledgeBase interface.
func (kb *MilvusKnowledgebase[T]) GetEmedder() core.Embedder {
	return kb.embedder
}

// GetDatabase returns the target Milvus database name.
// auto-added: getter for database configuration.
func (kb *MilvusKnowledgebase[T]) GetOptions() milvus.MilvusOptions {
	return kb.Options
}

// Save stores entities directly into Milvus.
// auto-added: generic save that accepts user-built entities of type T.
func (kb *MilvusKnowledgebase[T]) Save(entities []T) *core.Diagnostic {
	if len(entities) == 0 {
		return nil
	}

	count, diag := milvus.MilvusClientInstance.Insert(context.Background(), kb.Options, entities)
	log.Printf("count: %d", count)
	return diag
}

// Search embeds keyword and searches for similar vectors in Milvus.
// auto-added: implements core.KnowledgeBase interface, embeds keyword and searches.
func (kb *MilvusKnowledgebase[T]) Search(keyword any, topK int) ([]T, *core.Diagnostic) {
	// convert keyword to string
	text, ok := keyword.(string)
	if !ok {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "keyword must be a string",
		}
	}

	// embed the keyword
	result, diag := kb.embedder.Embed(text)
	log.Printf("query %s emedding: %v", keyword, result)
	if diag != nil {
		return nil, diag
	}
	if result == nil || len(result.Vectors) == 0 {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to embed keyword",
		}
	}

	// convert []float64 to []float32
	vec32 := make([]float32, len(result.Vectors[0]))
	for i, v := range result.Vectors[0] {
		vec32[i] = float32(v)
	}

	// create search request
	req := milvus.SearchRequest{
		MilvusOptions: kb.Options,
		VectorField:   "embedding",
		Vectors:       [][]float32{vec32},
		TopK:          topK,
		MetricType:    "COSINE",
		OutputFields:  []string{"*"},
	}

	results, diag := milvus.MilvusClientInstance.Search(context.Background(), req)
	if diag != nil {
		return nil, diag
	}

	// convert []map[string]any to []T via JSON
	entities := make([]T, 0, len(results))
	for _, r := range results {
		entityJSON, _ := json.Marshal(r)
		var entity T
		if err := json.Unmarshal(entityJSON, &entity); err != nil {
			return nil, &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeStorageError,
				Message: "failed to convert search result to entity: " + err.Error(),
			}
		}
		entities = append(entities, entity)
	}

	return entities, nil
}
