# Go-agent

An interface-implemented, multi-layered agent framework. Simple but not easy.

## Architecture

Using param injection from outside app to start eventual tool call

![architecture](https://github.com/David3310273/go-agent/blob/main/images/layer.png?raw=true)

## Design Principles

- Multi-layered architecture, clear boundary and simple organization of directories, easy to understand and extend.
- Minimal skills should be provided. In this framework, only one tool provided with two skills.
- - accquire tools using MCP server skill
- - extend backgrounds with knowledge base skill. 
- **Framework process control rather than LLM process or prompt control**. For example, control the process and context using framework logic, rather than often modify the prompt with hint words or rely on the LLM's response or param input.

## How to start

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

### Make request to the agent

For example:

```bash
curl -X POST -H "Content-Type: application/json" -d '{"question": "hello"}' http://localhost:8080/v1/agent/ask
```

For mode details, see README.md under `app` directory