package loader

import (
	"github.com/David3310273/go-agent/core"
)

// TXTLoader loads plain text files.
// loader for .txt files.
type TXTLoader struct{}

// Load processes binary data as plain text.
// [auto-added] accepts binary data and filename instead of file path.
func (l *TXTLoader) Load(data []byte, filename string) (string, *core.Diagnostic) {
	return string(data), nil
}

func (l *TXTLoader) SupportedExtensions() []string {
	return []string{".txt"}
}

// Chunk splits text content by paragraphs.
// placeholder for format-aware chunking.
func (l *TXTLoader) Chunk(content string) []string {
	return nil
}
