package core

import (
	"log"
	"os"
)

// NewLogger, create logger with custom output path
// if outputPath is empty, logs to stdout; otherwise logs to the specified file
func NewLogger(outputPath string) *log.Logger {
	var Logger *log.Logger
	if outputPath == "" {
		Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
	} else {
		file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
			Logger.Printf("failed to open log file %s, fallback to stdout: %v", outputPath, err)
		} else {
			Logger = log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
		}
	}
	return Logger
}
