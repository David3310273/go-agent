package tools

import (
	"encoding/json"
	"os"
	"path"

	"github.com/David3310273/go-agent/core"
)

// common functions and constants for tools

const (
	SearchMCPResourcesSchemaPath  = "agent/simple/tools/searchmcp"
	CreateSubsessionSchemaPath    = "agent/simple/tools/createsubsession"
	FileWriterSchemaPath          = "agent/simple/tools/filewriter"
	GetDateSchemaPath             = "agent/simple/tools/getdate"
	SearchKnowledgebaseSchemaPath = "agent/simple/tools/searchkb"
	UseSkillSchemaPath            = "agent/simple/tools/useskill"
)

// LoadToolConfig loads tool configuration from a JSON file.
func LoadToolConfig(rootPath string, schemaPath string, configFile string) core.ToolConfig {
	var config core.ToolConfig
	content, err := os.ReadFile(path.Join(rootPath, schemaPath, configFile))
	if err != nil {
		core.LogStd(core.LogLevelWarn, "configFile: failed to load tool config: %v", err)
		return config
	}
	if err := json.Unmarshal(content, &config); err != nil {
		core.LogStd(core.LogLevelWarn, "configFile: failed to unmarshal tool config: %v", err)
		return config
	}
	return config
}
