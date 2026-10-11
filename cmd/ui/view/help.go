package view

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/core"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

func FormatHelpDoc(cmdIntro map[string]any) string {
	var sb strings.Builder

	for cmdName, rawHelp := range cmdIntro {
		help := rawHelp.(map[string]any)
		core.LogStd(core.LogLevelDebug, "FormatHelpDoc: help msg: %#v", help)
		if desc, ok := help["desc"].(string); ok {
			sb.WriteString(fmt.Sprintf("%s: %s\n\n", cmdName, desc))
		}

		if format, ok := help["format"].(string); ok {
			sb.WriteString("- usage:  " + format)
			sb.WriteString("\n\n")
		}

		if options, ok := help["options"].(map[string]string); ok {
			if len(options) > 0 {
				sb.WriteString("    Options:\n\n")
				for key, desc := range options {
					sb.WriteString(fmt.Sprintf("      --%-20s %s\n", key, desc))
				}
			}
		}

		sb.WriteString("\n")
	}

	core.LogStd(core.LogLevelDebug, "FormatHelpDoc: help content: %s", sb.String())

	return sb.String()
}

type HelpPage struct {
	Viewport viewport.Model
	Active   bool
}

func NewHelpPage(width, height int, content string) *HelpPage {
	header := TitleStyle.Render("Help Documentation") + "\n\n"
	footer := "\n\n" + HintStyle.Render("Press q or Esc to exit help")
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	vp := viewport.New(width, vpHeight)
	vp.SetContent(content)
	return &HelpPage{
		Viewport: vp,
		Active:   true,
	}
}

func (h *HelpPage) SetContent(content string) {
	h.Viewport.SetContent(content)
	h.Viewport.SetYOffset(0)
}

func (h *HelpPage) View() string {
	header := TitleStyle.Render("Help Documentation") + "\n\n"
	footer := "\n\n" + HintStyle.Render("Press q or Esc to exit help")

	return lipgloss.JoinVertical(lipgloss.Left, header, h.Viewport.View(), footer)
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
