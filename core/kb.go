package core

// DocLoader defines the interface for loading different document formats.
// auto-added: abstraction for multi-format document loading, includes format-aware chunking.
type DocLoader interface {
	Load(path string) (string, *Diagnostic)
	Chunk(content string) []string
	SupportedExtensions() []string
}

type KnowledgeBaseConfig struct {
	RootPath string
	// domain
	Domain string `json:"domain"`
	// Description of the knowledge base
	Description string `json:"description"`
}

type KnowledgePrivacy string

const (
	PublicKB  = "public"
	GroupKB   = "group"
	PrivateKB = "private"
)

// EmbeddingUsage holds token usage statistics for embedding API calls.
// auto-added: tracks token consumption for cost monitoring.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResult holds the result of embedding a document.
// auto-added: contains vectors for all chunks, original chunks text, and usage statistics.
type EmbeddingResult struct {
	Vectors [][]float64     `json:"vectors"`
	Chunks  []string        `json:"chunks"`
	Usage   *EmbeddingUsage `json:"usage,omitempty"`
}

// Embedder defines the interface for converting text into vector representations.
// auto-added: abstraction for embedding models, returns EmbeddingResult with vector and usage.
type Embedder interface {
	Embed(data any) (*EmbeddingResult, *Diagnostic)
}

// KnowledgeBase defines the interface for knowledge management.
// generic interface that uses user-defined entity type for Save and Search.
type KnowledgeBase[T any] interface {
	// GetLoader returns the appropriate loader for the given file path.
	GetLoader(path string) DocLoader
	GetEmedder() Embedder
	// Save stores entities of type T directly.
	Save(entities []T) *Diagnostic
	// Search returns results with entity type T.
	Search(keyword any, topK int) ([]T, *Diagnostic)
}

// ProcessDoc loads a document, chunks it, embeds each chunk, and returns a single EmbeddingResult.
// auto-added: uses kb.GetLoader for format-aware loading and chunking, returns one EmbeddingResult per document.
func ProcessDoc[T any](kb KnowledgeBase[T], docPath string) (*EmbeddingResult, *Diagnostic) {
	loader := kb.GetLoader(docPath)
	if loader == nil {
		return nil, &Diagnostic{
			Level:   SeverityError,
			Code:    MessageCodeDocNotFound,
			Message: "no loader found for file: " + docPath,
		}
	}

	doc, diag := loader.Load(docPath)
	if diag != nil {
		return nil, diag
	}

	chunks := loader.Chunk(doc)
	embedder := kb.GetEmedder()

	result := &EmbeddingResult{
		Vectors: make([][]float64, 0, len(chunks)),
		Chunks:  chunks,
	}

	for _, chunk := range chunks {
		// TODO: use concurrency here
		embedResult, diag := embedder.Embed(chunk)
		if diag != nil {
			return nil, diag
		}

		if embedResult != nil && len(embedResult.Vectors) > 0 {
			// only one string of chunk, return the first
			result.Vectors = append(result.Vectors, embedResult.Vectors[0])
			if embedResult.Usage != nil {
				if result.Usage == nil {
					result.Usage = &EmbeddingUsage{}
				}
				result.Usage.PromptTokens += embedResult.Usage.PromptTokens
				result.Usage.TotalTokens += embedResult.Usage.TotalTokens
			}
		}
	}

	return result, nil
}
