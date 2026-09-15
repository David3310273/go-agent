package loader

import (
	"github.com/David3310273/go-agent/core"
)

// MDLoader loads markdown files.
// loader for .md files.
type MDLoader struct{}

// Load processes binary data as markdown content.
// accepts binary data and filename instead of file path.
func (l *MDLoader) Load(data []byte, filename string) (string, *core.Diagnostic) {
	return string(data), nil
}

func (l *MDLoader) SupportedExtensions() []string {
	return []string{".md"}
}
