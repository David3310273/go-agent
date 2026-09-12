package loader

import (
	"github.com/David3310273/go-agent/core"
)

// ExcelLoader loads Excel spreadsheets (.xlsx) and extracts data.
// loader for .xlsx files, placeholder for future implementation.
type ExcelLoader struct{}

func (l *ExcelLoader) Load(path string) (string, *core.Diagnostic) {
	// TODO: implement Excel file parsing with a third-party library
	return "", nil
}

func (l *ExcelLoader) SupportedExtensions() []string {
	return []string{".xlsx", ".xls"}
}

// Chunk splits Excel content by sheets or rows.
// placeholder for format-aware chunking.
func (l *ExcelLoader) Chunk(content string) []string {
	return nil
}
