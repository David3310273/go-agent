package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type SkillListEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

type SkillListResultPage struct {
	Viewport  viewport.Model
	Active    bool
	SkillList []SkillListEntry
	width     int
	height    int
}

func (s *SkillListResultPage) headerView() string {
	return TitleStyle.Render("Skills")
}

func (s *SkillListResultPage) footerView() string {
	return "\n\n" + HintStyle.Render("Press q or Esc to exit")
}

func NewSkillListResultPage(width, height int, skillList []SkillListEntry) *SkillListResultPage {
	s := &SkillListResultPage{
		Active:    true,
		SkillList: skillList,
		width:     width,
		height:    height,
	}
	header := s.headerView()
	footer := s.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	vp := viewport.New(width, vpHeight)
	vp.SetContent(BuildSkillListContent(skillList))
	s.Viewport = vp
	return s
}

func (s *SkillListResultPage) SetContent(skillList []SkillListEntry) {
	s.SkillList = skillList
	s.Viewport.SetContent(BuildSkillListContent(skillList))
}

func (s *SkillListResultPage) View() string {
	header := s.headerView()
	footer := s.footerView()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	vpCalc := s.height - headerH - footerH
	if vpCalc < 5 {
		vpCalc = 5
	}
	if s.Viewport.Height != vpCalc {
		s.Viewport.Height = vpCalc
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, s.Viewport.View(), footer)
}

func (s *SkillListResultPage) ScrollUp(n int) {
	s.Viewport.ScrollUp(n)
	if s.Viewport.YOffset < 0 {
		s.Viewport.YOffset = 0
	}
}

func (s *SkillListResultPage) ScrollDown(n int) {
	s.Viewport.ScrollDown(n)
	if s.Viewport.YOffset < 0 {
		s.Viewport.YOffset = 0
	}
}

func (s *SkillListResultPage) Resize(width, height int, skillList []SkillListEntry) {
	s.width = width
	s.height = height

	header := s.headerView()
	footer := s.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	s.Viewport.Width = width
	s.Viewport.Height = vpHeight
	s.SetContent(skillList)
}

func BuildSkillListContent(skillList []SkillListEntry) string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(TitleStyle.Render("Available Skills"))
	contentBuilder.WriteString("\n\n")
	separator := strings.Repeat("-", 40)
	for _, skill := range skillList {
		contentBuilder.WriteString(separator)
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(fmt.Sprintf("- name: %s\n", skill.Name))
		contentBuilder.WriteString(fmt.Sprintf("- desc: %s\n", skill.Description))
		contentBuilder.WriteString(fmt.Sprintf("- tools: %v\n", skill.Tools))
		contentBuilder.WriteString(separator)
		contentBuilder.WriteString("\n")
	}
	return contentBuilder.String()
}
