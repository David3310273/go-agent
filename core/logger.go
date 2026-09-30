package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// log level constants
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

func logLine(outputPath string, level string, format string, args ...any) {
	msg := fmt.Sprintf("[%s] %s", level, fmt.Sprintf(format, args...))

	var logger *log.Logger
	if outputPath == "" {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
	} else {
		// create directory if not exists
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
			logger.Printf("[WARN] failed to create log directory %s, fallback to stdout: %v", dir, err)
			logger.Output(3, msg)
			return
		}

		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
			logger.Printf("[WARN] failed to open log file %s, fallback to stdout: %v", outputPath, err)
		} else {
			defer f.Close()
			logger = log.New(f, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)
		}
	}

	logger.Output(3, msg)
}

// LogDebug writes a [DEBUG] log. outputPath="" for stdout, or a file path.
func LogDebug(outputPath string, format string, args ...any) {
	logLine(outputPath, LogLevelDebug, format, args...)
}

// LogInfo writes an [INFO] log. outputPath="" for stdout, or a file path.
func LogInfo(outputPath string, format string, args ...any) {
	logLine(outputPath, LogLevelInfo, format, args...)
}

// LogWarn writes a [WARN] log. outputPath="" for stdout, or a file path.
func LogWarn(outputPath string, format string, args ...any) {
	logLine(outputPath, LogLevelWarn, format, args...)
}

// LogError writes an [ERROR] log. outputPath="" for stdout, or a file path.
func LogError(outputPath string, format string, args ...any) {
	logLine(outputPath, LogLevelError, format, args...)
}

// ---- legacy ----

// LogStd is kept for gradual migration.
// Deprecated: use LogDebug/LogInfo/LogWarn/LogError with outputPath.
func LogStd(levelTag, format string, args ...any) {
	switch levelTag {
	case LogLevelDebug:
		LogDebug("", format, args...)
	case LogLevelInfo:
		LogInfo("", format, args...)
	case LogLevelWarn:
		LogWarn("", format, args...)
	case LogLevelError:
		LogError("", format, args...)
	default:
		LogInfo("", format, args...)
	}
}
