package models

// MarkdownCollection represents a chunk of markdown document stored in Milvus.
// implements core.Readable interface for knowledge base search results.
type MarkdownCollection struct {
	ChunkID    int64     `json:"chunk_id"`
	Privacy    string    `json:"privacy"`
	Filename   string    `json:"filename"`
	DocumentID string    `json:"document_id"`
	CreateAt   int64     `json:"create_at"`
	Domain     string    `json:"domain"`
	Embedding  []float32 `json:"embedding"`
	Content    string    `json:"content"`
}

// GetContent implements core.Readable interface.
// returns the content field for Readable interface.
func (m MarkdownCollection) GetContent() string {
	return m.Content
}
