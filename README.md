# Go-agent

An interface-implemented, multi-layered agent framework. Simple but not easy.

## Architecture

Using param injection from outside app to start eventual tool call

![architecture](https://github.com/David3310273/go-agent/blob/main/images/layer.png?raw=true)

### Design Principles

- Param injection rather than variables in struct
- Only composition on final interface, interface unit should have single responsibility.

## Highlights

- Multi-layered architecture, clear boundary and simple organization of directories, easy to understand
- Easy to extend and customize based on interface implementation

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
curl -X POST -H "Content-Type: application/json" -d '{"question": "hello"}' http://localhost:8080/v1/ask
```

For mode details, see README.md under `app` directory