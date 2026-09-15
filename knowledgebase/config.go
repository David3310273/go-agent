package knowledgebase

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
	OpenAITest "github.com/David3310273/go-agent/embedders/openai-text"
	"github.com/David3310273/go-agent/knowledgebase/chunker"
)

// StorageConfig holds configuration for a single storage backend.
// [auto-added] each element in config.json array represents one storage config.
type StorageConfig struct {
	Domain      string         `json:"domain"`
	StorageType string         `json:"storageType"`
	Embedder    EmbedderConfig `json:"embedder"`
	Chunker     ChunkerConfig  `json:"chunker"`
}

// Config holds the configuration for the knowledgebase package.
// [auto-added] config is now an array of storage configs.
type Config []StorageConfig

// EmbedderConfig specifies which embedder to use and its settings.
// embedder configuration within knowledgebase config.
type EmbedderConfig struct {
	Type     string `json:"type"` // e.g., "openai-text"
	RootPath string `json:"-"`    // injected at runtime, not from JSON
}

// ChunkerConfig specifies which chunker to use and its settings.
// [auto-added] chunker configuration within knowledgebase config.
type ChunkerConfig struct {
	Type      string `json:"type"`                // e.g., "markdown", "text"
	ChunkSize int    `json:"chunkSize,omitempty"` // max chunk size in characters
}

const (
	// ConfigPath is the relative path to the knowledgebase config file.
	// config file location within the project root.
	ConfigPath = "knowledgebase/config.json"
)

// LoadConfig reads and parses the knowledgebase config file.
// [auto-added] loads config as array of storage configs from rootPath/knowledgebase/config.json.
func LoadConfig(rootPath string) (Config, error) {
	configPath := path.Join(rootPath, ConfigPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read knowledgebase config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse knowledgebase config: %w", err)
	}

	// inject rootPath into each embedder config
	for i := range cfg {
		cfg[i].Embedder.RootPath = rootPath
	}

	return cfg, nil
}

// GetStorageConfig returns the config for a specific storage type.
// [auto-added] helper to find storage config by type.
func (c Config) GetStorageConfig(storageType string) *StorageConfig {
	for i := range c {
		if c[i].StorageType == storageType {
			return &c[i]
		}
	}
	return nil
}

// GetStorageConfigByDomain returns the config for a specific domain.
// [auto-added] helper to find storage config by domain.
func (c Config) GetStorageConfigByDomain(domain string) *StorageConfig {
	for i := range c {
		if c[i].Domain == domain {
			return &c[i]
		}
	}
	return nil
}

// CreateEmbedder creates an embedder based on the config type.
// factory function that creates the appropriate embedder from config.
func CreateEmbedder(cfg *EmbedderConfig) (core.Embedder, error) {
	switch cfg.Type {
	case "openai-text":
		embedder, diag := OpenAITest.NewOpenAIEmbedder(cfg.RootPath)
		if diag != nil {
			return nil, fmt.Errorf("failed to create openai embedder: %s", diag.Message)
		}
		return embedder, nil
	default:
		return nil, fmt.Errorf("unknown embedder type: %s", cfg.Type)
	}
}

// CreateChunker creates a chunker based on the config type.
// [auto-added] factory function that creates the appropriate chunker from config.
func CreateChunker(cfg *ChunkerConfig) (core.Chunker, error) {
	c := chunker.GetChunkerByType(cfg.Type)
	if c == nil {
		return nil, fmt.Errorf("unknown chunker type: %s", cfg.Type)
	}

	// set chunk size if the chunker supports it
	switch v := c.(type) {
	case *chunker.MarkdownChunker:
		v.ChunkSize = cfg.ChunkSize
	case *chunker.TextChunker:
		v.ChunkSize = cfg.ChunkSize
	}

	return c, nil
}
