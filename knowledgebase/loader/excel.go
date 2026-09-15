package loader

import (
	"github.com/David3310273/go-agent/core"
)

// ExcelLoader loads Excel spreadsheets (.xlsx) and extracts data.
// loader for .xlsx files, placeholder for future implementation.
type ExcelLoader struct{}

// Load processes binary Excel file data.
// accepts binary data and filename instead of file path.
func (l *ExcelLoader) Load(data []byte, filename string) (string, *core.Diagnostic) {
	// TODO: implement Excel file parsing with a third-party library
	return "", nil
}

func (l *ExcelLoader) SupportedExtensions() []string {
	return []string{".xlsx", ".xls"}
}
