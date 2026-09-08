package utils

import (
	"bytes"
	"os"
	"path/filepath"
)

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
