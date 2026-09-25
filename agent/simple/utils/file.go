package utils

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
)

// ResolvePath resolves a configured path against the root path.
// auto-add: moved here from agent/simple/session.go so the agent and the session share one rule.
// An absolute configured path used to be joined with RootPath, and path.Join turns
// ".." + "/tmp/x/logs/session_%s.log" into the relative "../tmp/x/logs/session_%s.log", so the
// file was written outside the configured directory. An absolute path is kept as it is, a
// relative one is still joined with the root path.
func ResolvePath(rootPath string, configPath string) string {
	if path.IsAbs(configPath) {
		return configPath
	}

	return path.Join(rootPath, configPath)
}

func FindByName(baseUrl string, name string, recursive bool) (string, error) {
	return "", nil
}

func RotateWrite(filename string, splitter string, maxSize int64, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	fileInfo, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil && fileInfo.Size() > maxSize {
		if err := evictUntilSize(filename, splitter, maxSize); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	return nil
}

func evictUntilSize(filename string, splitter string, maxSize int64) error {
	for {
		stat, err := os.Stat(filename)
		if err != nil {
			return err
		}
		if stat.Size() <= maxSize {
			return nil
		}

		if err := evictFirst(filename, splitter); err != nil {
			return err
		}
	}
}

func evictFirst(filename string, splitter string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return nil
	}

	sepBytes := []byte(splitter)
	idx := bytes.Index(content, sepBytes)

	if idx == -1 {
		return os.Remove(filename)
	}

	deleteEnd := idx + len(sepBytes)
	if deleteEnd >= len(content) {
		return os.Remove(filename)
	}

	remaining := content[deleteEnd:]
	return os.WriteFile(filename, remaining, 0644)
}
