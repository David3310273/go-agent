package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/David3310273/go-agent/agent/simple"
	"github.com/David3310273/go-agent/cmd/config"
	"github.com/David3310273/go-agent/cmd/controller"
	"github.com/David3310273/go-agent/cmd/model"
	"github.com/David3310273/go-agent/core"
	_ "github.com/David3310273/go-agent/providers/qwen"
)

func listenToInput(agent core.AgentCore, parser core.CommandParser) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			core.LogStd(core.LogLevelError, "failed to read input: %v", err)
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// parse command
		command, diagnostics := parser.Parse(line)
		if len(diagnostics) > 0 {
			for _, d := range diagnostics {
				if d.Level >= core.SeverityError {
					core.LogStd(core.LogLevelError, "Parse error: %s", d.Message)
				} else {
					core.LogStd(core.LogLevelWarn, "Parse warning: %s", d.Message)
				}
			}
			continue
		}

		// run cmd
		switch cmd := command.(type) {
		case *model.AskQuestionCommand:
			responseChan := make(chan core.Answer, 1)
			hintChan := make(chan core.Answer, 1)

			question := simple.NewSimpleQuestion(
				cmd.Query,
				"",
				cmd.Model,
				responseChan,
				hintChan,
				cmd.Stream,
				cmd.EnableThinking,
				cmd.QuestionType,
			)

			cmd.WithStdout(os.Stdout, os.Stdout)

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				for response := range responseChan {
					os.Stdout.Write([]byte(response.ToString()))
				}
			}()
			go func() {
				defer wg.Done()
				for hint := range hintChan {
					os.Stdout.Write([]byte(hint.ToString()))
				}
			}()

			diag := cmd.Run(&agent, question)
			close(responseChan)
			close(hintChan)
			wg.Wait()

			if diag != nil && diag.Level >= core.SeverityError {
				core.LogStd(core.LogLevelError, "Command execution error: %s", diag.Message)
			}
		case *model.QuitCommand:
			cmd.Run(&agent, nil)
			return nil
		default:
			core.LogStd(core.LogLevelError, "Unknown command type")
		}
	}
}

func startCmdApp(agent core.AgentCore) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// start signal listener
	go func() {
		sig := <-sigChan
		core.LogStd(core.LogLevelInfo, "received signal: %v, shutting down...", sig)
		err := core.StopAgentCore(agent)
		if len(err) > 0 {
			core.LogStd(core.LogLevelError, "agent %s stopped error: %v", agent.GetID(), core.DiagnosticList(err).ToString())
		} else {
			core.LogStd(core.LogLevelInfo, "agent %s stopped", agent.GetID())
		}
		os.Exit(0)
	}()

	parser := controller.SimpleCommandParser{}

	fmt.Println("Agent CLI started. Type commands or 'quit' to exit.")
	fmt.Println("Available commands: /ask [question] [--options]")
	fmt.Println()

	return listenToInput(agent, parser)
}

func main() {
	// get working directory
	cwd, err := os.Getwd()
	if err != nil {
		core.LogStd(core.LogLevelError, "failed to get working directory: %v", err.Error())
		os.Exit(1)
	}

	// load config first
	configPath := path.Join(cwd, "config.json")
	config, configErr := config.LoadAppConfig(configPath)
	if configErr != nil {
		core.LogStd(core.LogLevelError, "failed to load app config: %v", configErr.Message)
		os.Exit(1)
	}

	agent, diag := simple.NewSimpleAgent(config.RootPath)
	if diag != nil {
		core.LogStd(core.LogLevelError, "failed to create agent: %v", diag.ToString())
		os.Exit(1)
	}

	// redirect internal logs to file, keep stdout clean for CLI interaction
	logPath := path.Join(cwd, "logs", fmt.Sprintf("agent-%s.log", agent.GetID()))
	core.SetDefaultLogPath(logPath)
	core.LogStd(core.LogLevelInfo, "agent logs written to %s", logPath)
	core.LogStd(core.LogLevelInfo, "cmd app starting, config %#v loaded from %s", config, configPath)

	providers := simple.CreateProviders(config.RootPath, agent.Configs.Agent)
	if diag := agent.SetProviders(providers); diag != nil {
		core.LogStd(core.LogLevelError, "failed to set providers: %v", diag.ToString())
		os.Exit(1)
	}

	go func() {
		diagnostics := core.StartAgentCore(agent, config.AppConfig)
		if len(diagnostics) > 0 {
			for _, d := range diagnostics {
				if d.Level >= core.SeverityError {
					core.LogStd(core.LogLevelError, "StartAgentCore error: %s", d.ToString())
				}
			}
		}
	}()
	// start command app
	if err := startCmdApp(agent); err != nil {
		core.LogStd(core.LogLevelError, "Command app error: %v", err)
		os.Exit(1)
	}
}
