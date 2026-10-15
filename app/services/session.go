package services

import (
	"fmt"
	"os"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/agent/simple/utils"
	"github.com/David3310273/go-agent/core"
)

type SessionHistoryList struct {
	SessionID string `json:"sessionID"`
	Title     string `json:"title"`
}

func GetSessionHistory(agent *simple.SimpleAgent, sessionID string, size int) ([]byte, *core.Diagnostic) {
	session := agent.GetSession(sessionID)
	if session == nil {
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: fmt.Sprintf("session %s not found", sessionID),
		}
	}

	filePathFormat := utils.ResolvePath(session.GetConfigs().RootPath, session.GetConfigs().ChatHistoryFilePathFormat)
	filename := fmt.Sprintf(filePathFormat, sessionID)
	// no history file, return error
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, &core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: fmt.Sprintf("session %s not found", sessionID),
		}
	}

	messages, err := session.ExportChatHistory(size)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func GetSessionList(agent *simple.SimpleAgent) []SessionHistoryList {
	sessions := agent.ListSessions()
	result := make([]SessionHistoryList, 0, len(sessions))

	for _, session := range sessions {
		if session == nil {
			continue
		}
		messages := session.GetConversation()
		firstUserMessage := ""
		for _, message := range *messages {
			if message.Role == core.RoleUser {
				firstUserMessage = message.Content
				break
			}
		}
		result = append(result, SessionHistoryList{
			SessionID: session.GetID(),
			Title:     firstUserMessage,
		})
	}

	return result
}

type SessionMessage struct {
	Role      core.RoleType `json:"role"`
	Content   string        `json:"content"`
	Timestamp int64         `json:"timestamp"`
}

type SessionHistory struct {
	SessionID string           `json:"sessionID"`
	Messages  []SessionMessage `json:"messages"`
}
