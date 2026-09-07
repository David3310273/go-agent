# Contributing to go-agent

Thank you for your interest in contributing to go-agent! This document provides guidelines and conventions for contributing to this project.

## Project Overview

go-agent is an interface-implemented, multi-layered agent framework written in Go. The project follows a layered architecture with clear separation of concerns.

### Architecture Layers

- **core/**: Framework interfaces and core logic (AgentCore, Session, Provider, Tool, etc.)
- **agent/**: Agent implementations (e.g., `simple/`)
- **providers/**: LLM provider adapters (e.g., `qwen/`)
- **app/**: User-facing applications (server, CLI, web app)

### Design Principles

- **Param injection** over struct-embedded variables
- **Interface composition** with single-responsibility interface units
- **core/** must not import specific implementation packages, for example, from agent or app implementations

## Development Setup

### Prerequisites

- Go 1.25+
- Git

### Getting Started

```bash
git clone https://github.com/David3310273/go-agent.git
cd go-agent
go mod download
```

### Running the example application

```bash
cd app
go run main.go
```

### Running Tests

```bash
go test ./...
```

## Code Conventions

### Naming

| Element | Convention | Example |
|---------|-----------|---------|
| struct, interface, function | PascalCase | `AgentCore`, `ProcessQuestion` |
| variables | camelCase | `agentCore`, `maxRounds` |
| ID (not Id) | Always uppercase | `AgentID`, `SessionID` |

### Comments

- All new code must include English comments
- Explain **why**, not **what** — the code should be self-explanatory for the "what"

### Error Handling

- Use `*core.Diagnostic` for error reporting within the framework
- Include meaningful `Message` fields in diagnostics for debugging
- Use appropriate `MessageCode` constants from `core/errcode.go`

### File Organization

- One primary type per file
- Keep related helper functions in the same file
- Use `template.json` files for config examples (never commit real credentials)

## How to Contribute

### Adding a New Provider

1. Create a new directory under `providers/` (e.g., `providers/openai/`)
2. Implement the `core.Provider` interface
3. Register your provider factory in `init()`:
   ```go
   func init() {
       core.RegisterProviderFactory("openai", func() (core.Provider, *core.Diagnostic) {
           return NewOpenAIProvider()
       })
   }
   ```
4. Add a `<provider>.template.json` with placeholder values
5. **Never commit real API keys** — add your config file to `.gitignore`

### Adding a New Tool

1. Implement the `core.Tool` interface
2. Place tool implementations under `agent/<agent_name>/tools/`
3. Include a schema JSON file (e.g., `mytool.schema.json`)

### Adding a New Agent

1. Create a new directory under `agent/` (e.g., `agent/advanced/`)
2. Implement the `core.AgentCore` interface
3. Include a `config.json` for agent-specific configuration
4. Add a compile-time interface check:
   ```go
   var _ core.AgentCore = (*MyAgent)(nil)
   ```

## Configuration

### Provider Config

Each provider has its own config file (e.g., `providers/qwen/qwen.json`). Use the template file as a reference:

```bash
cp providers/qwen/qwen.template.json providers/qwen/qwen.json
# Edit qwen.json with your actual credentials
```

### Agent Config

Agent configs are located under `agent/<agent_name>/config.json`. Key sections:

- `agent.prompt`: Prompt file paths and buffer size
- `agent.tool`: Tool configurations
- `session`: Session-specific settings (log path, max rounds)

## Security

- **Never commit API keys, secrets, or credentials**
- Use environment variables or local config files (gitignored) for sensitive data
- Use `<name>.template.json` files as examples with placeholder values
- If you accidentally commit a secret, rotate it immediately

## Testing

- Write tests for new interfaces using `go-mock`
- Place test files under `test/` directory
- Name test files as `<interface_name>_test.go`

## Pull Request Guidelines

1. **Keep changes focused** — one feature/fix per PR
2. **Follow existing conventions** — read surrounding code before making changes, especially interfaces under core directory.
3. **Add comments** — mark new code with `// auto-added:` and explain the purpose
4. **Test your changes** — ensure `go build ./...` and `go test ./...` pass
5. **Update documentation** — if your change affects usage or configuration

## Questions or Advices?

If you're unsure about anything or have suggestions, feel free to open an issue for discussion before starting work.
