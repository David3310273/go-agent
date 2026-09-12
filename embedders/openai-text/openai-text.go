package OpenAITest

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

// OpenAIEmbedder uses OpenAI's embedding API.
// placeholder for OpenAI embedding implementation.
type OpenAIEmbedder struct {
	RootPath string
	Config   EmbedderConfig
}

// EmbedderConfig holds the configuration for the embedder.
// config structure for loading from config.json.
type EmbedderConfig struct {
	Name   string `json:"name"`
	APIKey struct {
		Key string `json:"key"`
	} `json:"apiKey"`
	BaseUrl         string `json:"baseUrl"`
	VectorDimension int    `json:"vectorDimension"`
}

// OpenAIEmbedderRequest holds the request body for embedding API.
// request structure for embedding API call.
type OpenAIEmbedderRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Dimensions     int    `json:"dimensions,omitempty"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

// OpenAIEmbedderResponse holds the response from embedding API.
// response structure for embedding API call.
type OpenAIEmbedderResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

const (
	EmbedderConfigPath = "embedders/openai-text/config.json"
)

func NewOpenAIEmbedder(rootPath string) (*OpenAIEmbedder, *core.Diagnostic) {
	embedder := &OpenAIEmbedder{
		RootPath: rootPath,
	}

	configPath := path.Join(rootPath, EmbedderConfigPath)
	embedderConfig := EmbedderConfig{}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeEmbedderConfigError,
			Message: err.Error(),
		}
	}

	if err := json.Unmarshal(configData, &embedderConfig); err != nil {
		log.Printf("failed to unmarshal config file: %v", err)
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeEmbedderConfigError,
			Message: err.Error(),
		}
	}

	embedder.Config = embedderConfig

	return embedder, nil
}

// Embed converts text into a vector representation using HTTP API call.
// implements embedding via HTTP POST to OpenAI-compatible API, returns EmbeddingResult with vector and usage.
func (e *OpenAIEmbedder) Embed(data any) (*core.EmbeddingResult, *core.Diagnostic) {
	// use default config if APIKey or Model is not set
	apiKey := e.Config.APIKey.Key
	model := e.Config.Name

	text, ok := data.(string)
	if !ok {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "invalid input type, expected string",
		}
	}

	body := OpenAIEmbedderRequest{
		Model:          model,
		Input:          text,
		Dimensions:     e.Config.VectorDimension,
		EncodingFormat: "float",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to marshal request: " + err.Error(),
		}
	}

	req, err := http.NewRequest("POST", e.Config.BaseUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to create request: " + err.Error(),
		}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to send request: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to read response: " + err.Error(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "embedding API error: " + string(respBody),
		}
	}

	var embeddingResp OpenAIEmbedderResponse
	if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to parse response: " + err.Error(),
		}
	}

	if len(embeddingResp.Data) == 0 {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "no embedding data in response",
		}
	}

	// single string input returns single embedding at index 0.
	return &core.EmbeddingResult{
		Vectors: [][]float64{embeddingResp.Data[0].Embedding},
		Usage: &core.EmbeddingUsage{
			PromptTokens: embeddingResp.Usage.PromptTokens,
			TotalTokens:  embeddingResp.Usage.TotalTokens,
		},
	}, nil
}

// Ensure OpenAIEmbedder implements core.Embedder interface.
var _ core.Embedder = (*OpenAIEmbedder)(nil)
