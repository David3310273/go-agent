package view

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type MainPage struct {
	Header       HeaderSection
	Viewport     viewport.Model
	Footer       FooterSection
	AutoScroll   bool
	LastContent  string
	LastMsgCount int
	width        int
	height       int
	// 流式输出状态
	Messages      []MessageEntry
	AllMessages   []MessageEntry
	Waiting       bool
	Streaming     bool
	StreamContent string
	StreamTime    string
	CurrentHint   string
	HintOverwrite bool
}

type HeaderSection struct {
	Title   string
	Welcome string
}

type FooterSection struct {
	LastError   string
	RMStatuses  []RMStatusEntry
	Input       textinput.Model
	CommandHint string
	Width       int
}

func NewMainPage(width, height int, welcomeMsg string, input textinput.Model) *MainPage {
	header := HeaderSection{
		Title:   "Agent CLI",
		Welcome: welcomeMsg,
	}

	footer := FooterSection{
		Input: input,
		Width: width,
	}

	vpHeight := calcMainPageViewportHeight(header, footer, height, width)
	vp := viewport.New(width, vpHeight)

	return &MainPage{
		Header:     header,
		Viewport:   vp,
		Footer:     footer,
		AutoScroll: true,
		width:      width,
		height:     height,
	}
}

func (m *MainPage) SetWelcome(welcomeMsg string) {
	m.Header.Welcome = welcomeMsg
}

func (m *MainPage) SetMessages(messages []MessageEntry, waiting bool, streaming bool, currentHint string, streamContent string, streamTime string) {
	m.Messages = messages
	m.Waiting = waiting
	m.Streaming = streaming
	m.CurrentHint = currentHint
	m.StreamContent = streamContent
	m.StreamTime = streamTime

	content := BuildViewportContent(messages, waiting, streaming, currentHint, streamContent, streamTime, m.Footer.Width)
	if content != m.LastContent {
		m.Viewport.SetContent(content)
		m.LastContent = content
		if len(messages) > m.LastMsgCount {
			m.LastMsgCount = len(messages)
			if m.AutoScroll {
				m.Viewport.GotoBottom()
			}
		}
	}
}

func (m *MainPage) SetLastError(err string) {
	m.Footer.LastError = err
}

func (m *MainPage) SetRMStatuses(statuses []RMStatusEntry) {
	m.Footer.RMStatuses = statuses
}

func (m *MainPage) SetCommandHint(hint string) {
	m.Footer.CommandHint = hint
}

func (m *MainPage) SetFooterInput(input textinput.Model) {
	m.Footer.Input = input
}

func (m *MainPage) Resize(width, height int) {
	m.width = width
	m.height = height
	m.Footer.Width = width
	m.Footer.Input.Width = width

	headerHeight := m.Header.Height()
	footerHeight := m.Footer.Height()
	vpHeight := height - headerHeight - footerHeight
	if vpHeight < 5 {
		vpHeight = 5
	}

	m.Viewport.Width = width
	m.Viewport.Height = vpHeight
}

func (m *MainPage) Refresh() {
	calcHeight := m.CalcViewportHeight()
	if calcHeight != m.Viewport.Height {
		m.Viewport.Height = calcHeight
	}
}

func (m *MainPage) CalcViewportHeight() int {
	return calcMainPageViewportHeight(m.Header, m.Footer, m.Viewport.Height+m.Header.Height()+m.Footer.Height(), m.Footer.Width)
}

func (m *MainPage) ScrollToLatest() {
	m.AutoScroll = true
	m.Viewport.GotoBottom()
}

func (m *MainPage) View() string {
	headerView := m.Header.View()
	footerView := m.Footer.View()

	headerHeight := lipgloss.Height(headerView)
	footerHeight := lipgloss.Height(footerView)

	vpHeight := m.Viewport.Height
	calcHeight := m.height - headerHeight - footerHeight
	if calcHeight < 5 {
		calcHeight = 5
	}

	if vpHeight != calcHeight {
		m.Viewport.Height = calcHeight
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerView, m.Viewport.View(), footerView)
}

func (h HeaderSection) Height() int {
	return lipgloss.Height(h.View())
}

func (h HeaderSection) View() string {
	return TitleStyle.Render(h.Title) + "\n" + TitleStyle.Render(h.Welcome)
}

func (f FooterSection) Height() int {
	return lipgloss.Height(f.View())
}

func (f FooterSection) View() string {
	return BuildFooter(f.LastError, f.RMStatuses, f.Input.View(), f.CommandHint, f.Width)
}

func calcMainPageViewportHeight(header HeaderSection, footer FooterSection, totalHeight int, width int) int {
	headerStr := header.View()
	footerStr := footer.View()
	headerHeight := lipgloss.Height(headerStr)
	footerHeight := lipgloss.Height(footerStr)
	vpHeight := totalHeight - headerHeight - footerHeight
	if vpHeight < 5 {
		vpHeight = 5
	}
	return vpHeight
}
