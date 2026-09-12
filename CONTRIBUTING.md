# Contributing to go-agent

Thank you for your interest in contributing to go-agent! This document provides guidelines and conventions for contributing to this project.

## Project Overview

go-agent is an interface-implemented, multi-layered agent framework written in Go. The project follows a layered architecture with clear separation of concerns.

### Main entities

- App: user oriented service or app, such as server, command or web application.
- AgentCore: entrypoint to app, mainly for status management and session management, with customized system prompt, history, knowledge base and tools
- Session: a conversation between user and agent, including questions, context(injected from AgentCore) and own history
- Provider: model adapter between framework and specific llms
- Context: agent/session runtime context. including prompt, history, knowledge base, available providers and tools
- Tool: local tools for agent.
- Sandbox: decorator of tool calls
- Benchmark: single agent metrics watcher, should not belong to any agents.

### Architecture Layers

- **core/**: Framework interfaces and core logic (AgentCore, Session, Provider, Tool, etc.)
- **agent/**: Agent implementations (e.g., `simple/`)
- **providers/**: LLM provider adapters (e.g., `qwen/`)
- **app/**: User-facing applications (server, CLI, web app)

### Design Principles

- **Param injection** over struct-embedded variables
- **Interface composition** with single-responsibility interface units
- **core/** must not import specific implementation packages, for example, from agent or app implementations

## Contributing Guidelines

1. **Start from supporting other model providers if you're interested**, such as claude, openai, etc.
2. **Providing more thoughts on architecture or design partterns**. Such as what do you think of sandbox, MCP server and how to insert these components into the framework.
3. **Don't have to pay more attention on the simple app**, that is just an example of this framework. Of course, good advices and implementations are also welcome.

### Pull Request Guidelines

1. for the specific PR, **using following format for commit message**, and don't forget to add changelog.
    ```bash
    <type>(<scope>): <description>
    # e.g. "feat(agent/core/app/session/provider...): add new provider for agent"
    ``` 
2. **Keep changes focused** — one feature/fix per PR
3. **Follow existing conventions** — read surrounding code before making changes, especially interfaces under core directory.
4. **Test your changes** — ensure `go build ./...` and `go test ./...` pass
5. **Update documentation** — if your change affects usage or configuration


### Running Tests

1. unit test: 
```bash
go test ./...
```

2. integration test: using simple app to test the whole framework.

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
- AI coding is also welcome. **But don't submit your local ai coding Prompt or any related ai coding tool configs**.
- Keep related helper functions/package in the same file/directory
- Use `template.json` files for critical config examples (never commit real credentials)

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

## Questions or Advices?

If you're unsure about anything or have suggestions, feel free to open an issue for discussion before starting work.
