// Milvus RESTful API v2 client
package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

// =============================================================================
// MilvusStorage - Storage interface implementation
// =============================================================================

// MilvusStorage wraps HTTP client for Milvus RESTful API.
type MilvusStorage struct {
	Client *MilvusClient
}

// MilvusClientInstance is the singleton instance.
var MilvusClientInstance *MilvusStorage

const (
	MilvusStoragePath = "storage/milvus/config.json"
)

// Init initializes the Milvus singleton with the given rootPath.
// lazy init to allow rootPath injection for config resolution.
func Init(rootPath string) {
	if MilvusClientInstance != nil {
		return
	}

	MilvusClientInstance = &MilvusStorage{}
	core.InitStorageClient(MilvusClientInstance, rootPath)

	fmt.Printf("[Milvus] init done: Client=%v\n", MilvusClientInstance.Client)
}

// SetConfig implements core.Storage interface
func (m *MilvusStorage) SetConfig(rootPath string) {
	configPath := path.Join(rootPath, MilvusStoragePath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[Milvus] failed to read config file %s: %v\n", configPath, err)
		return
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("[Milvus] failed to parse config file: %v\n", err)
		return
	}

	m.Client = NewMilvusClient(cfg.Endpoint, cfg.APIKey)
	fmt.Printf("[Milvus] client initialized with endpoint: %s\n", cfg.Endpoint)
}

// Stop closes the Milvus client connection.
func (m *MilvusStorage) Stop() {
	// HTTP client doesn't need explicit close
}

// MilvusOptions holds options for data targeting.
type MilvusOptions struct {
	Type          string
	DBName        string
	Collection    string
	PartitionName string
}

// Insert inserts data into the specified collection.
func (m *MilvusStorage) Insert(ctx context.Context, opts MilvusOptions, data any) (int, *core.Diagnostic) {
	if m.Client == nil {
		return 0, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "milvus client not initialized",
		}
	}

	req := InsertRequest{
		DBName:         opts.DBName,
		CollectionName: opts.Collection,
		PartitionName:  opts.PartitionName,
		Data:           data,
	}

	resp, err := m.Client.Insert(ctx, req)
	if err != nil {
		return 0, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to insert: " + err.Error(),
		}
	}

	return resp.InsertCount, nil
}

// SearchRequest holds all parameters for a search operation.
type SearchRequest struct {
	MilvusOptions
	VectorField  string
	Vectors      [][]float32
	TopK         int
	MetricType   string
	OutputFields []string
	Filter       string
}

// Search searches for vectors in the specified collection.
// returns raw search results as []map[string]any.
func (m *MilvusStorage) Search(ctx context.Context, req SearchRequest) ([]map[string]any, *core.Diagnostic) {
	if m.Client == nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "milvus client not initialized",
		}
	}

	httpReq := MilvusSearchRequest{
		DBName:         req.DBName,
		CollectionName: req.Collection,
		Data:           req.Vectors,
		AnnsField:      req.VectorField,
		Limit:          req.TopK,
		OutputFields:   req.OutputFields,
		Filter:         req.Filter,
	}

	if req.MetricType != "" {
		httpReq.SearchParams = &SearchParams{
			MetricType: req.MetricType,
		}
	}

	resp, err := m.Client.Search(ctx, httpReq)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to search: " + err.Error(),
		}
	}

	// parse data as array of map[string]any
	// Milvus API returns: [{"id": xxx, "distance": 0.5, "field1": value1, ...}, ...]
	var results []map[string]any
	if err := json.Unmarshal(resp.Data, &results); err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to parse search results: " + err.Error(),
		}
	}

	fmt.Printf("[Milvus Debug] Parsed %d results\n", len(results))

	return results, nil
}

// Delete deletes entities from the specified collection.
func (m *MilvusStorage) Delete(ctx context.Context, opts MilvusOptions, filter string) *core.Diagnostic {
	if m.Client == nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "milvus client not initialized",
		}
	}

	req := DeleteRequest{
		DBName:         opts.DBName,
		CollectionName: opts.Collection,
		PartitionName:  opts.PartitionName,
		Filter:         filter,
	}

	err := m.Client.Delete(ctx, req)
	if err != nil {
		return &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeStorageError,
			Message: "failed to delete: " + err.Error(),
		}
	}

	return nil
}

// Flush is a no-op for RESTful API (Milvus auto-flushes).
func (m *MilvusStorage) Flush(ctx context.Context, opts MilvusOptions) *core.Diagnostic {
	return nil
}

// =============================================================================
// MilvusClient - HTTP client for Milvus RESTful API
// =============================================================================

// MilvusClient wraps HTTP client for Milvus RESTful API.
type MilvusClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewMilvusClient creates a new HTTP client for Milvus.
func NewMilvusClient(baseURL, apiKey string) *MilvusClient {
	return &MilvusClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// doRequest executes an HTTP request to Milvus API.
func (c *MilvusClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// MilvusResponse is the common response structure from Milvus API.
type MilvusResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// InsertRequest is the request body for insert operation.
type InsertRequest struct {
	DBName         string `json:"dbName,omitempty"`
	CollectionName string `json:"collectionName"`
	PartitionName  string `json:"partitionName,omitempty"`
	Data           any    `json:"data"`
}

// InsertResponse is the response body for insert operation.
type InsertResponse struct {
	InsertCount int   `json:"insertCount"`
	InsertIds   []any `json:"insertIds"`
}

// Insert inserts entities into a collection.
func (c *MilvusClient) Insert(ctx context.Context, req InsertRequest) (*InsertResponse, error) {
	// debug: print request
	reqJSON, _ := json.Marshal(req)
	fmt.Printf("[Milvus Debug] Insert Request (truncated): %s\n", string(reqJSON[:min(len(reqJSON), 500)]))

	respBody, err := c.doRequest(ctx, http.MethodPost, "/v2/vectordb/entities/insert", req)
	if err != nil {
		return nil, err
	}

	// debug: print response
	fmt.Printf("[Milvus Debug] Insert Response: %s\n", string(respBody))

	var milvusResp MilvusResponse
	if err := json.Unmarshal(respBody, &milvusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if milvusResp.Code != 0 {
		return nil, fmt.Errorf("milvus error %d: %s", milvusResp.Code, milvusResp.Message)
	}

	var insertResp InsertResponse
	if err := json.Unmarshal(milvusResp.Data, &insertResp); err != nil {
		return nil, fmt.Errorf("failed to parse insert response: %w", err)
	}

	return &insertResp, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MilvusSearchRequest is the request body for search operation.
type MilvusSearchRequest struct {
	DBName         string        `json:"dbName,omitempty"`
	CollectionName string        `json:"collectionName"`
	Data           [][]float32   `json:"data"`
	AnnsField      string        `json:"annsField"`
	Limit          int           `json:"limit,omitempty"`
	Offset         int           `json:"offset,omitempty"`
	OutputFields   []string      `json:"outputFields,omitempty"`
	PartitionNames []string      `json:"partitionNames,omitempty"`
	Filter         string        `json:"filter,omitempty"`
	SearchParams   *SearchParams `json:"searchParams,omitempty"`
}

// SearchParams holds search-specific parameters.
type SearchParams struct {
	MetricType string         `json:"metricType"`
	Params     map[string]any `json:"params,omitempty"`
}

// Search searches for vectors in a collection.
// returns raw Milvus response, caller parses data with their entity type.
func (c *MilvusClient) Search(ctx context.Context, req MilvusSearchRequest) (*MilvusResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.SearchParams == nil {
		req.SearchParams = &SearchParams{
			MetricType: "COSINE",
		}
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/v2/vectordb/entities/search", req)
	if err != nil {
		return nil, err
	}

	var milvusResp MilvusResponse
	if err := json.Unmarshal(respBody, &milvusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if milvusResp.Code != 0 {
		return nil, fmt.Errorf("milvus error %d: %s", milvusResp.Code, milvusResp.Message)
	}

	return &milvusResp, nil
}

// DeleteRequest is the request body for delete operation.
type DeleteRequest struct {
	DBName         string `json:"dbName,omitempty"`
	CollectionName string `json:"collectionName"`
	PartitionName  string `json:"partitionName,omitempty"`
	Filter         string `json:"filter"`
}

// Delete deletes entities from a collection.
func (c *MilvusClient) Delete(ctx context.Context, req DeleteRequest) error {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/v2/vectordb/entities/delete", req)
	if err != nil {
		return err
	}

	var milvusResp MilvusResponse
	if err := json.Unmarshal(respBody, &milvusResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if milvusResp.Code != 0 {
		return fmt.Errorf("milvus error %d: %s", milvusResp.Code, milvusResp.Message)
	}

	return nil
}
