package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/David3310273/go-agent/core"
	"github.com/David3310273/go-agent/knowledgebase/loader"
	"github.com/David3310273/go-agent/storage/milvus"
)

// MilvusKnowledgebase contains many same format docs with the uniform domain
// Use this struct directly in your app for agent-level knowledge management.
// generic knowledgebase that uses user-defined entity type.
type MilvusKnowledgebase[T any] struct {
	embedder core.Embedder
	chunker  core.Chunker
	Config   core.KnowledgeBaseConfig
	Options  milvus.MilvusOptions
	name     string
	desc     string
	docDesc  atomic.Pointer[map[string]map[string]any]
	stopCh   chan struct{}
	stopOnce sync.Once
}

var _ core.KnowledgeBase[any] = (*MilvusKnowledgebase[any])(nil)

const (
	MilvusKnowledgeBaseStorageType = "milvus"
	MilvusKnowledgeBaseName        = "milvus_test"
)

// NewMilvusKnowledgebase creates a MilvusKnowledgebase with the given configuration.
// constructor now accepts chunker parameter for content chunking.
func NewMilvusKnowledgebase[T any](embedder core.Embedder, chunker core.Chunker, config core.KnowledgeBaseConfig, options milvus.MilvusOptions) *MilvusKnowledgebase[T] {
	milvus.Init(config.RootPath)

	milvusKnowledgebase := &MilvusKnowledgebase[T]{
		embedder: embedder,
		chunker:  chunker,
		name:     MilvusKnowledgeBaseName,
		desc:     config.Description,
		Options:  options,
		Config:   config,
	}

	milvusKnowledgebase.stopCh = make(chan struct{})
	go milvusKnowledgebase.startDocDescCron()

	return milvusKnowledgebase
}

// GetEmbedder returns the embedder instance.
// getter for embedder, implements core.KnowledgeBase interface.
func (kb *MilvusKnowledgebase[T]) GetEmedder() core.Embedder {
	return kb.embedder
}

func (kb *MilvusKnowledgebase[T]) startDocDescCron() {
	kb.SetDocDescription()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			kb.SetDocDescription()
		case <-kb.stopCh:
			return
		}
	}
}

func (kb *MilvusKnowledgebase[T]) Close() {
	kb.stopOnce.Do(func() {
		close(kb.stopCh)
	})
}

func (kb *MilvusKnowledgebase[T]) SetDocDescription() *core.Diagnostic {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func(ctx context.Context) {
		collectionName := kb.Options.Collection

		results, diag := milvus.MilvusClientInstance.Query(context.Background(), milvus.QueryRequest{
			MilvusOptions: milvus.MilvusOptions{
				Type:          kb.Options.Type,
				DBName:        kb.Options.DBName,
				Collection:    collectionName,
				PartitionName: kb.Options.PartitionName,
			},
			Filter:       "",
			GroupFields:  []string{"domain", "filename", "description"},
			OutputFields: []string{"domain", "filename", "description"},
		})

		if diag != nil {
			core.LogStd(core.LogLevelWarn, "failed to query doc description: %s", diag.ToString())
			return
		}

		descriptions := make(map[string]map[string]any)
		for _, result := range results {
			domain := result["domain"].(string)
			filename := result["filename"].(string)
			description := result["description"].(string)
			if _, ok := descriptions[domain]; !ok {
				descriptions[domain] = make(map[string]any)
			}
			descriptions[domain][filename] = description
		}

		kb.docDesc.Store(&descriptions)
		core.LogStd(core.LogLevelInfo, "doc description of kb %s loaded: %#v", kb.name, *kb.docDesc.Load())
		done <- struct{}{}
	}(ctx)

	select {
	case <-ctx.Done():
		core.LogStd(core.LogLevelWarn, "fetch doc description of kb %s failed in 30s", kb.name)
		return &core.Diagnostic{
			Message: "fetch doc description of kb %s failed in 30s",
			Code:    core.MessageCodeFetchKBDescError,
			Level:   core.SeverityError,
		}
	case <-done:
		core.LogStd(core.LogLevelInfo, "fetch doc description of kb %s success", kb.name)
		return &core.Diagnostic{
			Message: "fetch doc description of kb %s success",
			Code:    core.MessageCodeSuccess,
			Level:   core.SeverityInfo,
		}
	}

}

func (kb *MilvusKnowledgebase[T]) GetDocDescription(collectionName, domain string) map[string]any {
	desc := kb.docDesc.Load()
	if desc == nil {
		return nil
	}

	if domain != "" {
		if domainDocs, ok := (*desc)[domain]; ok {
			if domainDocs != nil {
				return domainDocs
			}
		}
		return nil
	}

	result := make(map[string]any, len(*desc))
	for domain, docsSpec := range *desc {
		result[domain] = docsSpec
	}
	return result
}

// Delete removes entities from Milvus by domain and filename filter.
func (kb *MilvusKnowledgebase[T]) Delete(collectionName, domain, filename string) *core.Diagnostic {
	if collectionName == "" {
		collectionName = kb.Options.Collection
	}

	filterParts := []string{}
	if domain != "" {
		filterParts = append(filterParts, fmt.Sprintf(`domain == "%s"`, domain))
	}
	if filename != "" {
		filterParts = append(filterParts, fmt.Sprintf(`filename == "%s"`, filename))
	}
	filter := strings.Join(filterParts, " && ")

	diag := milvus.MilvusClientInstance.Delete(context.Background(), milvus.MilvusOptions{
		Type:          kb.Options.Type,
		DBName:        kb.Options.DBName,
		Collection:    collectionName,
		PartitionName: kb.Options.PartitionName,
	}, filter)

	if diag == nil {
		if domain != "" {
			desc := kb.docDesc.Load()
			if desc != nil {
				newDesc := make(map[string]map[string]any, len(*desc))
				for k, v := range *desc {
					newDesc[k] = v
				}
				delete(newDesc, domain)
				kb.docDesc.Store(&newDesc)
			}
		} else if filename != "" {
			desc := kb.docDesc.Load()
			if desc != nil {
				newDesc := make(map[string]map[string]any, len(*desc))
				for domain, docsSpec := range *desc {
					newDesc[domain] = docsSpec
				}
				for d, dVal := range newDesc {
					if _, ok := dVal[filename]; ok {
						newDesc[d] = make(map[string]any)
					}
				}
				kb.docDesc.Store(&newDesc)
			}
		}
	}

	return diag
}

// Process loads a document from binary data, chunks it, and embeds each chunk.
// accepts binary data and filename instead of file path.
// implements core.KnowledgeBase interface, auto-selects loader by filename.
func (kb *MilvusKnowledgebase[T]) Process(data []byte, filename string) (*core.EmbeddingResult, *core.Diagnostic) {
	docLoader := loader.GetLoaderByFilename(filename)
	if docLoader == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeDocNotFound,
			Message: "no loader found for file: " + filename,
		}
	}

	doc, diag := docLoader.Load(data, filename)
	if diag != nil {
		return nil, diag
	}

	// use chunker to split content
	if kb.chunker == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "chunker not initialized",
		}
	}
	chunks := kb.chunker.Chunk(doc)

	core.LogStd(core.LogLevelInfo, "kb finished chunk file %s, total chunks: %d", filename, len(chunks))

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

	core.LogStd(core.LogLevelInfo, "kb finished embedding file: %s", filename)

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
	core.LogStd(core.LogLevelDebug, "milvus insert: count=%d", count)
	// sync local doc description
	if diag == nil {
		go func() {
			kb.SetDocDescription()
		}()
	} else {
		core.LogStd(core.LogLevelError, "err in insert doc: %s", diag.ToString())
	}
	return diag
}

// Search embeds keyword and searches for similar vectors in Milvus.
// filter parameter for server-side filtering in vector db.
// implements core.KnowledgeBase interface, embeds keyword and searches.
func (kb *MilvusKnowledgebase[T]) Search(keyword any, topK int, filter string) ([]core.Readable, *core.Diagnostic) {
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

	// create search request with filter
	req := milvus.SearchRequest{
		MilvusOptions: kb.Options,
		VectorField:   "embedding",
		Vectors:       [][]float32{vec32},
		TopK:          topK,
		MetricType:    "COSINE",
		OutputFields:  []string{"*"},
		Filter:        filter,
	}

	results, diag := milvus.MilvusClientInstance.Search(context.Background(), req)
	if diag != nil {
		return nil, diag
	}

	// convert []map[string]any to []core.Readable
	readables := make([]core.Readable, 0, len(results))
	for _, r := range results {
		// try to convert to T first
		entityJSON, _ := json.Marshal(r)
		var entity T
		if err := json.Unmarshal(entityJSON, &entity); err != nil {
			return nil, &core.Diagnostic{
				Level:   core.SeverityError,
				Code:    core.MessageCodeStorageError,
				Message: "failed to convert search result to entity: " + err.Error(),
			}
		}

		// check if T implements Readable
		if readable, ok := any(entity).(core.Readable); ok {
			readables = append(readables, readable)
		} else {
			// T is any or doesn't implement Readable, wrap map[string]any as MapReadable
			readables = append(readables, core.MapReadable(r))
		}
	}

	return readables, nil
}

func (kb *MilvusKnowledgebase[T]) GetName() string {
	return kb.name
}

func (kb *MilvusKnowledgebase[T]) GetDescription() string {
	return kb.desc
}

// Query queries entities from Milvus by domain and filename filter.
// collectionName: target collection, domain: document domain, filename: document filename.
func (kb *MilvusKnowledgebase[T]) Query(collectionName, domain, filename string, outputFields, groupFields []string) ([]map[string]any, *core.Diagnostic) {
	if collectionName == "" {
		collectionName = kb.Options.Collection
	}

	filterParts := []string{}
	if domain != "" {
		filterParts = append(filterParts, fmt.Sprintf(`domain == "%s"`, domain))
	}
	if filename != "" {
		filterParts = append(filterParts, fmt.Sprintf(`filename == "%s"`, filename))
	}
	filter := strings.Join(filterParts, " && ")

	req := milvus.QueryRequest{
		MilvusOptions: milvus.MilvusOptions{
			Type:          kb.Options.Type,
			DBName:        kb.Options.DBName,
			Collection:    collectionName,
			PartitionName: kb.Options.PartitionName,
		},
		Filter:       filter,
		OutputFields: outputFields,
		GroupFields:  groupFields,
	}

	core.LogStd(core.LogLevelDebug, "milvus query: filter=%s, outputFields=%v, db=%s, collection=%s, groupFields=%v", req.Filter, req.OutputFields, req.DBName, req.Collection, req.GroupFields)

	results, diag := milvus.MilvusClientInstance.Query(context.Background(), req)
	if diag != nil {
		return nil, diag
	}

	core.LogStd(core.LogLevelDebug, "milvus query result: %#v", results)

	return results, nil
}

// HasCollection checks if a collection exists in Milvus.
func (kb *MilvusKnowledgebase[T]) HasCollection() (bool, *core.Diagnostic) {
	return milvus.MilvusClientInstance.HasCollection(context.Background(), kb.Options)
}

// CreateCollection creates a new collection in Milvus.
func (kb *MilvusKnowledgebase[T]) CreateCollection(req milvus.CreateCollectionRequest) *core.Diagnostic {
	return milvus.MilvusClientInstance.CreateCollection(context.Background(), req)
}
