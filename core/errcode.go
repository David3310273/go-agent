package core

import json "encoding/json"

type Serializable interface {
	ToString() string
}

type Diagnostic struct {
	Code    MessageCode `json:"code"`
	Level   Severity    `json:"level"`
	Message string      `json:"message"`
	Data    string      `json:"data"`
}

func (d Diagnostic) ToString() string {
	response, err := json.Marshal(d)
	if err != nil {
		return d.Message
	}

	return string(response)
}

type DiagnosticList []Diagnostic

func (d DiagnosticList) ToString() string {
	response, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(response)
}

type Severity int

const (
	SeverityInfo  Severity = 0
	SeverityWarn  Severity = 1
	SeverityError Severity = 2
)

type MessageCode int

const (
	// system code
	MessageCodeSuccess     MessageCode = 0
	MessageCodeSystemError MessageCode = 1
	MessageSystemClosed    MessageCode = 2
	// 1xx: config error
	MessageCodeConfigFileNotFound    MessageCode = 100
	MessageCodeConfigFileFormatError MessageCode = 101
	MessageCodePromptLoadFailed      MessageCode = 302
	// 2xx: event IO error
	EventHandlerAlreadyRegistered  MessageCode = 200
	MessageCodeEventSourceNotFound MessageCode = 201
	MessageCodeEventStorageError   MessageCode = 202
	MessageCodeEventEmitTimeOut    MessageCode = 203
	// 3xx: provider error
	MessageCodeProviderCreateError MessageCode = 300
	// MessageCodeEmptyProviderError    MessageCode = 301
	MessageCodeProviderAuthError   MessageCode = 302
	MessageCodeNoAvailableProvider MessageCode = 303
	MessageCodeProviderNotFound    MessageCode = 304
	// 4xx: lock error
	MessageCodeLockError  MessageCode = 400
	MessageCodeLockFailed MessageCode = 401
	// 5xx: session error
	MessageCodeWriteHistoryError  MessageCode = 500
	MessageCodeSessionNotFound    MessageCode = 501
	MessageCodeSessionCreateError MessageCode = 502
	MessageCodeSessionStopError   MessageCode = 503
	// 6xx: tool error
	MessageCodeToolNotFound      MessageCode = 600
	MessageCodeToolRunError      MessageCode = 601
	MessageCodeToolValidateError MessageCode = 602
	// 7xx: agent core error
	MessageCodeAgentCoreConfigError   MessageCode = 700
	MessageCodeInvalidResponseFromLLM MessageCode = 701
	MessageCodeErrorFromLLM           MessageCode = 702
	// 8xx: kb error
	MessageCodeDocNotFound MessageCode = 800
	// 9xx: storage error
	// auto-added: storage error codes.
	MessageCodeStorageError     MessageCode = 900
	MessageCodeDatabaseNotFound MessageCode = 901
	// 10xx: embedder error
	// auto-added: embedder error codes.
	MessageCodeEmbedderConfigError MessageCode = 1000
)
