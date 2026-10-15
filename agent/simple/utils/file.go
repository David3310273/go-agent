package utils

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
)

// ResolvePath resolves a configured path against the root path.
// moved here from agent/simple/session.go so the agent and the session share one rule.
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

func TailFile(filename string, rounds int) ([]byte, error) {
	lineCount := 2 * rounds
	if lineCount <= 0 {
		return nil, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return nil, nil
	}

	const bufSize = 512

	var tailBytes []byte
	pos := fileSize
	newlinesFound := 0

	for pos > 0 && newlinesFound <= lineCount {
		chunkSize := int64(bufSize)
		if pos < chunkSize {
			chunkSize = pos
		}
		pos -= chunkSize

		buf := make([]byte, chunkSize)
		if _, err := f.ReadAt(buf, pos); err != nil && err != io.EOF {
			return nil, err
		}

		for i := len(buf) - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				newlinesFound++
				if newlinesFound == lineCount+1 {
					tailBytes = append(buf[i+1:], tailBytes...)
					return tailBytes, nil
				}
			}
		}

		tailBytes = append(buf, tailBytes...)
	}

	return tailBytes, nil
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
	sepBytes := []byte(splitter)
	if len(sepBytes) == 0 {
		return nil
	}

	src, err := os.Open(filename)
	if err != nil {
		return err
	}

	const bufSize = 8192
	buf := make([]byte, bufSize)
	overlap := len(sepBytes)
	if overlap < 1 {
		overlap = 1
	}

	var searchBuf []byte
	var splitterEnd int64 = -1
	var offset int64 = 0

	for {
		n, readErr := src.ReadAt(buf, offset)
		if n > 0 {
			searchBuf = append(searchBuf, buf[:n]...)
			idx := bytes.Index(searchBuf, sepBytes)
			if idx >= 0 {
				splitterEnd = offset - int64(len(searchBuf)-n) + int64(idx) + int64(len(sepBytes))
				break
			}
			if len(searchBuf) > overlap {
				searchBuf = searchBuf[len(searchBuf)-overlap:]
			}
		}
		offset += int64(n)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			src.Close()
			return readErr
		}
	}

	src.Close()

	if splitterEnd < 0 {
		return os.Remove(filename)
	}

	src, err = os.Open(filename)
	if err != nil {
		return err
	}
	defer src.Close()

	if _, err := src.Seek(splitterEnd, io.SeekStart); err != nil {
		return err
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	tmp, err := os.CreateTemp(dir, base+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := io.CopyBuffer(tmp, src, make([]byte, bufSize)); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, filename)
}
