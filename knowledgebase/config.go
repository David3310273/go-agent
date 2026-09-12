package knowledgebase

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
	OpenAITest "github.com/David3310273/go-agent/embedders/openai-text"
)

// Config holds the configuration for the knowledgebase package.
// config structure for loading embedder settings from config.json.
type Config struct {
	Milvus MilvusKBConfig `json:"milvus"`
}

// MilvusKBConfig holds milvus-specific knowledgebase configuration.
// milvus-level config containing embedder settings.
type MilvusKBConfig struct {
	Embedder EmbedderConfig `json:"embedder"`
}

// EmbedderConfig specifies which embedder to use and its settings.
// embedder configuration within knowledgebase config.
type EmbedderConfig struct {
	Type     string `json:"type"` // e.g., "openai-text"
	RootPath string `json:"-"`    // injected at runtime, not from JSON
}

const (
	// ConfigPath is the relative path to the knowledgebase config file.
	// config file location within the project root.
	ConfigPath = "knowledgebase/config.json"
)

// LoadConfig reads and parses the knowledgebase config file.
// loads config from rootPath/knowledgebase/config.json.
func LoadConfig(rootPath string) (*Config, error) {
	configPath := path.Join(rootPath, ConfigPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read knowledgebase config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse knowledgebase config: %w", err)
	}
	cfg.Milvus.Embedder.RootPath = rootPath
	return &cfg, nil
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
