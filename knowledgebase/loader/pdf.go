package loader

import (
	"github.com/David3310273/go-agent/core"
)

// PDFLoader loads PDF files and extracts text.
// auto-added: loader for .pdf files, placeholder for future implementation.
type PDFLoader struct{}

func (l *PDFLoader) Load(path string) (string, *core.Diagnostic) {
	// TODO: implement PDF parsing with a third-party library
	return "", nil
}

func (l *PDFLoader) SupportedExtensions() []string {
	return []string{".pdf"}
}

// Chunk splits PDF content by pages or sections.
// auto-added: placeholder for format-aware chunking.
func (l *PDFLoader) Chunk(content string) []string {
	return nil
}
