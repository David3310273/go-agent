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

func (a *SimpleCommandApp) GetHelpDoc() string {
	var sb strings.Builder
	sb.WriteString("Available commands:\n\n")
	for _, cmd := range a.Commands {
		help := cmd.GetHelp()
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", cmd.GetBase(), help["desc"].(string)))
		sb.WriteString("usage:  " + help["format"].(string))
		if desc, ok := help["desc"].(string); ok {
			sb.WriteString("\n    " + desc)
		}
		sb.WriteString("\n\n")
		if options, ok := help["options"].(map[string]string); ok {
			sb.WriteString("    Options:\n")
			for key, desc := range options {
				sb.WriteString(fmt.Sprintf("      --%-20s %s\n", key, desc))
			}
		}
		sb.WriteString("\n")
	}
	// add more help info here
	sb.WriteString("\n")
	sb.WriteString("Additional Information:\n")
	sb.WriteString("  This is a test line to make help doc longer for scrolling.\n")
	sb.WriteString("  Line 1: Lorem ipsum dolor sit amet, consectetur adipiscing elit.\n")
	sb.WriteString("  Line 2: Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.\n")
	sb.WriteString("  Line 3: Ut enim ad minim veniam, quis nostrud exercitation ullamco.\n")
	sb.WriteString("  Line 4: Duis aute irure dolor in reprehenderit in voluptate velit esse.\n")
	sb.WriteString("  Line 5: Excepteur sint occaecat cupidatat non proident, sunt in culpa.\n")
	sb.WriteString("  Line 6: Qui officia deserunt mollit anim id est laborum.\n")
	sb.WriteString("  Line 7: Sed ut perspiciatis unde omnis iste natus error sit voluptatem.\n")
	sb.WriteString("  Line 8: Accusantium doloremque laudantium, totam rem aperiam, eaque ipsa.\n")
	sb.WriteString("  Line 9: Quae ab illo inventore veritatis et quasi architecto beatae vitae.\n")
	sb.WriteString("  Line 10: Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit.\n")
	sb.WriteString("  Line 11: Aut fugit, sed quia consequuntur magni dolores eos qui ratione.\n")
	sb.WriteString("  Line 12: Voluptatem sequi nesciunt neque porro quisquam est qui dolorem.\n")
	sb.WriteString("  Line 13: Ipsum quia dolor sit amet, consectetur, adipisci velit.\n")
	sb.WriteString("  Line 14: Sed quia non numquam eius modi tempora incidunt ut labore.\n")
	sb.WriteString("  Line 15: Et dolore magnam aliquam quaerat voluptatem ut enim ad minima.\n")
	sb.WriteString("  Line 16: Veniam, quis nostrum exercitationem ullam corporis suscipit.\n")
	sb.WriteString("  Line 17: Laboriosam, nisi ut aliquid ex ea commodi consequatur.\n")
	sb.WriteString("  Line 18: Quis autem vel eum iure reprehenderit qui in ea voluptate velit.\n")
	sb.WriteString("  Line 19: Esse quam nihil molestiae consequatur, vel illum qui dolorem.\n")
	sb.WriteString("  Line 20: Eum fugiat quo voluptas nulla pariatur at vero eos et accusamus.\n")
	sb.WriteString("\n")
	sb.WriteString("  End of help documentation.\n")
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
	i18nPath := path.Join(a.Config.I18nConfig.FilePath, fmt.Sprintf("%s.ini", string(targetLanguage)))
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
