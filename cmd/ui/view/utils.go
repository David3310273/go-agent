package view

import (
	"strings"
	"time"
)

func TruncateHint(content string, maxWidth int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 1 {
		content = strings.Join(lines, " ")
	}
	if len(content) > maxWidth {
		return content[:maxWidth-3] + "..."
	}
	return content
}

func FormatTimestamp() string {
	now := time.Now()
	return now.Format("[2006-01-02 15:04:05]")
}

func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrapped []string

	for _, line := range lines {
		if len(line) <= width {
			wrapped = append(wrapped, line)
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			wrapped = append(wrapped, line)
			continue
		}

		var currentLine string
		for _, word := range words {
			if len(currentLine)+len(word)+1 <= width {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				if currentLine != "" {
					wrapped = append(wrapped, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			wrapped = append(wrapped, currentLine)
		}
	}

	return strings.Join(wrapped, "\n")
}
