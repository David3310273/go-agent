package view

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type KBListResultPage struct {
	Viewport viewport.Model
	Active   bool
	FileList []string
	width    int
	height   int
}

func (k *KBListResultPage) headerView() string {
	return TitleStyle.Render("Knowledge Base")
}

func (k *KBListResultPage) footerView() string {
	return "\n\n" + HintStyle.Render("Press q or Esc to exit")
}

func NewKBListResultPage(width, height int, fileList []string) *KBListResultPage {
	k := &KBListResultPage{
		Active:   true,
		FileList: fileList,
		width:    width,
		height:   height,
	}
	header := k.headerView()
	footer := k.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	vp := viewport.New(width, vpHeight)
	vp.SetContent(BuildKBListContent(fileList))
	k.Viewport = vp
	return k
}

func (k *KBListResultPage) SetContent(fileList []string) {
	k.FileList = fileList
	k.Viewport.SetContent(BuildKBListContent(fileList))
}

func (k *KBListResultPage) View() string {
	header := k.headerView()
	footer := k.footerView()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	vpCalc := k.height - headerH - footerH
	if vpCalc < 5 {
		vpCalc = 5
	}
	if k.Viewport.Height != vpCalc {
		k.Viewport.Height = vpCalc
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, k.Viewport.View(), footer)
}

func (k *KBListResultPage) ScrollUp(n int) {
	k.Viewport.ScrollUp(n)
	if k.Viewport.YOffset < 0 {
		k.Viewport.YOffset = 0
	}
}

func (k *KBListResultPage) ScrollDown(n int) {
	k.Viewport.ScrollDown(n)
	if k.Viewport.YOffset < 0 {
		k.Viewport.YOffset = 0
	}
}

func (k *KBListResultPage) Resize(width, height int, fileList []string) {
	k.width = width
	k.height = height

	header := k.headerView()
	footer := k.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	k.Viewport.Width = width
	k.Viewport.Height = vpHeight
	k.SetContent(fileList)
}

func BuildKBListContent(fileList []string) string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(TitleStyle.Render("Your Knowledgebase Files"))
	contentBuilder.WriteString("\n\n")
	for _, file := range fileList {
		contentBuilder.WriteString(PromptUserStyle.Render("- " + file))
		contentBuilder.WriteString("\n")
	}
	return contentBuilder.String()
}
