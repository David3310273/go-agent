package view

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	PromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	PromptUserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			Bold(true).
			Align(lipgloss.Left)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	ResponseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#32FF7E"))

	StreamStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#32FF7E"))

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	PendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	HintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true)
)
