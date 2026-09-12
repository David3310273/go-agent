package loader

import (
	"path/filepath"
	"strings"

	"github.com/David3310273/go-agent/core"
)

// GetLoaderByPath returns the appropriate loader for the given file path.
// returns nil if no loader supports the file extension.
func GetLoaderByPath(path string) core.DocLoader {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		return &TXTLoader{}
	case ".md":
		return &MDLoader{}
	case ".pdf":
		return &PDFLoader{}
	case ".docx", ".doc":
		return &WordLoader{}
	case ".xlsx", ".xls":
		return &ExcelLoader{}
	default:
		return nil
	}
}
