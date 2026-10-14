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

### Running the command line app

```bash
cd cmd
go run main.go
```

#### demo

![ui display](https://github.com/David3310273/go-agent/blob/main/images/cmdui.png?raw=true)

Please refer to the [README.md](https://github.com/David3310273/go-agent/blob/main/cmd/README.md) in the `cmd` directory for more details.

### Running the example application

#### UI demo

![ui display](https://github.com/David3310273/go-agent/blob/main/images/webui.png?raw=true)

#### frontend

```bash
cd app/frontend/web
npm install
npm run dev
```

#### backend

```bash
cd app
go run main.go
```

For mode details, see README.md under `app` directory