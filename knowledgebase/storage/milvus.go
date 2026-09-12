package storage

import (
	"context"
	"encoding/json"
	"log"

	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/knowledgebase/loader"
	"github.com/David3310273/go-agent/storage/milvus"
)

// MilvusKnowledgebase contains many same format docs with the uniform domain
// Use this struct directly in your app for agent-level knowledge management.
// generic knowledgebase that uses user-defined entity type.
type MilvusKnowledgebase[T any] struct {
	embedder core.Embedder
	Config   core.KnowledgeBaseConfig
	Options  milvus.MilvusOptions
}

var _ core.KnowledgeBase[any] = (*MilvusKnowledgebase[any])(nil)

const (
	KnowledgeBaseStorageType = "milvus"
)

// NewMilvusKnowledgebase creates a MilvusKnowledgebase with the given configuration.
// constructor for generic MilvusKnowledgebase with entity type injection.
func NewMilvusKnowledgebase[T any](embedder core.Embedder, config core.KnowledgeBaseConfig, options milvus.MilvusOptions) *MilvusKnowledgebase[T] {
	milvus.Init(config.RootPath)

	return &MilvusKnowledgebase[T]{
		embedder: embedder,
		Options:  options,
		Config:   config,
	}
}

// GetEmbedder returns the embedder instance.
// getter for embedder, implements core.KnowledgeBase interface.
func (kb *MilvusKnowledgebase[T]) GetEmedder() core.Embedder {
	return kb.embedder
}

// Process loads a document, chunks it, and embeds each chunk.
// implements core.KnowledgeBase interface, auto-selects loader by docPath.
func (kb *MilvusKnowledgebase[T]) Process(docPath string) (*core.EmbeddingResult, *core.Diagnostic) {
	docLoader := loader.GetLoaderByPath(docPath)
	if docLoader == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeDocNotFound,
			Message: "no loader found for file: " + docPath,
		}
	}

	doc, diag := docLoader.Load(docPath)
	if diag != nil {
		return nil, diag
	}

	chunks := docLoader.Chunk(doc)
	embedder := kb.GetEmedder()

	result := &core.EmbeddingResult{
		Vectors: make([][]float64, 0, len(chunks)),
		Chunks:  chunks,
	}

	for _, chunk := range chunks {
		embedResult, diag := embedder.Embed(chunk)
		if diag != nil {
			return nil, diag
		}

		if embedResult != nil && len(embedResult.Vectors) > 0 {
			result.Vectors = append(result.Vectors, embedResult.Vectors[0])
			if embedResult.Usage != nil {
				if result.Usage == nil {
					result.Usage = &core.EmbeddingUsage{}
				}
				result.Usage.PromptTokens += embedResult.Usage.PromptTokens
				result.Usage.TotalTokens += embedResult.Usage.TotalTokens
			}
		}
	}

	return result, nil
}

// GetDatabase returns the target Milvus database name.
// getter for database configuration.
func (kb *MilvusKnowledgebase[T]) GetOptions() milvus.MilvusOptions {
	return kb.Options
}

// Save stores entities directly into Milvus.
// generic save that accepts user-built entities of type T.
func (kb *MilvusKnowledgebase[T]) Save(entities []T) *core.Diagnostic {
	if len(entities) == 0 {
		return nil
	}

	count, diag := milvus.MilvusClientInstance.Insert(context.Background(), kb.Options, entities)
	log.Printf("count: %d", count)
	return diag
}

// Search embeds keyword and searches for similar vectors in Milvus.
// implements core.KnowledgeBase interface, embeds keyword and searches.
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
