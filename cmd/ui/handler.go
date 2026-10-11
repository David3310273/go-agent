package ui

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/cmd/ui/view"
	"github.com/David3310273/go-agent/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofrs/uuid/v5"
)

// OutputMsg represents output from command execution
type OutputMsg struct {
	Content       string
	MsgType       string
	IsComplete    bool
	IsFinalAnswer bool
}

// RMResultMsg represents the result of a long-running operation
type RMResultMsg struct {
	Index   int
	Message string
	Code    core.MessageCode
}

// HandleSystemCommand handles system commands like /quit, /help, etc.
func HandleSystemCommand(page *UIPage, input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/quit", "/q":
		page.Quitting = true
		return tea.Quit
	case "/help", "/h":
		page.HelpHandler = NewHelpPageHandler(page.Width, page.Height, view.FormatHelpDoc(page.App.GetHelpDoc()))
		page.MainPage.Footer.Input.Blur()
		page.MainPage.Footer.Input.SetValue("")
		page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
		return nil
	case "/i18n":
		if len(parts) < 2 {
			page.MainPage.SetLastError("Language not specified")
			return nil
		}
		if parts[1] != "en" && parts[1] != "zh" {
			page.MainPage.SetLastError("Not supported language type")
			return nil
		}
		page.language = core.LanguageType(parts[1])
		page.MainPage.SetLastError("")
		return nil
	case "/new-page", "/np":
		id, _ := uuid.NewV4()
		page.SessionID = id.String()
		page.MainPage.Messages = []view.MessageEntry{}
		page.MainPage.Waiting = false
		page.MainPage.Streaming = false
		page.MainPage.StreamContent = ""
		page.MainPage.StreamTime = ""
		page.MainPage.CurrentHint = ""
		page.MainPage.HintOverwrite = false
		page.MainPage.SetMessages(page.MainPage.Messages, page.MainPage.Waiting, page.MainPage.Streaming,
			page.MainPage.CurrentHint, page.MainPage.StreamContent, page.MainPage.StreamTime)
		page.MainPage.ScrollToLatest()
		page.MainPage.Footer.Input.SetValue("")
		page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
		return nil
	case "/clear", "/clr":
		page.MainPage.Messages = []view.MessageEntry{}
		page.MainPage.Waiting = false
		page.MainPage.Streaming = false
		page.MainPage.StreamContent = ""
		page.MainPage.StreamTime = ""
		page.MainPage.CurrentHint = ""
		page.MainPage.HintOverwrite = false
		page.MainPage.SetMessages(page.MainPage.Messages, page.MainPage.Waiting, page.MainPage.Streaming,
			page.MainPage.CurrentHint, page.MainPage.StreamContent, page.MainPage.StreamTime)
		page.MainPage.ScrollToLatest()
		page.MainPage.Footer.Input.SetValue("")
		page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
		return nil
	}

	return nil
}

func handleAskCommand(page *UIPage, command *model.AskQuestionCommand) tea.Cmd {
	page.MainPage.Messages = append(page.MainPage.Messages, view.MessageEntry{
		Content: view.FormatTimestamp() + " " + command.Query,
		MsgType: "prompt",
	})
	page.MainPage.SetMessages(page.MainPage.Messages, page.MainPage.Waiting, page.MainPage.Streaming,
		page.MainPage.CurrentHint, page.MainPage.StreamContent, page.MainPage.StreamTime)
	page.MainPage.ScrollToLatest()
	page.MainPage.Streaming = command.Stream
	page.MainPage.StreamContent = ""

	return runAskCommand(page, command)
}

func runAskCommand(page *UIPage, command *model.AskQuestionCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	diag := command.Run(page.App)
	if diag != nil {
		page.MainPage.SetLastError(page.App.Render(diag.Code, page.language))
		return nil
	}

	outputChans := command.GetOutputChans()
	if len(outputChans) == 0 {
		return nil
	}

	page.MainPage.Waiting = true
	page.MainPage.Footer.Input.Blur()
	page.MainPage.SetFooterInput(page.MainPage.Footer.Input)

	if page.MainPage.Streaming {
		page.ResponseChan = outputChans[0].Chan.(chan core.Answer)
		page.HintChan = outputChans[1].Chan.(chan core.Answer)
		return waitForStreamOutput(outputChans)
	}

	var cmds []tea.Cmd
	for _, oc := range outputChans {
		ch, _ := oc.Chan.(chan core.Answer)
		cmds = append(cmds, waitForOutput(ch, oc.Type))
	}

	return tea.Batch(cmds...)
}

func handleSkillCommand(page *UIPage, command *model.SkillCommand) tea.Cmd {
	switch command.SubCommand {
	case "ls":
		return runSkillLsCommand(page, command)
	default:
		page.MainPage.SetLastError(fmt.Sprintf("Unknown skill action: %s", command.SubCommand))
		return nil
	}
}

func runSkillLsCommand(page *UIPage, command *model.SkillCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	diag := command.Run(page.App)
	if diag != nil {
		if diag.Data != nil {
			if skillDefs, ok := diag.Data.([]core.SkillDefinition); ok {
				skillList := make([]view.SkillListEntry, 0, len(skillDefs))
				for _, s := range skillDefs {
					skillList = append(skillList, view.SkillListEntry{
						Name:        s.Name,
						Description: s.Description,
						Tools:       s.Tools,
					})
				}
				page.SkillHandler = NewSkillListPageHandler(page.Width, page.Height, skillList)
				page.MainPage.Footer.Input.Blur()
				page.MainPage.Footer.Input.SetValue("")
				page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
				return nil
			}
		}
		if diag.Code != core.MessageCodeSuccess {
			page.MainPage.SetLastError(page.App.Render(diag.Code, page.language))
		}
		return nil
	}

	ch := command.ResponseChan
	if ch == nil {
		return nil
	}

	page.MainPage.Waiting = true
	page.MainPage.Footer.Input.Blur()
	page.MainPage.SetFooterInput(page.MainPage.Footer.Input)

	return func() tea.Msg {
		select {
		case diag, ok := <-ch:
			if !ok {
				return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
			}

			if diag.Data != nil {
				if skillDefs, ok := diag.Data.([]core.SkillDefinition); ok {
					skillList := make([]view.SkillListEntry, 0, len(skillDefs))
					for _, s := range skillDefs {
						skillList = append(skillList, view.SkillListEntry{
							Name:        s.Name,
							Description: s.Description,
							Tools:       s.Tools,
						})
					}
					page.SkillHandler = NewSkillListPageHandler(page.Width, page.Height, skillList)
					page.MainPage.Footer.Input.Blur()
					page.MainPage.Footer.Input.SetValue("")
					page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
					return nil
				}
			}
			page.MainPage.Messages = append(page.MainPage.Messages, view.MessageEntry{
				Content: view.FormatTimestamp() + " " + diag.Message,
				MsgType: "response",
			})
			page.MainPage.Waiting = false
			page.MainPage.Footer.Input.SetValue("")
			page.MainPage.Footer.Input.Focus()
			page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
			return nil

		case <-time.After(60 * time.Second):
			return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
		}
	}
}

// SkillListPageHandler
type SkillListPageHandler struct {
	Page *view.SkillListResultPage
}

func NewSkillListPageHandler(width, height int, skillList []view.SkillListEntry) *SkillListPageHandler {
	return &SkillListPageHandler{
		Page: view.NewSkillListResultPage(width, height, skillList),
	}
}

func (h *SkillListPageHandler) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter || msg.String() == "q" || msg.Type == tea.KeyEsc {
			return nil, true
		}
		switch msg.String() {
		case "up":
			h.Page.ScrollUp(1)
		case "down":
			h.Page.ScrollDown(1)
		}
		return nil, false
	case tea.WindowSizeMsg:
		h.Page.Resize(msg.Width, msg.Height, h.Page.SkillList)
		return nil, false
	}

	var cmd tea.Cmd
	h.Page.Viewport, cmd = h.Page.Viewport.Update(msg)
	return cmd, false
}

func (h *SkillListPageHandler) View() string {
	return h.Page.View()
}

func (h *SkillListPageHandler) IsActive() bool {
	return h.Page.Active
}

func (h *SkillListPageHandler) Deactivate() {
	h.Page.Active = false
}

func handleKBCommand(page *UIPage, command *model.KnowledgeBaseCommand) tea.Cmd {
	switch command.SubCommand {
	case "ls":
		return runKBLsCommand(page, command)
	case "add":
		return runKBAddCommand(page, command)
	case "rm":
		return runKBRmCommand(page, command)
	default:
		page.MainPage.SetLastError(fmt.Sprintf("Unknown KB action: %s", command.SubCommand))
		return nil
	}
}

func handleMCPCommand(page *UIPage, command *model.MCPCommand) tea.Cmd {
	switch command.SubCommand {
	case "ls":
		return runMCPLsCommand(page, command)
	case "add":
		return runMCPAddCommand(page, command)
	case "rm":
		return runMCPRmCommand(page, command)
	default:
		page.MainPage.SetLastError(fmt.Sprintf("Unknown MCP action: %s", command.SubCommand))
		return nil
	}
}

type commandHint struct {
	name   string
	format string
}

func buildCommandHints() []commandHint {
	commands := model.GetAllCommands()
	var hints []commandHint

	for _, cmd := range commands {
		help := cmd.GetHelp()
		hints = append(hints, parseHelpToHints(help, cmd.GetBase())...)
	}

	return hints
}

func getFormatHint(currentInput string, hints []commandHint) string {
	currentInput = strings.TrimSpace(currentInput)
	for _, hint := range hints {
		if strings.HasPrefix(hint.name, currentInput) || strings.HasPrefix(currentInput, hint.name) {
			return hint.format
		}
	}
	return ""
}

func parseHelpToHints(help map[string]any, name string) []commandHint {
	var hints []commandHint

	if format, ok := help["format"].(string); ok {
		if name != "" {
			hints = append(hints, commandHint{
				name:   name,
				format: format,
			})
		}
	} else {
		for key, value := range help {
			if subHelp, ok := value.(map[string]any); ok {
				if subFormat, ok := subHelp["format"].(string); ok {
					hints = append(hints, commandHint{
						name:   key,
						format: subFormat,
					})
				}
			}
		}
	}

	return hints
}

func runKBLsCommand(page *UIPage, command *model.KnowledgeBaseCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	diag := command.Run(page.App)
	if diag != nil {
		if diag.Data != nil {
			if fileList, ok := diag.Data.([]string); ok {
				page.KBHandler = NewKBListPageHandler(page.Width, page.Height, fileList)
				page.MainPage.Footer.Input.Blur()
				page.MainPage.Footer.Input.SetValue("")
				page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
				return nil
			}
		}
		if diag.Code != core.MessageCodeSuccess {
			page.MainPage.SetLastError(page.App.Render(diag.Code, page.language))
		}
		return nil
	}

	ch := command.ResponseChan
	if ch == nil {
		return nil
	}

	page.MainPage.Waiting = true
	page.MainPage.Footer.Input.Blur()
	page.MainPage.SetFooterInput(page.MainPage.Footer.Input)

	return func() tea.Msg {
		select {
		case diag, ok := <-ch:
			if !ok {
				return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
			}

			if diag.Data != nil {
				if fileList, ok := diag.Data.([]string); ok {
					page.KBHandler = NewKBListPageHandler(page.Width, page.Height, fileList)
					page.MainPage.Footer.Input.Blur()
					page.MainPage.Footer.Input.SetValue("")
					page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
					return nil
				}
			}
			page.MainPage.Messages = append(page.MainPage.Messages, view.MessageEntry{
				Content: view.FormatTimestamp() + " " + diag.Message,
				MsgType: "response",
			})
			page.MainPage.Waiting = false
			page.MainPage.Footer.Input.SetValue("")
			page.MainPage.Footer.Input.Focus()
			page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
			return nil

		case <-time.After(60 * time.Second):
			return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
		}
	}
}

func runKBAddCommand(page *UIPage, command *model.KnowledgeBaseCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	filename := path.Base(command.FilePath)
	index := len(page.MainPage.Footer.RMStatuses)
	page.MainPage.Footer.RMStatuses = append(page.MainPage.Footer.RMStatuses, view.RMStatusEntry{
		Message: fmt.Sprintf("Adding file %s...", filename),
		Done:    false,
	})
	page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)

	return func() tea.Msg {
		diag := command.Run(page.App)
		if diag != nil {
			return RMResultMsg{
				Index:   index,
				Message: page.App.Render(diag.Code, page.language),
				Code:    diag.Code,
			}
		}

		select {
		case diag, ok := <-command.ResponseChan:
			if !ok {
				return RMResultMsg{Index: index, Message: "Operation failed", Code: core.MessageCodeSystemError}
			}
			return RMResultMsg{
				Index:   index,
				Message: diag.Message,
				Code:    diag.Code,
			}
		case <-time.After(60 * time.Second):
			return RMResultMsg{Index: index, Message: fmt.Sprintf("Adding %s timed out after 60s", filename), Code: core.MessageCodeCommandTimeout}
		}
	}
}

func runKBRmCommand(page *UIPage, command *model.KnowledgeBaseCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	filename := path.Base(command.FilePath)
	index := len(page.MainPage.Footer.RMStatuses)
	page.MainPage.Footer.RMStatuses = append(page.MainPage.Footer.RMStatuses, view.RMStatusEntry{
		Message: fmt.Sprintf("Removing file %s...", filename),
		Done:    false,
	})
	page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)

	return func() tea.Msg {
		diag := command.Run(page.App)
		if diag != nil {
			return RMResultMsg{
				Index:   index,
				Message: page.App.Render(diag.Code, page.language),
				Code:    diag.Code,
			}
		}

		select {
		case diag, ok := <-command.ResponseChan:
			if !ok {
				return RMResultMsg{Index: index, Message: "Operation failed", Code: core.MessageCodeSystemError}
			}
			return RMResultMsg{
				Index:   index,
				Message: diag.Message,
				Code:    diag.Code,
			}
		case <-time.After(60 * time.Second):
			return RMResultMsg{Index: index, Message: fmt.Sprintf("Removing %s timed out", filename), Code: core.MessageCodeCommandTimeout}
		}
	}
}

// KBListPageHandler
type KBListPageHandler struct {
	Page *view.KBListResultPage
}

func NewKBListPageHandler(width, height int, fileList []string) *KBListPageHandler {
	return &KBListPageHandler{
		Page: view.NewKBListResultPage(width, height, fileList),
	}
}

func (h *KBListPageHandler) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter || msg.String() == "q" || msg.Type == tea.KeyEsc {
			return nil, true
		}
		switch msg.String() {
		case "up":
			h.Page.ScrollUp(1)
		case "down":
			h.Page.ScrollDown(1)
		}
		return nil, false
	case tea.WindowSizeMsg:
		h.Page.Resize(msg.Width, msg.Height, h.Page.FileList)
		return nil, false
	}

	var cmd tea.Cmd
	h.Page.Viewport, cmd = h.Page.Viewport.Update(msg)
	return cmd, false
}

func (h *KBListPageHandler) View() string {
	return h.Page.View()
}

func (h *KBListPageHandler) IsActive() bool {
	return h.Page.Active
}

func (h *KBListPageHandler) Deactivate() {
	h.Page.Active = false
}

func runMCPLsCommand(page *UIPage, command *model.MCPCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	diag := command.Run(page.App)
	if diag != nil {
		if diag.Data != nil {
			if serverList, ok := diag.Data.([]core.MCPConfig); ok {
				page.MCPHandler = NewMCPListPageHandler(page.Width, page.Height, serverList)
				page.MainPage.Footer.Input.Blur()
				page.MainPage.Footer.Input.SetValue("")
				page.MainPage.SetFooterInput(page.MainPage.Footer.Input)
				return nil
			}
		}
		if diag.Code != core.MessageCodeSuccess {
			page.MainPage.SetLastError(page.App.Render(diag.Code, page.language))
		}
		return nil
	}

	return nil
}

func runMCPAddCommand(page *UIPage, command *model.MCPCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	index := len(page.MainPage.Footer.RMStatuses)
	page.MainPage.Footer.RMStatuses = append(page.MainPage.Footer.RMStatuses, view.RMStatusEntry{
		Message: fmt.Sprintf("Adding MCP server %s...", command.ServerName),
		Done:    false,
	})
	page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)

	diag := command.Run(page.App)
	if diag != nil {
		page.MainPage.Footer.RMStatuses[index].Message = page.App.Render(diag.Code, page.language)
		page.MainPage.Footer.RMStatuses[index].Done = true
		page.MainPage.Footer.RMStatuses[index].Code = diag.Code
		page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)
	}
	return nil
}

func runMCPRmCommand(page *UIPage, command *model.MCPCommand) tea.Cmd {
	if page.App.Agent == nil {
		page.MainPage.SetLastError("Agent not initialized")
		return nil
	}

	index := len(page.MainPage.Footer.RMStatuses)
	page.MainPage.Footer.RMStatuses = append(page.MainPage.Footer.RMStatuses, view.RMStatusEntry{
		Message: fmt.Sprintf("Removing MCP server %s...", command.ServerName),
		Done:    false,
	})
	page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)

	diag := command.Run(page.App)
	if diag != nil {
		page.MainPage.Footer.RMStatuses[index].Message = page.App.Render(diag.Code, page.language)
		page.MainPage.Footer.RMStatuses[index].Done = true
		page.MainPage.Footer.RMStatuses[index].Code = diag.Code
		page.MainPage.SetRMStatuses(page.MainPage.Footer.RMStatuses)
	}
	return nil
}

// MCPListPageHandler
type MCPListPageHandler struct {
	Page *view.MCPListResultPage
}

func NewMCPListPageHandler(width, height int, serverList []core.MCPConfig) *MCPListPageHandler {
	return &MCPListPageHandler{
		Page: view.NewMCPListResultPage(width, height, serverList),
	}
}

func (h *MCPListPageHandler) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter || msg.String() == "q" || msg.Type == tea.KeyEsc {
			return nil, true
		}
		switch msg.String() {
		case "up":
			h.Page.ScrollUp(1)
		case "down":
			h.Page.ScrollDown(1)
		}
		return nil, false
	case tea.WindowSizeMsg:
		h.Page.Resize(msg.Width, msg.Height, h.Page.ServerList)
		return nil, false
	}

	var cmd tea.Cmd
	h.Page.Viewport, cmd = h.Page.Viewport.Update(msg)
	return cmd, false
}

func (h *MCPListPageHandler) View() string {
	return h.Page.View()
}

func (h *MCPListPageHandler) IsActive() bool {
	return h.Page.Active
}

func (h *MCPListPageHandler) Deactivate() {
	h.Page.Active = false
}

// HelpPageHandler
type HelpPageHandler struct {
	Page    *view.HelpPage
	content string
}

func NewHelpPageHandler(width, height int, content string) *HelpPageHandler {
	return &HelpPageHandler{
		Page:    view.NewHelpPage(width, height, content),
		content: content,
	}
}

func (h *HelpPageHandler) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter || msg.String() == "q" || msg.Type == tea.KeyEsc {
			return nil, true
		}
		switch msg.String() {
		case "up", "k":
			h.Page.ScrollUp(1)
		case "down", "j":
			h.Page.ScrollDown(1)
		}
		return nil, false
	case tea.WindowSizeMsg:
		h.Page.Resize(msg.Width, msg.Height, h.content)
		return nil, false
	}

	var cmd tea.Cmd
	h.Page.Viewport, cmd = h.Page.Viewport.Update(msg)
	return cmd, false
}

func (h *HelpPageHandler) View() string {
	return h.Page.View()
}

func (h *HelpPageHandler) IsActive() bool {
	return h.Page.Active
}

func (h *HelpPageHandler) Deactivate() {
	h.Page.Active = false
}

// Helper functions
func extractOutput(answer core.Answer, msgType string) OutputMsg {
	if simpleResp, ok := answer.(simple.SimpleNormalResponse); ok {
		if simpleResp.Response.Response != "" {
			return OutputMsg{Content: simpleResp.Response.Response, MsgType: msgType, IsFinalAnswer: true}
		}
		if len(simpleResp.Response.Choices) > 0 && simpleResp.Response.Choices[0].Message != nil {
			content := simpleResp.Response.Choices[0].Message.Content
			isComplete := simpleResp.Response.Choices[0].FinishReason != ""
			return OutputMsg{Content: content, MsgType: msgType, IsComplete: isComplete, IsFinalAnswer: isComplete}
		}
	}
	if confirmResp, ok := answer.(simple.SimpleToolConfirmResponse); ok {
		if confirmResp.Response.Response != "" {
			return OutputMsg{Content: confirmResp.Response.Response, MsgType: msgType, IsFinalAnswer: true}
		}
		if len(confirmResp.Response.Choices) > 0 && confirmResp.Response.Choices[0].Message != nil {
			return OutputMsg{Content: confirmResp.Response.Choices[0].Message.Content, MsgType: msgType}
		}
	}
	if resp, ok := answer.(core.AgentResponse); ok {
		if resp.Response != "" {
			return OutputMsg{Content: resp.Response, MsgType: msgType, IsFinalAnswer: true}
		}
		if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
			if resp.Choices[0].Message.ToolCalls != nil && len(*resp.Choices[0].Message.ToolCalls) > 0 {
				return OutputMsg{Content: "", MsgType: msgType, IsFinalAnswer: false}
			}
			content := resp.Choices[0].Message.Content
			if content == "" && resp.Choices[0].Message.ReasoningContent != nil {
				content = *resp.Choices[0].Message.ReasoningContent
			}
			isComplete := resp.Choices[0].FinishReason != ""
			return OutputMsg{Content: content, MsgType: msgType, IsComplete: isComplete, IsFinalAnswer: isComplete}
		}
	}
	return OutputMsg{Content: answer.ToString(), MsgType: msgType}
}

func waitForStreamOutput(outputChans []core.OutputChannel) tea.Cmd {
	return func() tea.Msg {
		responseChan, _ := outputChans[0].Chan.(chan core.Answer)
		hintChan, _ := outputChans[1].Chan.(chan core.Answer)

		select {
		case answer, ok := <-responseChan:
			if !ok {
				core.LogStd(core.LogLevelDebug, "UI: responseChan closed, stream complete")
				return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
			}
			return extractOutput(answer, "response")

		case answer, ok := <-hintChan:
			if !ok {
				core.LogStd(core.LogLevelDebug, "UI: hintChan closed")
				return OutputMsg{Content: "", MsgType: "hint", IsComplete: true, IsFinalAnswer: true}
			}
			return extractOutput(answer, "hint")

		case <-time.After(60 * time.Second):
			core.LogStd(core.LogLevelWarn, "UI: stream output timeout")
			return OutputMsg{Content: "", MsgType: "response", IsComplete: true, IsFinalAnswer: true}
		}
	}
}

func waitForOutput(ch chan core.Answer, msgType string) tea.Cmd {
	return func() tea.Msg {
		answer := <-ch
		return extractOutput(answer, msgType)
	}
}

func FormatTimestamp() string {
	return view.FormatTimestamp()
}
