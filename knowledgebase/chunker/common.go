package chunker

import "github.com/David3310273/go-agent/core"

// GetChunkerByType returns the appropriate chunker for the given content type.
// [auto-added] factory function to get chunker by content type.
// returns nil if no chunker supports the content type.
func GetChunkerByType(contentType string) core.Chunker {
	switch contentType {
	case "markdown":
		return &MarkdownChunker{}
	case "text":
		return &TextChunker{}
	default:
		return nil
	}
}
