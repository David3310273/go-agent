package view

import (
	"fmt"
	"strings"

	"github.com/David3310273/go-agent/core"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type MCPListResultPage struct {
	Viewport   viewport.Model
	Active     bool
	ServerList []core.MCPConfig
	width      int
	height     int
}

func (m *MCPListResultPage) headerView() string {
	return TitleStyle.Render("MCP Servers")
}

func (m *MCPListResultPage) footerView() string {
	return "\n\n" + HintStyle.Render("Press q or Esc to exit")
}

func NewMCPListResultPage(width, height int, serverList []core.MCPConfig) *MCPListResultPage {
	m := &MCPListResultPage{
		Active:     true,
		ServerList: serverList,
		width:      width,
		height:     height,
	}
	header := m.headerView()
	footer := m.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	vp := viewport.New(width, vpHeight)
	vp.SetContent(BuildMCPListContent(serverList))
	m.Viewport = vp
	return m
}

func (m *MCPListResultPage) SetContent(serverList []core.MCPConfig) {
	m.ServerList = serverList
	m.Viewport.SetContent(BuildMCPListContent(serverList))
}

func (m *MCPListResultPage) View() string {
	header := m.headerView()
	footer := m.footerView()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	vpCalc := m.height - headerH - footerH
	if vpCalc < 5 {
		vpCalc = 5
	}
	if m.Viewport.Height != vpCalc {
		m.Viewport.Height = vpCalc
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, m.Viewport.View(), footer)
}

func (m *MCPListResultPage) ScrollUp(n int) {
	m.Viewport.ScrollUp(n)
	if m.Viewport.YOffset < 0 {
		m.Viewport.YOffset = 0
	}
}

func (m *MCPListResultPage) ScrollDown(n int) {
	m.Viewport.ScrollDown(n)
	if m.Viewport.YOffset < 0 {
		m.Viewport.YOffset = 0
	}
}

func (m *MCPListResultPage) Resize(width, height int, serverList []core.MCPConfig) {
	m.width = width
	m.height = height

	header := m.headerView()
	footer := m.footerView()
	vpHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.Viewport.Width = width
	m.Viewport.Height = vpHeight
	m.SetContent(serverList)
}

func BuildMCPListContent(serverList []core.MCPConfig) string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString(TitleStyle.Render("Your MCP Servers"))
	contentBuilder.WriteString("\n\n")
	separator := strings.Repeat("-", 40)
	for _, server := range serverList {
		contentBuilder.WriteString(separator)
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(fmt.Sprintf("- name: %s\n", server.Name))
		contentBuilder.WriteString(fmt.Sprintf("- host: %s:%d\n", server.Host, server.Port))
		contentBuilder.WriteString(fmt.Sprintf("- baseUrl: %s\n", server.BaseUrl))
		contentBuilder.WriteString(fmt.Sprintf("- protocolVersion: %s\n", server.ProtocolVersion))
		contentBuilder.WriteString(fmt.Sprintf("- jsonrpcVersion: %s\n", server.JSONRPCVersion))
		if server.Description != "" {
			contentBuilder.WriteString(fmt.Sprintf("- description: %s\n", server.Description))
		}
		contentBuilder.WriteString(separator)
		contentBuilder.WriteString("\n")
	}
	return contentBuilder.String()
}
