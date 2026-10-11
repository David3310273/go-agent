package ui

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/cmd/app"
	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/cmd/ui/view"
	"github.com/David3310273/go-agent/core"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// UIPage holds all UI state that handlers can modify
type UIPage struct {
	MainPage     *view.MainPage
	App          *app.SimpleCommandApp
	SessionID    string
	Quitting     bool
	ResponseChan chan core.Answer
	HintChan     chan core.Answer
	HelpHandler  *HelpPageHandler
	KBHandler    *KBListPageHandler
	SkillHandler *SkillListPageHandler
	Width        int
	Height       int
	// frontend storage variables
	commandHints []commandHint
	language     core.LanguageType
}

func NewModel(app *app.SimpleCommandApp, sessionID string) *UIPage {
	return &UIPage{
		App:          app,
		SessionID:    sessionID,
		language:     core.Language_EN,
		commandHints: buildCommandHints(),
	}
}

func (m *UIPage) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.ClearScreen,
	)
}

func (m *UIPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.HelpHandler != nil && m.HelpHandler.IsActive() {
		return m.updateHelpPage(msg)
	}

	if m.KBHandler != nil && m.KBHandler.IsActive() {
		return m.updateKBListResultPage(msg)
	}

	if m.SkillHandler != nil && m.SkillHandler.IsActive() {
		return m.updateSkillListResultPage(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.MainPage == nil {
			ti := textinput.New()
			ti.Placeholder = "Type /ask your question or /quit to exit..."
			ti.Focus()
			ti.CharLimit = 1024
			ti.Width = msg.Width
			ti.Prompt = "> "
			ti.PromptStyle = view.PromptStyle
			m.MainPage = view.NewMainPage(msg.Width, msg.Height, m.App.Welcome(), ti)
		} else {
			m.MainPage.Resize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		core.LogStd(core.LogLevelDebug, "UI KeyMsg: Type=%v String=%q", msg.Type, msg.String())
		switch msg.Type {
		case tea.KeyCtrlC:
			diags := core.StopCommandApp(m.App)
			if len(diags) > 0 {
				core.LogStd(core.LogLevelError, "UI: error stopping command app: %s", diags.ToString())
				return m, nil
			}
			m.Quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.MainPage.Waiting {
				return m, nil
			}

			diags := core.StopCommandApp(m.App)
			if len(diags) > 0 {
				core.LogStd(core.LogLevelError, "UI: error stopping command app: %s", diags.ToString())
				return m, nil
			}

			m.Quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.MainPage.Waiting {
				return m, nil
			}
			return m.handleInput()
		case tea.KeyUp, tea.KeyDown:
			core.LogStd(core.LogLevelDebug, "UI: routing arrow/page key to viewport: Type=%v", msg.Type)
			m.MainPage.AutoScroll = false
			core.LogStd(core.LogLevelDebug, "UI: before viewport.Update: YOffset=%d TotalLines=%d Height=%d", m.MainPage.Viewport.YOffset, m.MainPage.Viewport.TotalLineCount(), m.MainPage.Viewport.Height)
			m.MainPage.Viewport, cmd = m.MainPage.Viewport.Update(msg)
			core.LogStd(core.LogLevelDebug, "UI: after viewport.Update: YOffset=%d", m.MainPage.Viewport.YOffset)
			return m, cmd
		case tea.KeyLeft, tea.KeyRight:
			if m.MainPage.Footer.Input.Value() != "" {
				m.MainPage.Footer.Input, cmd = m.MainPage.Footer.Input.Update(msg)
				m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
				return m, cmd
			}
			m.MainPage.AutoScroll = false
			m.MainPage.Viewport, cmd = m.MainPage.Viewport.Update(msg)
			return m, cmd
		}
		// switch msg.String() {
		// case "up", "down":
		// 	core.LogStd(core.LogLevelDebug, "UI: routing String() key to viewport: %s", msg.String())
		// 	m.MainPage.AutoScroll = false
		// 	core.LogStd(core.LogLevelDebug, "UI: before viewport.Update: YOffset=%d TotalLines=%d Height=%d", m.MainPage.Viewport.YOffset, m.MainPage.Viewport.TotalLineCount(), m.MainPage.Viewport.Height)
		// 	m.MainPage.Viewport, cmd = m.MainPage.Viewport.Update(msg)
		// 	core.LogStd(core.LogLevelDebug, "UI: after viewport.Update: YOffset=%d", m.MainPage.Viewport.YOffset)
		// 	return m, cmd
		// case "left", "right":
		// 	if m.MainPage.Footer.Input.Value() != "" {
		// 		m.MainPage.Footer.Input, cmd = m.MainPage.Footer.Input.Update(msg)
		// 		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		// 		return m, cmd
		// 	}
		// 	m.MainPage.AutoScroll = false
		// 	m.MainPage.Viewport, cmd = m.MainPage.Viewport.Update(msg)
		// 	return m, cmd
		// }

	case OutputMsg:
		if msg.MsgType == "response" {
			if m.MainPage.Streaming {
				if m.MainPage.CurrentHint != "" {
					m.MainPage.CurrentHint = ""
					m.MainPage.HintOverwrite = false
				}
				if msg.IsComplete || msg.IsFinalAnswer {
					finalContent := m.MainPage.StreamContent
					if finalContent == "" {
						finalContent = msg.Content
					}
					if finalContent != "" {
						timePrefix := m.MainPage.StreamTime
						if timePrefix == "" {
							timePrefix = view.FormatTimestamp()
						}
						m.MainPage.Messages = append(m.MainPage.Messages, view.MessageEntry{Content: timePrefix + " " + finalContent, MsgType: "stream"})
					}
					m.MainPage.StreamContent = ""
					m.MainPage.StreamTime = ""
					m.MainPage.Waiting = false
					m.MainPage.Streaming = false
					m.MainPage.Footer.Input.SetValue("")
					m.MainPage.Footer.Input.Focus()
					m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
				} else {
					if m.MainPage.StreamContent == "" && m.MainPage.StreamTime == "" {
						m.MainPage.StreamTime = view.FormatTimestamp()
					}
					m.MainPage.StreamContent += msg.Content
					m.refreshMainPage()
					return m, waitForStreamOutput([]core.OutputChannel{
						{Chan: m.ResponseChan, Type: "response"},
						{Chan: m.HintChan, Type: "hint"},
					})
				}
			} else {
				if msg.Content != "" {
					m.MainPage.Messages = append(m.MainPage.Messages, view.MessageEntry{Content: view.FormatTimestamp() + " " + msg.Content, MsgType: "stream"})
				}
				m.MainPage.Waiting = false
				m.MainPage.Footer.Input.SetValue("")
				m.MainPage.Footer.Input.Focus()
				m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
			}
		} else if msg.MsgType == "hint" {
			if m.MainPage.Streaming {
				core.LogStd(core.LogLevelDebug, "UI received hint chunk: %s", msg.Content)
				if m.MainPage.HintOverwrite {
					m.MainPage.CurrentHint = msg.Content
					m.MainPage.HintOverwrite = false
				} else {
					m.MainPage.CurrentHint += msg.Content
				}
				if strings.Contains(msg.Content, ".") {
					m.MainPage.HintOverwrite = true
				}
				displayHint := m.MainPage.CurrentHint
				if len(displayHint) > 80 {
					displayHint = displayHint[:77] + "..."
				}
				m.MainPage.CurrentHint = displayHint
				m.refreshMainPage()
				return m, waitForStreamOutput([]core.OutputChannel{
					{Chan: m.ResponseChan, Type: "response"},
					{Chan: m.HintChan, Type: "hint"},
				})
			} else {
				m.MainPage.Messages = append(m.MainPage.Messages, view.MessageEntry{Content: msg.Content, MsgType: "hint"})
			}
		}
		m.refreshMainPage()
		return m, nil

	case errorMsg:
		m.MainPage.SetLastError(msg.err)
		m.refreshMainPage()
		return m, nil

	case RMResultMsg:
		if msg.Index >= 0 && msg.Index < len(m.MainPage.Footer.RMStatuses) {
			m.MainPage.Footer.RMStatuses[msg.Index].Message = msg.Message
			m.MainPage.Footer.RMStatuses[msg.Index].Done = true
			m.MainPage.Footer.RMStatuses[msg.Index].Code = msg.Code
		}
		m.MainPage.SetRMStatuses(m.MainPage.Footer.RMStatuses)
		m.refreshMainPage()
		return m, nil
	}

	if m.MainPage == nil {
		return m, nil
	}

	m.MainPage.Footer.Input, cmd = m.MainPage.Footer.Input.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if len(keyMsg.String()) == 1 {
			m.MainPage.SetLastError("")
			var pending []view.RMStatusEntry
			for _, entry := range m.MainPage.Footer.RMStatuses {
				if !entry.Done {
					pending = append(pending, entry)
				}
			}
			m.MainPage.Footer.RMStatuses = pending
			m.MainPage.SetLastError("")
			m.MainPage.SetRMStatuses(m.MainPage.Footer.RMStatuses)
		}
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		m.refreshMainPage()
	}

	currentInput := m.MainPage.Footer.Input.Value()
	if currentInput != "" && currentInput != "/" {
		m.MainPage.SetCommandHint(getFormatHint(currentInput, m.commandHints))
	} else {
		m.MainPage.SetCommandHint("")
	}

	return m, cmd
}

func (m *UIPage) updateHelpPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd, shouldExit := m.HelpHandler.Update(msg)
	if shouldExit {
		m.HelpHandler.Deactivate()
		m.MainPage.Footer.Input.Focus()
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		m.MainPage.SetCommandHint("")
		return m, nil
	}
	return m, cmd
}

func (m *UIPage) updateKBListResultPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd, shouldExit := m.KBHandler.Update(msg)
	if shouldExit {
		m.KBHandler.Deactivate()
		m.MainPage.Footer.Input.Focus()
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		m.MainPage.SetCommandHint("")
		return m, nil
	}
	return m, cmd
}

func (m *UIPage) updateSkillListResultPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd, shouldExit := m.SkillHandler.Update(msg)
	if shouldExit {
		m.SkillHandler.Deactivate()
		m.MainPage.Footer.Input.Focus()
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		m.MainPage.SetCommandHint("")
		return m, nil
	}
	return m, cmd
}

func (m *UIPage) handleInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.MainPage.Footer.Input.Value())
	if input == "" {
		return m, nil
	}

	core.LogStd(core.LogLevelDebug, "Parsing user input: %s", input)

	// Try system commands first
	parts := strings.Fields(input)
	if len(parts) > 0 {
		sysCmd := HandleSystemCommand(m, input)
		if sysCmd != nil {
			if m.Quitting {
				return m, tea.Quit
			}
			return m, sysCmd
		}
		// HandleSystemCommand may activate a page handler (e.g., /help) without returning a tea.Cmd
		if (m.HelpHandler != nil && m.HelpHandler.IsActive()) || (m.KBHandler != nil && m.KBHandler.IsActive()) || (m.SkillHandler != nil && m.SkillHandler.IsActive()) {
			return m, nil
		}
	}

	// Parse and handle other commands
	command, diagnostics := m.App.Parse(input)
	if len(diagnostics) > 0 {
		var messages []string
		for _, diag := range diagnostics {
			if diag.Message != "" {
				messages = append(messages, diag.Message)
			}
		}
		var errMsg string
		if len(messages) > 0 {
			errMsg = strings.Join(messages, "\n")
			m.MainPage.SetLastError(errMsg)
		} else {
			errMsg = "Unknown error occurred"
			m.MainPage.SetLastError(errMsg)
		}
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		return m, nil
	}

	if command == nil {
		m.MainPage.SetLastError("Unknown command.")
		m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
		return m, nil
	}

	m.MainPage.Footer.Input.SetValue("")
	m.MainPage.SetFooterInput(m.MainPage.Footer.Input)
	m.MainPage.SetCommandHint("")

	command = command.WithOption("sessionID", m.SessionID)

	m.MainPage.SetLastError("")

	var cmd tea.Cmd
	switch c := command.(type) {
	case *model.AskQuestionCommand:
		cmd = handleAskCommand(m, c)
	case *model.KnowledgeBaseCommand:
		cmd = handleKBCommand(m, c)
	case *model.SkillCommand:
		cmd = handleSkillCommand(m, c)
	default:
		m.MainPage.SetLastError(fmt.Sprintf("%s: command not found", parts[0]))
	}
	return m, cmd
}

func (m *UIPage) View() string {
	if m.HelpHandler != nil && m.HelpHandler.IsActive() {
		return m.HelpHandler.View()
	}

	if m.KBHandler != nil && m.KBHandler.IsActive() {
		return m.KBHandler.View()
	}

	if m.SkillHandler != nil && m.SkillHandler.IsActive() {
		return m.SkillHandler.View()
	}

	if m.MainPage == nil {
		return "Loading..."
	}

	return m.MainPage.View()
}

func (m *UIPage) refreshMainPage() {
	if m.MainPage != nil {
		m.MainPage.SetMessages(m.MainPage.Messages, m.MainPage.Waiting, m.MainPage.Streaming,
			m.MainPage.CurrentHint, m.MainPage.StreamContent, m.MainPage.StreamTime)
		m.MainPage.ScrollToLatest()
	}
}

type errorMsg struct {
	err string
}
