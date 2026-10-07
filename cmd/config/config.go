package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/David3310273/go-agent/core"
)

type CommandAppConfig struct {
	core.AppConfig
}

func LoadAppConfig(configPath string) (*CommandAppConfig, *core.Diagnostic) {
	if configPath == "" {
		return nil, nil
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSystemError,
			Message: fmt.Sprintf("failed to read config file: %v", err),
		}
	}

	var config CommandAppConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, &core.Diagnostic{
			Level:   core.SeverityError,
			Code:    core.MessageCodeSystemError,
			Message: fmt.Sprintf("failed to parse config file: %v", err),
		}
	}

	return &config, nil
}
