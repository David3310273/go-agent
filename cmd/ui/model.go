package ui

import (
	"strings"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/cmd/app"
	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uuid "github.com/gofrs/uuid/v5"
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

	promptUserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			Bold(true).
			Align(lipgloss.Left)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	responseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	streamStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true)
)

type messageEntry struct {
	content string
	msgType string // "response", "hint", "error", "prompt", "info", "stream"
}

type UIPage struct {
	input         textinput.Model
	messages      []messageEntry
	app           *app.SimpleCommandApp
	sessionID     string // hidden session ID, not displayed in UI
	quitting      bool
	waiting       bool   // true when waiting for command output, input is locked
	streaming     bool   // true when stream mode
	streamContent string // accumulated stream content for response display
	streamTime    string // timestamp of first chunk
	// output stream in command page
	responseChan  chan core.Answer // response channel for stream mode
	hintChan      chan core.Answer // hint channel for stream mode
	currentHint   string           // current accumulated hint content
	hintOverwrite bool             // next hint chunk should overwrite current hint
	helpPage      *helpPage
	width         int // terminal width for text wrapping
	height        int // terminal height for viewport
	welcomeHeight int
	lastError     string // last error message, displayed above input area
	viewport      viewport.Model
	autoScroll    bool   // true when user hasn't manually scrolled, auto-scroll to bottom
	lastMsgCount  int    // track message count to detect new messages
	lastContent   string // track viewport content to avoid resetting on every View()
}

type helpPage struct {
	viewport viewport.Model
	active   bool
}

func NewModel(app *app.SimpleCommandApp, sessionID string) UIPage {
	ti := textinput.New()
	ti.Placeholder = "Type /ask your question or /quit to exit..."
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 50
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle

	return UIPage{
		input:      ti,
		messages:   []messageEntry{},
		app:        app,
		sessionID:  sessionID,
		autoScroll: true,
	}
}

func (m UIPage) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.ClearScreen,
	)
}

func (m *UIPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.helpPage != nil && m.helpPage.active {
		return m.updateHelpPage(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width
		vpHeight := m.calcViewportHeight()
		if m.viewport.Width == 0 {
			m.viewport = viewport.New(msg.Width, vpHeight)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		return m, nil
	case tea.KeyMsg:
		core.LogStd(core.LogLevelDebug, "UI KeyMsg: Type=%v String=%q", msg.Type, msg.String())
		// let viewport handle navigation keys first
		switch msg.Type {
		case tea.KeyCtrlC:
			diags := core.StopCommandApp(m.app)
			if len(diags) > 0 {
				core.LogStd(core.LogLevelError, "UI: error stopping command app: %s", diags.ToString())
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.waiting {
				return m, nil
			}

			diags := core.StopCommandApp(m.app)
			if len(diags) > 0 {
				core.LogStd(core.LogLevelError, "UI: error stopping command app: %s", diags.ToString())
				return m, nil
			}

			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.waiting {
				return m, nil
			}
			return m.handleInput()
		case tea.KeyUp, tea.KeyDown:
			core.LogStd(core.LogLevelDebug, "UI: routing arrow/page key to viewport: Type=%v", msg.Type)
			m.autoScroll = false
			core.LogStd(core.LogLevelDebug, "UI: before viewport.Update: YOffset=%d TotalLines=%d Height=%d", m.viewport.YOffset, m.viewport.TotalLineCount(), m.viewport.Height)
			m.viewport, cmd = m.viewport.Update(msg)
			core.LogStd(core.LogLevelDebug, "UI: after viewport.Update: YOffset=%d", m.viewport.YOffset)
			return m, cmd
		case tea.KeyLeft, tea.KeyRight:
			// when input has content, let textinput handle cursor movement;
			// otherwise pass to viewport for horizontal scrolling
			if m.input.Value() != "" {
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			m.autoScroll = false
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		// fallback: also check String() for keys that might not have Type set correctly
		switch msg.String() {
		case "up", "down", "pgup", "pgdown":
			core.LogStd(core.LogLevelDebug, "UI: routing String() key to viewport: %s", msg.String())
			m.autoScroll = false
			core.LogStd(core.LogLevelDebug, "UI: before viewport.Update: YOffset=%d TotalLines=%d Height=%d", m.viewport.YOffset, m.viewport.TotalLineCount(), m.viewport.Height)
			m.viewport, cmd = m.viewport.Update(msg)
			core.LogStd(core.LogLevelDebug, "UI: after viewport.Update: YOffset=%d", m.viewport.YOffset)
			return m, cmd
		case "left", "right":
			if m.input.Value() != "" {
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			m.autoScroll = false
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case outputMsg:
		if msg.msgType == "response" {
			if m.streaming {
				// clear hint once response starts coming in
				if m.currentHint != "" {
					m.currentHint = ""
					m.hintOverwrite = false
				}
				if msg.isComplete {
					// stream finished: use complete content directly (don't accumulate delta again)
					finalContent := msg.content
					if finalContent != "" {
						// use recorded first chunk time or current time
						timePrefix := m.streamTime
						if timePrefix == "" {
							timePrefix = formatTimestamp()
						}
						m.messages = append(m.messages, messageEntry{content: timePrefix + " " + finalContent, msgType: "stream"})
					}
					m.streamContent = ""
					m.streamTime = ""
					m.waiting = false
					m.streaming = false
					m.input.SetValue("")
					m.input.Focus()
				} else {
					// continue streaming: accumulate delta content for real-time display
					// record time only for first chunk
					if m.streamContent == "" && m.streamTime == "" {
						m.streamTime = formatTimestamp()
					}
					m.streamContent += msg.content
					// continue listening for next chunk
					m.refreshViewport()
					return m, waitForStreamOutput([]core.OutputChannel{
						{Chan: m.responseChan, Type: "response"},
						{Chan: m.hintChan, Type: "hint"},
					})
				}
			} else {
				// non-stream mode: show message directly
				if msg.content != "" {
					m.messages = append(m.messages, messageEntry{content: formatTimestamp() + " " + msg.content, msgType: "stream"})
				}
				m.waiting = false
				m.input.SetValue("")
				m.input.Focus()
			}
		} else if msg.msgType == "hint" {
			if m.streaming {
				// accumulate hint content in one line
				core.LogStd(core.LogLevelDebug, "UI received hint chunk: %s", msg.content)
				// if previous chunk contained a period, overwrite current hint
				if m.hintOverwrite {
					m.currentHint = msg.content
					m.hintOverwrite = false
				} else {
					m.currentHint += msg.content
				}
				// check if current chunk contains a period, mark for next overwrite
				if strings.Contains(msg.content, ".") {
					m.hintOverwrite = true
				}
				// truncate with ellipsis if exceeds 80 characters
				displayHint := m.currentHint
				if len(displayHint) > 80 {
					displayHint = displayHint[:77] + "..."
				}
				m.currentHint = displayHint
				// continue listening for more chunks
				m.refreshViewport()
				return m, waitForStreamOutput([]core.OutputChannel{
					{Chan: m.responseChan, Type: "response"},
					{Chan: m.hintChan, Type: "hint"},
				})
			} else {
				// non-stream mode: show as normal hint
				m.messages = append(m.messages, messageEntry{content: msg.content, msgType: "hint"})
			}
		}
		m.refreshViewport()
		return m, nil

	case errorMsg:
		m.lastError = msg.err
		m.refreshViewport()
		return m, nil
	}

	m.input, cmd = m.input.Update(msg)

	// clear error message only when user types actual characters (not cursor movement)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if len(keyMsg.String()) == 1 {
			m.lastError = ""
		}
		m.refreshViewport()
	}

	return m, cmd
}

func (m *UIPage) updateHelpPage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter || msg.String() == "q" || msg.Type == tea.KeyEsc {
			m.helpPage.active = false
			m.input.Focus()
			return m, nil
		}
		// handle navigation keys manually for better compatibility
		switch msg.String() {
		case "up", "k":
			m.helpPage.viewport.ScrollUp(1)
		case "down", "j":
			m.helpPage.viewport.ScrollDown(1)
		}
		// ensure YOffset doesn't go negative
		if m.helpPage.viewport.YOffset < 0 {
			m.helpPage.viewport.YOffset = 0
		}
		return m, nil
	case tea.WindowSizeMsg:
		// calculate help content header/footer height
		helpHeader := titleStyle.Render("Help Documentation") + "\n\n"
		helpFooter := "\n\n" + hintStyle.Render("Press q or Esc to exit help")
		helpVpHeight := msg.Height - lipgloss.Height(helpHeader) - lipgloss.Height(helpFooter)
		if helpVpHeight < 5 {
			helpVpHeight = 5
		}
		m.helpPage.viewport.Width = msg.Width
		m.helpPage.viewport.Height = helpVpHeight
		m.helpPage.viewport.SetContent(m.app.GetHelpDoc())
		return m, nil
	}

	var cmd tea.Cmd
	m.helpPage.viewport, cmd = m.helpPage.viewport.Update(msg)
	return m, cmd
}

func (m *UIPage) handleInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	core.LogStd(core.LogLevelDebug, "Parsing user input: %s", input)

	parts := strings.Fields(input)
	cmdName := parts[0]

	switch cmdName {
	case "/quit", "/exit", "/q":
		diags := core.StopCommandApp(m.app)
		if len(diags) > 0 {
			core.LogStd(core.LogLevelError, "UI: error running /quit: %v\n", diags.ToString())
		}
		m.quitting = true
		return m, tea.Quit
	case "/help", "/h":
		// calculate help content header/footer height
		helpHeader := titleStyle.Render("Help Documentation") + "\n\n"
		helpFooter := "\n\n" + hintStyle.Render("Press q or Esc to exit help")
		helpVpHeight := m.height - lipgloss.Height(helpHeader) - lipgloss.Height(helpFooter)
		if helpVpHeight < 5 {
			helpVpHeight = 5
		}
		m.helpPage = &helpPage{
			viewport: viewport.New(m.width, helpVpHeight),
			active:   true,
		}
		m.helpPage.viewport.SetContent(m.app.GetHelpDoc())
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	case "/new":
		id, _ := uuid.NewV4()
		m.sessionID = id.String()
		m.messages = []messageEntry{}
		m.refreshViewport()
		m.input.SetValue("")
		return m, nil
	}

	command, diagnostics := m.app.Parse(input)
	if len(diagnostics) > 0 {
		var messages []string
		for _, diag := range diagnostics {
			if diag.Message != "" {
				messages = append(messages, diag.Message)
			}
		}
		if len(messages) > 0 {
			m.lastError = strings.Join(messages, "\n")
		} else {
			m.lastError = "Unknown error occurred"
		}
		m.input.SetValue("")
		return m, nil
	}

	if command == nil {
		m.lastError = "Unknown command."
		m.input.SetValue("")
		return m, nil
	}

	m.input.SetValue("")

	command = command.WithOption("sessionID", m.sessionID)

	m.lastError = ""

	switch command.GetBase() {
	case "/ask":
		askCmd, ok := command.(*model.AskQuestionCommand)
		if ok {
			m.messages = append(m.messages, messageEntry{content: formatTimestamp() + " " + askCmd.Query, msgType: "prompt"})
			m.refreshViewport()
			m.streaming = askCmd.Stream
			m.streamContent = ""
			return m.runAskCommand(askCmd)
		}
		m.lastError = "Unknown command."
		return m, nil
	default:
		m.lastError = "Unknown command."
		return m, nil
	}
}

func (m UIPage) runAskCommand(command *model.AskQuestionCommand) (tea.Model, tea.Cmd) {
	if m.app.Agent == nil {
		m.lastError = "Agent not initialized"
		return &m, nil
	}

	diag := command.Run(m.app)
	if diag != nil {
		m.lastError = diag.Message
		return &m, nil
	}

	outputChans := command.GetOutputChans()
	if len(outputChans) == 0 {
		return &m, nil
	}

	m.waiting = true
	m.input.Blur()

	if m.streaming {
		// save channels for continued listening
		m.responseChan = outputChans[0].Chan
		m.hintChan = outputChans[1].Chan
		// stream mode: listen to both channels continuously
		return &m, waitForStreamOutput(outputChans)
	}

	// non-stream mode: just wait for the first response
	var cmds []tea.Cmd
	for _, oc := range outputChans {
		cmds = append(cmds, waitForOutput(oc.Chan, oc.Type))
	}

	return &m, tea.Batch(cmds...)
}

// func (m UIPage) runCommand(command core.Command) (tea.Model, tea.Cmd) {
// 	if m.app.Agent == nil {
// 		m.lastError = "Agent not initialized"
// 		return m, nil
// 	}

// 	diag := command.Run(m.app)
// 	if diag != nil {
// 		m.lastError = diag.Message
// 		return m, nil
// 	}

// 	outputChans := command.GetOutputChans()
// 	if len(outputChans) == 0 {
// 		return m, nil
// 	}

// 	m.waiting = true
// 	m.input.Blur()

// 	var cmds []tea.Cmd
// 	for _, oc := range outputChans {
// 		cmds = append(cmds, waitForOutput(oc.Chan, oc.Type))
// 	}

// 	return m, tea.Batch(cmds...)
// }

func waitForStreamOutput(outputChans []core.OutputChannel) tea.Cmd {
	return func() tea.Msg {
		// listen to both channels
		responseChan := outputChans[0].Chan
		hintChan := outputChans[1].Chan

		select {
		case answer, ok := <-responseChan:
			if !ok {
				// channel closed, stream is complete
				core.LogStd(core.LogLevelDebug, "UI: responseChan closed, stream complete")
				return outputMsg{content: "", msgType: "response", isComplete: true, isFinalAnswer: true}
			}
			// debug: log the actual type received
			core.LogStd(core.LogLevelDebug, "UI: received response chunk, type=%T", answer)

			// try SimpleNormalResponse first
			if simpleResp, ok := answer.(simple.SimpleNormalResponse); ok {
				content := simpleResp.Response.Response
				if content != "" {
					core.LogStd(core.LogLevelDebug, "UI: extracted content from SimpleNormalResponse.Response: %s", content[:min(len(content), 50)])
					return outputMsg{content: content, msgType: "response", isComplete: true, isFinalAnswer: true}
				}
				// fallback to Choices content
				if len(simpleResp.Response.Choices) > 0 && simpleResp.Response.Choices[0].Message != nil {
					content = simpleResp.Response.Choices[0].Message.Content
					isComplete := simpleResp.Response.Choices[0].FinishReason != ""
					core.LogStd(core.LogLevelDebug, "UI: extracted content from SimpleNormalResponse.Choices, isComplete=%v", isComplete)
					return outputMsg{content: content, msgType: "response", isComplete: isComplete, isFinalAnswer: isComplete}
				}
			}

			// try SimpleToolConfirmResponse
			if confirmResp, ok := answer.(simple.SimpleToolConfirmResponse); ok {
				content := confirmResp.Response.Response
				if content != "" {
					core.LogStd(core.LogLevelDebug, "UI: extracted content from SimpleToolConfirmResponse.Response: %s", content[:min(len(content), 50)])
					return outputMsg{content: content, msgType: "response", isComplete: true, isFinalAnswer: true}
				}
				// fallback to Choices content
				if len(confirmResp.Response.Choices) > 0 && confirmResp.Response.Choices[0].Message != nil {
					content = confirmResp.Response.Choices[0].Message.Content
					core.LogStd(core.LogLevelDebug, "UI: extracted content from SimpleToolConfirmResponse.Choices")
					return outputMsg{content: content, msgType: "response", isComplete: true, isFinalAnswer: true}
				}
			}

			// try core.AgentResponse
			if resp, ok := answer.(core.AgentResponse); ok {
				// extract content from choices
				content := ""
				isComplete := false
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
					// skip tool call responses, don't show to user
					if resp.Choices[0].Message.ToolCalls != nil && len(*resp.Choices[0].Message.ToolCalls) > 0 {
						core.LogStd(core.LogLevelDebug, "UI: skipping tool call response")
						return outputMsg{content: "", msgType: "response", isComplete: false, isFinalAnswer: false}
					}
					content = resp.Choices[0].Message.Content
					// check if this is the final chunk
					if resp.Choices[0].FinishReason != "" {
						isComplete = true
						core.LogStd(core.LogLevelDebug, "UI: response chunk isComplete=true, finishReason=%s", resp.Choices[0].FinishReason)
					}
				}
				// if content is empty but has usage, this is the final chunk
				if content == "" && resp.Usage.TotalTokens > 0 {
					isComplete = true
					core.LogStd(core.LogLevelDebug, "UI: response chunk isComplete=true (usage only)")
				}
				// if has final response text, use it instead of stream content
				if resp.Response != "" {
					content = resp.Response
					isComplete = true
				}
				return outputMsg{content: content, msgType: "response", isComplete: isComplete, isFinalAnswer: isComplete}
			}

			// for other types, treat as complete
			core.LogStd(core.LogLevelDebug, "UI: response chunk type assertion failed, using ToString()")
			return outputMsg{content: answer.ToString(), msgType: "response", isComplete: true, isFinalAnswer: true}
		case hint, ok := <-hintChan:
			if !ok {
				// hint channel closed, continue waiting for response
				return outputMsg{content: "", msgType: "hint", isComplete: false}
			}
			// try SimpleNormalResponse for hint
			if simpleResp, ok := hint.(simple.SimpleNormalResponse); ok {
				if len(simpleResp.Response.Choices) > 0 && simpleResp.Response.Choices[0].Message != nil && simpleResp.Response.Choices[0].Message.ReasoningContent != nil {
					hintContent := *simpleResp.Response.Choices[0].Message.ReasoningContent
					time.Sleep(500 * time.Millisecond)
					return outputMsg{content: hintContent, msgType: "hint", isComplete: false}
				}
			}
			// try core.AgentResponse for hint
			if resp, ok := hint.(core.AgentResponse); ok {
				if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && resp.Choices[0].Message.ReasoningContent != nil {
					hintContent := *resp.Choices[0].Message.ReasoningContent
					// add delay for hint display to avoid flashing too fast
					time.Sleep(500 * time.Millisecond)
					return outputMsg{content: hintContent, msgType: "hint", isComplete: false}
				}
			}
			// add delay for hint display
			time.Sleep(500 * time.Millisecond)
			return outputMsg{content: hint.ToString(), msgType: "hint", isComplete: false}
		}
	}
}

func waitForOutput(ch chan core.Answer, msgType string) tea.Cmd {
	return func() tea.Msg {
		answer := <-ch
		// try SimpleNormalResponse first
		if simpleResp, ok := answer.(simple.SimpleNormalResponse); ok {
			// use final response content if available
			if simpleResp.Response.Response != "" {
				return outputMsg{content: simpleResp.Response.Response, msgType: msgType, isFinalAnswer: true}
			}
			// fallback to choices content
			if len(simpleResp.Response.Choices) > 0 && simpleResp.Response.Choices[0].Message != nil {
				return outputMsg{content: simpleResp.Response.Choices[0].Message.Content, msgType: msgType}
			}
		}
		// try SimpleToolConfirmResponse
		if confirmResp, ok := answer.(simple.SimpleToolConfirmResponse); ok {
			// use final response content if available
			if confirmResp.Response.Response != "" {
				return outputMsg{content: confirmResp.Response.Response, msgType: msgType, isFinalAnswer: true}
			}
			// fallback to choices content
			if len(confirmResp.Response.Choices) > 0 && confirmResp.Response.Choices[0].Message != nil {
				return outputMsg{content: confirmResp.Response.Choices[0].Message.Content, msgType: msgType}
			}
		}
		// try AgentResponse
		if resp, ok := answer.(core.AgentResponse); ok {
			// use final response content if available
			if resp.Response != "" {
				return outputMsg{content: resp.Response, msgType: msgType, isFinalAnswer: true}
			}
			// fallback to choices content
			if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
				// skip tool call responses, don't show to user
				if resp.Choices[0].Message.ToolCalls != nil && len(*resp.Choices[0].Message.ToolCalls) > 0 {
					core.LogStd(core.LogLevelDebug, "UI: skipping tool call response in non-stream mode")
					return outputMsg{content: "", msgType: msgType, isFinalAnswer: false}
				}
				return outputMsg{content: resp.Choices[0].Message.Content, msgType: msgType}
			}
		}
		// for other types, use ToString
		return outputMsg{content: answer.ToString(), msgType: msgType}
	}
}

type outputMsg struct {
	content       string
	msgType       string
	isComplete    bool // true when stream is complete
	isFinalAnswer bool // true when this is the final parsed response
}

type errorMsg struct {
	err string
}

func (m UIPage) buildViewportContent() string {
	wrapWidth := m.width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	var contentBuilder strings.Builder
	for i, entry := range m.messages {
		switch entry.msgType {
		case "response":
			wrapped := wrapText(entry.content, wrapWidth)
			contentBuilder.WriteString(responseStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "stream":
			wrapped := wrapText(entry.content, wrapWidth)
			contentBuilder.WriteString(streamStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "hint":
			truncated := truncateHint(entry.content, wrapWidth)
			contentBuilder.WriteString(infoStyle.Render("💡 " + truncated))
			contentBuilder.WriteString("\n")
			contentBuilder.WriteString("\n")
		case "error":
			wrapped := wrapText(entry.content, wrapWidth)
			contentBuilder.WriteString(errorStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "prompt":
			if i > 0 {
				contentBuilder.WriteString("\n")
			}
			wrapped := wrapText(entry.content, wrapWidth)
			contentBuilder.WriteString(promptUserStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		case "info":
			wrapped := wrapText(entry.content, wrapWidth)
			contentBuilder.WriteString(helpStyle.Render(wrapped))
			contentBuilder.WriteString("\n")
		}
	}

	if m.waiting {
		if m.streaming {
			if m.currentHint != "" && m.streamContent == "" {
				contentBuilder.WriteString(infoStyle.Render("💡 " + m.currentHint))
				contentBuilder.WriteString("\n")
				contentBuilder.WriteString("\n")
			}
			if m.streamContent != "" {
				timePrefix := m.streamTime
				if timePrefix == "" {
					timePrefix = formatTimestamp()
				}
				wrapped := wrapText(timePrefix+" "+m.streamContent, wrapWidth)
				contentBuilder.WriteString(streamStyle.Render(wrapped))
				contentBuilder.WriteString("\n")
			}
		} else {
			contentBuilder.WriteString(infoStyle.Render("⏳ Waiting for response..."))
		}
	}

	return contentBuilder.String()
}

func (m UIPage) calcViewportHeight() int {
	wrapWidth := m.width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	header := titleStyle.Render("Agent CLI") + "\n" + titleStyle.Render(m.app.Welcome())
	headerHeight := lipgloss.Height(header)

	var footerBuilder strings.Builder
	if m.lastError != "" {
		wrapped := wrapText(m.lastError, wrapWidth)
		footerBuilder.WriteString(errorStyle.Render(wrapped))
		footerBuilder.WriteString("\n")
	}
	footerBuilder.WriteString(m.input.View())
	if hint := m.getCommandHint(); hint != "" {
		footerBuilder.WriteString("\n")
		footerBuilder.WriteString(hintStyle.Render(hint))
	}
	footerHeight := lipgloss.Height(footerBuilder.String())

	vpHeight := m.height - headerHeight - footerHeight
	if vpHeight < 5 {
		vpHeight = 5
	}
	return vpHeight
}

func (m *UIPage) refreshViewport() {
	vpHeight := m.calcViewportHeight()
	if vpHeight != m.viewport.Height {
		m.viewport.Height = vpHeight
	}
	newContent := m.buildViewportContent()
	if newContent != m.lastContent {
		m.viewport.SetContent(newContent)
		m.lastContent = newContent
		if len(m.messages) > m.lastMsgCount {
			m.lastMsgCount = len(m.messages)
		}
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
	}
}

func (m *UIPage) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.helpPage != nil && m.helpPage.active {
		return m.viewHelpPage()
	}

	wrapWidth := m.width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	header := titleStyle.Render("Agent CLI") + "\n" + titleStyle.Render(m.app.Welcome())
	welcomeHeaderHeight := lipgloss.Height(header)
	m.welcomeHeight = welcomeHeaderHeight

	var footerBuilder strings.Builder
	if m.lastError != "" {
		wrapped := wrapText(m.lastError, wrapWidth)
		footerBuilder.WriteString(errorStyle.Render(wrapped))
		footerBuilder.WriteString("\n")
	}
	footerBuilder.WriteString(m.input.View())
	if hint := m.getCommandHint(); hint != "" {
		footerBuilder.WriteString("\n")
		footerBuilder.WriteString(hintStyle.Render(hint))
	}
	footer := footerBuilder.String()

	return lipgloss.JoinVertical(lipgloss.Left, header, m.viewport.View(), footer)
}

func (m UIPage) viewHelpPage() string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(titleStyle.Render("Help Documentation"))
	contentBuilder.WriteString("\n\n")
	contentBuilder.WriteString(m.helpPage.viewport.View())
	contentBuilder.WriteString("\n\n")
	contentBuilder.WriteString(hintStyle.Render("Press q or Esc to exit help"))

	return contentBuilder.String()
}

func (m UIPage) getCommandHint() string {
	input := strings.TrimSpace(m.input.Value())
	if input == "" || !strings.HasPrefix(input, "/") {
		return ""
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	cmdName := parts[0]
	if cmd, ok := m.app.Commands[cmdName]; ok {
		help := cmd.GetHelp()
		if format, ok := help["format"].(string); ok {
			return "Format: " + format
		}
	}

	return ""
}

func truncateHint(content string, maxWidth int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 1 {
		content = strings.Join(lines, " ")
	}
	if len(content) > maxWidth {
		return content[:maxWidth-3] + "..."
	}
	return content
}

// formatTimestamp returns current time in [YYYY-MM-DD HH:MM:SS] format
func formatTimestamp() string {
	now := time.Now()
	return now.Format("[2006-01-02 15:04:05]")
}

// wrapText wraps text to the specified width, preserving existing newlines
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrapped []string

	for _, line := range lines {
		if len(line) <= width {
			wrapped = append(wrapped, line)
			continue
		}

		// split long line into chunks
		words := strings.Fields(line)
		if len(words) == 0 {
			wrapped = append(wrapped, line)
			continue
		}

		var currentLine string
		for _, word := range words {
			if len(currentLine)+len(word)+1 <= width {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				if currentLine != "" {
					wrapped = append(wrapped, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			wrapped = append(wrapped, currentLine)
		}
	}

	return strings.Join(wrapped, "\n")
}
