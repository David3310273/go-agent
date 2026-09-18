package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// auto-add: generic HTTP request/response wrappers

// Request represents a generic request with typed body and headers
type Request[T any] struct {
	Body    T
	Headers map[string]string
	Method  string
}

// Response represents a generic response with typed result
type Response[T any] struct {
	Result T         `json:"result,omitempty"`
	Error  *RPCError `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error returns the error message
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// HTTPConfig holds configuration for HTTP requests
type HTTPConfig struct {
	URL       string
	SessionID string
}

// SendRequest sends a request and returns the raw HTTP response.
// auto-add: caller decides how to parse headers or body
func SendRequest[ReqBody any](
	config HTTPConfig,
	request Request[ReqBody],
) (*http.Response, error) {
	// marshal request body
	reqBody, err := json.Marshal(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// create HTTP request
	req, err := http.NewRequest(request.Method, config.URL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// set headers from request
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}

	// send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}
