package view

import (
	"strings"

	"github.com/David3310273/go-agent/core"
	"github.com/charmbracelet/lipgloss"
)

type MessageEntry struct {
	Content string
	MsgType string // "response", "hint", "error", "prompt", "info", "stream"
}

type RMStatusEntry struct {
	Message string
	Done    bool
	Code    core.MessageCode
}

func BuildViewportContent(messages []MessageEntry, waiting bool, streaming bool, currentHint string, streamContent string, streamTime string, width int) string {
	wrapWidth := width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	var contentBuilder strings.Builder
	for i, entry := range messages {
		switch entry.MsgType {
		case "response":
			wrapped := WrapText(entry.Content, wrapWidth)
			contentBuilder.WriteString(ResponseStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "stream":
			wrapped := WrapText(entry.Content, wrapWidth)
			contentBuilder.WriteString(StreamStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "hint":
			truncated := TruncateHint(entry.Content, wrapWidth)
			contentBuilder.WriteString(InfoStyle.Render("💡 " + truncated))
			contentBuilder.WriteString("\n")
			contentBuilder.WriteString("\n")
		case "error":
			wrapped := WrapText(entry.Content, wrapWidth)
			contentBuilder.WriteString(ErrorStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "prompt":
			if i > 0 {
				contentBuilder.WriteString("\n")
			}
			wrapped := WrapText(entry.Content, wrapWidth)
			contentBuilder.WriteString(PromptUserStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "info":
			wrapped := WrapText(entry.Content, wrapWidth)
			contentBuilder.WriteString(HelpStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		}
	}

	if waiting {
		if streaming {
			if currentHint != "" && streamContent == "" {
				contentBuilder.WriteString(InfoStyle.Render("💡 " + currentHint))
				contentBuilder.WriteString("\n")
				contentBuilder.WriteString("\n")
			}
			if streamContent != "" {
				timePrefix := streamTime
				if timePrefix == "" {
					timePrefix = FormatTimestamp()
				}
				wrapped := WrapText(timePrefix+" "+streamContent, wrapWidth)
				contentBuilder.WriteString(StreamStyle.Render(wrapped))
				contentBuilder.WriteString("\n")
			}
		} else {
			contentBuilder.WriteString(InfoStyle.Render("⏳ Waiting for response..."))
		}
	}

	return contentBuilder.String()
}

func BuildFooter(lastError string, rmStatuses []RMStatusEntry, inputView string, commandHint string, width int) string {
	wrapWidth := width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	var footerBuilder strings.Builder
	if lastError != "" {
		wrapped := WrapText(lastError, wrapWidth)
		footerBuilder.WriteString(ErrorStyle.Render(wrapped))
		footerBuilder.WriteString("\n")
	}
	for _, entry := range rmStatuses {
		wrapped := WrapText(entry.Message, wrapWidth)
		if !entry.Done {
			footerBuilder.WriteString(PendingStyle.Render(wrapped))
		} else if entry.Code == core.MessageCodeSuccess {
			footerBuilder.WriteString(SuccessStyle.Render(wrapped))
		} else {
			footerBuilder.WriteString(ErrorStyle.Render(wrapped))
		}
		footerBuilder.WriteString("\n")
	}
	footerBuilder.WriteString(inputView)
	if commandHint != "" {
		footerBuilder.WriteString("\n")
		footerBuilder.WriteString(HintStyle.Render(commandHint))
	}
	return footerBuilder.String()
}

func CalcViewportHeight(height int, header string, footer string) int {
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	vpHeight := height - headerHeight - footerHeight
	if vpHeight < 5 {
		vpHeight = 5
	}
	return vpHeight
}

func BuildHeader(welcomeMsg string) string {
	return TitleStyle.Render("Agent CLI") + "\n" + TitleStyle.Render(welcomeMsg)
}
