package loader

import (
	"os"

	"github.com/David3310273/go-agent/core"
)

// TXTLoader loads plain text files.
// loader for .txt files.
type TXTLoader struct{}

func (l *TXTLoader) Load(path string) (string, *core.Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", &core.Diagnostic{}
	}
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
