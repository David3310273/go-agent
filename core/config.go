package core

import "time"

type Model int

const (
	ModelQwen Model = iota
	ModelClaude
)

type AgentCoreConfig struct {
	//  project root path for resolving relative paths at runtime
	RootPath string
	Agent    AgentConfig   `json:"agent"`
	Session  SessionConfig `json:"session"`
	Model    []ModelConfig `json:"model"`
	Tool     []ToolConfig  `json:"tool"`
}

type PlanType int

const (
	PlanTrial PlanType = iota
	PlanClassic
	PlanEnterprise
)

type Plan struct {
	Name    PlanType `json:"name"`
	MaxUser int      `json:"maxUser"`
	// the max round of one conversation
	SessionMaxRounds int `json:"sessionMaxRounds"`
	SubSessionLimit  int `json:"subSessionLimit"`
	// start date
	StartUseDate time.Time `json:"startDate"`
	// expiry date
	ExpiryDate time.Time `json:"expiryDate"`
}

type AppConfig struct {
	//  project root path, used as base path for all runtime file operations
	RootPath           string        `json:"rootPath"`
	TimeZone           string        `json:"timeZone"`
	Language           LanguageType  `json:"language"`
	LogPath            string        `json:"logPath"`
	MaxWaitingSeconds  int           `json:"maxWaitingSeconds"`
	MaxWaitingRequest  int           `json:"maxWaitingRequest"`
	RequestQueueLength int           `json:"requestQueueLength"`
	Service            ServiceConfig `json:"service"`
}

type ServiceConfig struct {
	Port int `json:"port"`
	// max waiting seconds
	MaxCloseWaitingSeconds int `json:"maxCloseWaitingSeconds"`
}

type LoggerConfig struct {
	AgentLogPath   string `json:"agentLogPath"`
	SessionLogPath string `json:"sessionLogPath"`
}

type AgentConfig struct {
	// version
	Version string `json:"version"`
	// Plan
	UserPlan Plan `json:"userPlan"`
	// Prompt
	Prompt PromptConfig `json:"prompt"`
	// skills
	Skill SkillConfig `json:"skill"`
	// knowledge base
	KnowledgeBase KnowledgeBaseConfig `json:"knowledgeBase"`
	// history path
	History HistoryConfig `json:"history"`
	// tool config
	Tool []ToolConfig `json:"tool"`
	// log path
	LogPath string `json:"logPath"`
	// streaming
	Streaming bool `json:"streaming"`
	// question buffer size
	QuestionBufferSize int `json:"questionBufferSize"`
}

type PromptConfig struct {
	// prompt file path
	Paths []string `json:"paths"`
	// buffer size
	BufferSizeInKB int `json:"bufferSizeInKB"`
	// output root
	OutputPathFormat string `json:"outputPathFormat"`
}

type ToolConfig struct {
	// name
	Name string `json:"name"`
	// tool schema filename
	Schema string `json:"schema"`
	// brief description
	Description string `json:"description"`
	//  project root path, propagated at runtime for resolving tool resource files
	RootPath string
}

type KnowledgeBaseConfig struct {
	// knowledge base file path
	Paths []string `json:"paths"`
	// buffer size
	BufferSize int `json:"bufferSize"`
	// output root
	OutputPathFormat string `json:"outputPathFormat"`
}

type BenchmarkerConfig struct {
	MaxTimeout     int    `json:"maxTimeout"`
	Port           int    `json:"port"`
	OutputPath     string `json:"outputPath"`
	FilenameFormat string `json:"filenameFormat"`
}

type SkillConfig struct {
	// skill file path
	Paths []string `json:"paths"`
	// buffer size
	BufferSize int `json:"bufferSize"`
	// output root
	OutputPathFormat string `json:"outputPathFormat"`
}

type HistoryConfig struct {
	// memory file for whole agent
	OutputFilenameFormat string `json:"memoryFilePathFormat"`
	// max size in bytes
	BufferSize int `json:"maxSize"`
	// history folder path
	Paths []string `json:"paths"`
}

type SessionConfig struct {
	//  project root path, propagated at runtime for resolving file paths
	RootPath string
	// reAct max rounds
	ReActMaxRounds int `json:"reActMaxRounds"`
	// memory file path for current session
	MemoryFilePathFormat string `json:"memoryFilePathFormat"`
	MemoryFileSplitter   string
	MemoryFileSize       int64

	// retry on lock
	MaxRetryOnLock int `json:"maxRetryOnLock"`

	// log path
	LogPath string `json:"logPath"`
}

type APIKey struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

type ModelConfig struct {
	// name
	Name string `json:"name"`
	// version
	Version string `json:"version"`
	// max token usage
	MaxTokenUsage int `json:"maxTokenUsage"`
	// max waiting time for model response
	MaxWaitingTime int `json:"maxWaitingTime"`
	// base url
	BaseUrl string `json:"baseUrl"`
	// app api key, no secret
	APIKey APIKey `json:"apiKey"`
}
