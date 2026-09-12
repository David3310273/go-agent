package core

// DocLoader defines the interface for loading different document formats.
// abstraction for multi-format document loading, includes format-aware chunking.
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
	// storage options
	StorageOptions map[string]string
}

type KnowledgePrivacy string

const (
	PublicKB  = "public"
	GroupKB   = "group"
	PrivateKB = "private"
)

// EmbeddingUsage holds token usage statistics for embedding API calls.
// tracks token consumption for cost monitoring.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResult holds the result of embedding a document.
// contains vectors for all chunks, original chunks text, and usage statistics.
type EmbeddingResult struct {
	Vectors [][]float64     `json:"vectors"`
	Chunks  []string        `json:"chunks"`
	Usage   *EmbeddingUsage `json:"usage,omitempty"`
}

// Embedder defines the interface for converting text into vector representations.
// abstraction for embedding models, returns EmbeddingResult with vector and usage.
type Embedder interface {
	Embed(data any) (*EmbeddingResult, *Diagnostic)
}

// KnowledgeBase defines the interface for knowledge management.
// generic interface that uses user-defined entity type for Save and Search.
type KnowledgeBase[T any] interface {
	// embedder is optional, return nil if not using vector db
	GetEmedder() Embedder
	// Process loads a document, and process it.
	Process(docPath string) (*EmbeddingResult, *Diagnostic)
	// Save stores entities of type T directly.
	Save(entities []T) *Diagnostic
	// Search returns results with entity type T.
	Search(keyword any, topK int) ([]T, *Diagnostic)
}
