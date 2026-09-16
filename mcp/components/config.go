package components

import (
	"encoding/json"
	"os"
)

// MCPConfig represents the MCP server configuration.
// used for loading configuration from mcp/config.json.
type MCPConfig struct {
	RootPath  string   `json:"rootPath"`
	Port      string   `json:"port"`
	Tools     []string `json:"tools"`
	Prompts   []string `json:"prompts"`
	Resources []string `json:"resources"`
}

// LoadConfig loads MCP config from config.json.
// reads tool configurations for MCP registration.
// config.json should be in mcp/ directory relative to the working directory.
func LoadConfig() (*MCPConfig, error) {
	configPath := "config.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
