package chunker

// TextChunker chunks plain text content by paragraphs.
// [auto-added] implements core.Chunker interface for plain text content.
type TextChunker struct {
	ChunkSize int // max chunk size in characters, 0 means no limit
}

// Chunk splits text content by paragraphs.
// placeholder for format-aware chunking.
func (c *TextChunker) Chunk(content string) []string {
	return nil
}

// SupportedTypes returns the content types supported by this chunker.
func (c *TextChunker) SupportedTypes() []string {
	return []string{"text"}
}
