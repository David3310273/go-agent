package loader

import (
	"github.com/David3310273/go-agent/core"
)

// WordLoader loads Word documents (.docx) and extracts text.
// auto-added: loader for .docx files, placeholder for future implementation.
type WordLoader struct{}

func (l *WordLoader) Load(path string) (string, *core.Diagnostic) {
	// TODO: implement Word document parsing with a third-party library
	return "", nil
}

func (l *WordLoader) SupportedExtensions() []string {
	return []string{".docx", "doc"}
}

// Chunk splits Word document content by paragraphs or sections.
// auto-added: placeholder for format-aware chunking.
func (l *WordLoader) Chunk(content string) []string {
	return nil
}
