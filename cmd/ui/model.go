package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// colors
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	responseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			MarginLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)
)

type model struct {
	input    textinput.Model
	messages []string
	agent    AgentUI
	quitting bool
}

type AgentUI interface {
	SendQuestion(query string, responseChan chan string, hintChan chan string) error
	Stop() error
}

func NewModel(agent AgentUI) model {
	ti := textinput.New()
	ti.Placeholder = "Type /ask your question or /quit to exit..."
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 80
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle

	return model{
		input:    ti,
		messages: []string{},
		agent:    agent,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			return m.handleInput()
		}

	case responseMsg:
		m.messages = append(m.messages, responseStyle.Render(msg.content))
		return m, nil

	case hintMsg:
		m.messages = append(m.messages, infoStyle.Render(msg.content))
		return m, nil

	case errorMsg:
		m.messages = append(m.messages, errorStyle.Render("Error: "+msg.err))
		return m, nil
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) handleInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	m.messages = append(m.messages, promptStyle.Render("> ")+input)
	m.input.SetValue("")

	switch {
	case strings.HasPrefix(input, "/ask "):
		query := strings.TrimPrefix(input, "/ask ")
		return m.sendQuestion(query)
	case input == "/quit":
		m.messages = append(m.messages, infoStyle.Render("Exiting..."))
		m.quitting = true
		return m, tea.Quit
	case input == "/help":
		m.messages = append(m.messages, helpStyle.Render(m.getHelp()))
		return m, nil
	case input == "/clear":
		m.messages = []string{}
		return m, nil
	default:
		m.messages = append(m.messages, errorStyle.Render("Unknown command. Type /help for available commands."))
		return m, nil
	}
}

func (m model) sendQuestion(query string) (tea.Model, tea.Cmd) {
	if m.agent == nil {
		m.messages = append(m.messages, errorStyle.Render("Agent not initialized"))
		return m, nil
	}

	responseChan := make(chan string, 1)
	hintChan := make(chan string, 1)

	go func() {
		err := m.agent.SendQuestion(query, responseChan, hintChan)
		if err != nil {
			responseChan <- fmt.Sprintf("Error: %v", err)
		}
	}()

	return m, tea.Batch(
		waitForResponse(responseChan),
		waitForHint(hintChan),
	)
}

func waitForResponse(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return responseMsg{content: <-ch}
	}
}

func waitForHint(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return hintMsg{content: <-ch}
	}
}

type responseMsg struct {
	content string
}

type hintMsg struct {
	content string
}

type errorMsg struct {
	err string
}

func (m model) getHelp() string {
	return `Available commands:
  /ask [question]    Send a question to the agent
  /quit              Exit the application
  /clear             Clear the screen
  /help              Show this help message`
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Agent CLI"))
	b.WriteString("\n")

	for _, msg := range m.messages {
		b.WriteString(msg)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.input.View())

	return b.String()
}
