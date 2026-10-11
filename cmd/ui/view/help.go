package view

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type HelpPage struct {
	Viewport viewport.Model
	Active   bool
}

func NewHelpPage(width, height int, content string) *HelpPage {
	vp := viewport.New(width, height)
	vp.SetContent(content)
	return &HelpPage{
		Viewport: vp,
		Active:   true,
	}
}

func (h *HelpPage) SetContent(content string) {
	h.Viewport.SetContent(content)
}

func (h *HelpPage) View() string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(TitleStyle.Render("Help Documentation"))
	contentBuilder.WriteString("\n\n")
	contentBuilder.WriteString(h.Viewport.View())
	contentBuilder.WriteString("\n\n")
	contentBuilder.WriteString(HintStyle.Render("Press q or Esc to exit help"))

	return contentBuilder.String()
}

func (h *HelpPage) ScrollUp(n int) {
	h.Viewport.ScrollUp(n)
	if h.Viewport.YOffset < 0 {
		h.Viewport.YOffset = 0
	}
}

func (h *HelpPage) ScrollDown(n int) {
	h.Viewport.ScrollDown(n)
	if h.Viewport.YOffset < 0 {
		h.Viewport.YOffset = 0
	}
}

func (h *HelpPage) Resize(width, height int, content string) {
	header := TitleStyle.Render("Help Documentation") + "\n\n"
	footer := "\n\n" + HintStyle.Render("Press q or Esc to exit help")
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	h.Viewport.Width = width
	h.Viewport.Height = vpHeight
	h.SetContent(content)
}
