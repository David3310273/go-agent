package loader

import (
	"github.com/David3310273/go-agent/core"
)

// WordLoader loads Word documents (.docx) and extracts text.
// loader for .docx files, placeholder for future implementation.
type WordLoader struct{}

// Load processes binary Word document data.
// [auto-added] accepts binary data and filename instead of file path.
func (l *WordLoader) Load(data []byte, filename string) (string, *core.Diagnostic) {
	// TODO: implement Word document parsing with a third-party library
	return "", nil
}

func (l *WordLoader) SupportedExtensions() []string {
	return []string{".docx", "doc"}
}
