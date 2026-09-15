package loader

import (
	"github.com/David3310273/go-agent/core"
)

// PDFLoader loads PDF files and extracts text.
// loader for .pdf files, placeholder for future implementation.
type PDFLoader struct{}

// Load processes binary PDF data.
// [auto-added] accepts binary data and filename instead of file path.
func (l *PDFLoader) Load(data []byte, filename string) (string, *core.Diagnostic) {
	// TODO: implement PDF parsing with a third-party library
	return "", nil
}

func (l *PDFLoader) SupportedExtensions() []string {
	return []string{".pdf"}
}
