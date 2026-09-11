package loader

import (
	"os"
	"regexp"
	"strings"

	"github.com/David3310273/go-agent/core"
)

// MDLoader loads markdown files.
// auto-added: loader for .md files.
type MDLoader struct {
	ChunkSize int // max chunk size in characters, 0 means no limit
}

func (l *MDLoader) Load(path string) (string, *core.Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", &core.Diagnostic{}
	}
	return string(data), nil
}

func (l *MDLoader) SupportedExtensions() []string {
	return []string{".md"}
}

// Chunk splits markdown content by headings first, then by punctuation for long sections.
// auto-added: two-level chunking - by heading, then by sentence boundaries.
func (l *MDLoader) Chunk(content string) []string {
	if content == "" {
		return nil
	}

	content = strings.TrimSpace(content)
	chunkSize := l.ChunkSize

	// first level: split by headings
	sections := splitByHeading(content)

	var chunks []string
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// if no size limit or section is small enough, keep as one chunk
		if chunkSize <= 0 || len(section) <= chunkSize {
			chunks = append(chunks, section)
		} else {
			// second level: split long section by punctuation
			subChunks := splitByPunctuation(section, chunkSize)
			chunks = append(chunks, subChunks...)
		}
	}

	return chunks
}

// splitByHeading splits content by markdown headings (# ## ### etc.).
// auto-added: helper for first-level chunking.
func splitByHeading(content string) []string {
	headingRegex := regexp.MustCompile(`(?m)^#{1,6}\s+.+$`)
	locations := headingRegex.FindAllStringIndex(content, -1)

	if len(locations) == 0 {
		return []string{content}
	}

	var sections []string

	// content before first heading
	if locations[0][0] > 0 {
		sections = append(sections, content[:locations[0][0]])
	}

	// each heading section
	for i, loc := range locations {
		start := loc[0]
		end := len(content)
		if i+1 < len(locations) {
			end = locations[i+1][0]
		}
		sections = append(sections, content[start:end])
	}

	return sections
}

// splitByPunctuation splits content by sentence-ending punctuation with overlap.
// Each chunk starts with the last sentence of the previous chunk.
// auto-added: helper for second-level chunking with sentence overlap.
func splitByPunctuation(content string, maxSize int) []string {
	// split by sentence-ending punctuation: . ! ? 。！？
	sentenceRegex := regexp.MustCompile(`([.!?。！？]\s*)`)
	parts := sentenceRegex.Split(content, -1)
	delimiters := sentenceRegex.FindAllString(content, -1)

	// reconstruct sentences with their punctuation
	var sentences []string
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i < len(delimiters) {
			sentences = append(sentences, part+delimiters[i])
		} else {
			sentences = append(sentences, part)
		}
	}

	if len(sentences) == 0 {
		return nil
	}

	var chunks []string
	var current strings.Builder
	var lastSentence string

	for _, sentence := range sentences {
		// if adding this sentence exceeds maxSize, save current and start new
		if current.Len()+len(sentence) > maxSize && current.Len() > 0 {
			chunk := strings.TrimSpace(current.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			// start new chunk with overlap (last sentence)
			current.Reset()
			if lastSentence != "" {
				current.WriteString(lastSentence)
			}
		}

		current.WriteString(sentence)
		lastSentence = sentence
	}

	// don't forget the last chunk
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}
