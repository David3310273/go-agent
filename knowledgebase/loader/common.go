package loader

import (
	"path/filepath"
	"strings"

	"github.com/David3310273/go-agent/core"
)

// DefaultLoaders returns a map of built-in loaders keyed by file extension.
// auto-added: convenience function to get all default loaders.
func DefaultLoaders() map[string]core.DocLoader {
	loaders := make(map[string]core.DocLoader)
	for _, loader := range []core.DocLoader{&TXTLoader{}, &MDLoader{}, &PDFLoader{}, &WordLoader{}, &ExcelLoader{}} {
		for _, ext := range loader.SupportedExtensions() {
			loaders[ext] = loader
		}
	}
	return loaders
}

// NewLoaderByExtension creates a loader for the given file extension.
// auto-added: factory function to create loader by extension.
func NewLoaderByExtension(ext string) core.DocLoader {
	ext = strings.ToLower(ext)
	switch ext {
	case ".txt":
		return &TXTLoader{}
	case ".md":
		return &MDLoader{}
	case ".pdf":
		return &PDFLoader{}
	case ".docx":
	case ".doc":
		return &WordLoader{}
	case ".xlsx":
	case ".xls":
		return &ExcelLoader{}
	default:
		return nil
	}
	return nil
}

// GetLoaderByPath finds a suitable loader for the given file path.
// auto-added: helper to match file extension to loader.
func GetLoaderByPath(loaders map[string]core.DocLoader, path string) (core.DocLoader, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	ldr, ok := loaders[ext]
	return ldr, ok
}
