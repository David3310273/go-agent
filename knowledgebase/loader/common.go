package loader

import (
	"path/filepath"
	"strings"

	"github.com/David3310273/go-agent/core"
)

// GetLoaderByFilename returns the appropriate loader for the given filename.
// renamed from GetLoaderByPath, selects loader by file extension.
// returns nil if no loader supports the file extension.
func GetLoaderByFilename(filename string) core.DocLoader {
	ext := strings.ToLower(filepath.Ext(filename))
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
