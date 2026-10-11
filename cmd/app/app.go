package app

import (
	"fmt"
	"path"
	"strings"

	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
	"gopkg.in/ini.v1"
)

type SimpleCommandApp struct {
	Agent        core.AgentCore
	Config       core.CommandAppConfig
	Parser       core.CommandParser
	QuestionChan chan core.Command
	Commands     map[string]core.Command
}

var _ core.CommandApp = (*SimpleCommandApp)(nil)

func NewSimpleCommandApp(agent core.AgentCore, config core.CommandAppConfig, parser core.CommandParser) *SimpleCommandApp {
	app := &SimpleCommandApp{
		Agent:    agent,
		Config:   config,
		Parser:   parser,
		Commands: make(map[string]core.Command),
	}
	app.registerCommands()
	return app
}

func (a *SimpleCommandApp) registerCommands() {
	a.Commands = model.GetAllCommands()
}

func (a *SimpleCommandApp) GetHelpDoc() map[string]any {
	commandDetails := make(map[string]any)

	for _, cmd := range a.Commands {
		if cmd.GetBase() == model.SystemCommandName {
			systemCmds := cmd.GetHelp()
			for base, help := range systemCmds {
				commandDetails[base] = help.(map[string]any)
			}
		} else {
			help := cmd.GetHelp()
			commandDetails[cmd.GetBase()] = help
		}
	}

	return commandDetails
}

func (a *SimpleCommandApp) GetSystemCmdHelpDoc() string {
	var sb strings.Builder
	sb.WriteString("\n\n")
	sb.WriteString("/quit, /exit, /q\n")
	sb.WriteString("    Quit the application\n\n")
	sb.WriteString("/help, /h\n")
	sb.WriteString("    Show this help documentation\n\n")
	sb.WriteString("/new-page\n")
	sb.WriteString("    Start a new conversation page (clears current session)\n\n")
	sb.WriteString("/i18n <language>\n")
	sb.WriteString("    Switch language (supported: en_US, zh_CN)\n\n")
	sb.WriteString("/clear\n")
	sb.WriteString("    Clear the current screen\n\n")
	return sb.String()
}

// UI interface implementation

func (a *SimpleCommandApp) Render(code core.MessageCode, locale core.LanguageType) string {
	return a.Translate(code, locale)
}

func (a *SimpleCommandApp) GetAgentCore() *core.AgentCore {
	return &a.Agent
}

func (a *SimpleCommandApp) GetConfig() core.CommandAppConfig {
	return a.Config
}

func (a *SimpleCommandApp) RenderSystemMessage(diagnostics *core.DiagnosticList, locale core.LanguageType) string {
	var messages []string
	for _, diag := range *diagnostics {
		diag.Message = a.Translate(diag.Code, locale)
		if diag.Message != "" {
			messages = append(messages, diag.Message)
		}
	}
	return strings.Join(messages, "\n")
}

func (a *SimpleCommandApp) Welcome() string {
	welcomeMsg := `
========================================
   Welcome to Go-Agent Command Line
========================================
`
	return welcomeMsg
}

func (a *SimpleCommandApp) GetUserInput() chan core.Command {
	return a.QuestionChan
}

// CommandParser interface implementation

func (a *SimpleCommandApp) Parse(input string) (core.Command, []core.Diagnostic) {
	return a.Parser.Parse(input)
}

// UserManager interface implementation

func (a *SimpleCommandApp) AuthUser(token string) *core.Diagnostic {
	// TODO: implement auth user logic
	return nil
}

func (a *SimpleCommandApp) GetUserPlan(userID string) (core.Plan, *core.Diagnostic) {
	// TODO: if you need user management
	return core.Plan{}, nil
}

func (a *SimpleCommandApp) CheckUserPlan(userInfo any, plan core.Plan) *core.Diagnostic {
	// TODO if you need user management
	// 1. check valid date of user plan
	// 2. check token usage
	// 3. check available models
	return nil
}

// I18n interface implementation

func (a *SimpleCommandApp) Translate(key core.MessageCode, targetLanguage core.LanguageType) string {
	i18nPath := path.Join(a.Config.RootPath, a.Config.I18nConfig.FilePath, fmt.Sprintf("%s.ini", string(targetLanguage)))
	core.LogStd(core.LogLevelDebug, "i18nPath: %s", i18nPath)
	cfg, err := ini.Load(i18nPath)
	if err != nil {
		core.LogStd(core.LogLevelWarn, "failed to load i18n file: %s", i18nPath)
		return ""
	}

	keyStr := fmt.Sprintf("%d", key)
	return cfg.Section("").Key(keyStr).String()
}

func (a *SimpleCommandApp) GracefulQuit() {
	err := core.StopAgentCore(a.Agent)
	if len(err) > 0 {
		core.LogStd(core.LogLevelError, "agent %s stopped error: %v", a.Agent.GetID(), core.DiagnosticList(err).ToString())
	} else {
		core.LogStd(core.LogLevelInfo, "agent %s stopped", a.Agent.GetID())
	}
}

// WorkFlow interface implementation

func (a *SimpleCommandApp) BeforeStart() []core.Diagnostic {
	// TODO: pre check for user plan
	_, diag := a.GetUserPlan("")
	if diag != nil {
		return []core.Diagnostic{*diag}
	}

	diag = a.CheckUserPlan(nil, core.Plan{})

	if diag != nil {
		return []core.Diagnostic{*diag}
	}

	return nil
}

func (a *SimpleCommandApp) Start() []core.Diagnostic {
	// start agent core in a goroutine
	go func() {
		diagnostics := core.StartAgentCore(a.Agent, a.Config.AppConfig)
		if len(diagnostics) > 0 {
			for _, d := range diagnostics {
				if d.Level >= core.SeverityError {
					core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
				}
			}
		}
	}()

	for command := range a.GetUserInput() {
		if command == nil {
			continue
		}

		diag := command.Run(a)
		if diag != nil {
			a.RenderSystemMessage(&core.DiagnosticList{*diag}, a.Config.Language)
		}
	}

	return nil
}

func (a *SimpleCommandApp) BeforeStop() []core.Diagnostic {
	return nil
}

func (a *SimpleCommandApp) Stop() []core.Diagnostic {
	a.GracefulQuit()
	return nil
}
