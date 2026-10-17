# Go-Agent Deployment Guide

## Architecture Overview

Go-Agent consists of two independently deployable components sharing the same Go module:

| Component | Entry | Type | Artifact |
|-----------|-------|------|----------|
| **CLI** | `cmd/main.go` | Standalone binary | `go-agent-cli` |
| **Web Server** | `app/main.go` | HTTP microservice | `go-agent-server` |

Both depend on shared packages under `core/`, `agent/`, and `providers/`.

```
cmd/main.go ──┐
              ├── core/ ── agent/simple/ ── providers/qwen/
app/main.go ──┘
```

---

## Project Structure

```
go-agent/
├── cmd/                    # CLI application entry
│   ├── app/                #   CommandApp logic
│   ├── config/             #   Config loader
│   ├── controller/         #   Command parser
│   └── ui/                 #   Bubble Tea TUI
├── app/                    # Web server entry
│   ├── server/             #   Gin HTTP server + router
│   ├── controller/         #   HTTP handlers
│   ├── services/           #   Business logic
│   └── frontend/web/       #   React SPA (Vite)
├── core/                   # Shared interfaces and types
├── agent/simple/           # Agent implementation (worker pool)
├── providers/qwen/         # AI model provider
├── deploy/
│   └── helm/go-agent/      # Helm chart for K8s deployment
├── Dockerfile              # Server multi-stage build
├── Dockerfile.cli          # CLI multi-stage build
├── Makefile                # Build automation
├── go.mod / go.sum
└── DEPLOY.md               # This file
```

---

## Local Development

### Prerequisites

- Go 1.25+
- Node.js 20+ (for frontend)
- npm

### Quick Start

```bash
# One command to start both backend and frontend
make dev
# Backend: http://localhost:8080
# Frontend: http://localhost:5173 (with HMR via Vite)

# Or start individually
make run-cmd          # CLI only
make run-server       # Web server only
```

The frontend Vite dev server proxies `/v1/*` API requests to `http://localhost:8080`, so no CORS issues during development.

---

## Build

### Local Binaries

```bash
make build            # Build both CLI + server → bin/
make build-cmd        # CLI only
make build-server     # Server only
```

Output:
```
bin/
├── go-agent-cli      # ~20 MB, single binary
└── go-agent-server   # ~50 MB, single binary (embeds frontend dist/)
```

### Cross-Compilation

```bash
make build-linux      # Linux amd64
make build-darwin     # macOS amd64
```

---

## CI/CD Pipeline

### GitHub Actions Workflow (`.github/workflows/ci-build.yml`)

Triggered on push to `main`.

```
git push main
  │
  ▼
┌──────────────────────────────────────────────┐
│ Job: detect-changes                          │
│ ────────────────────────────────────────     │
│ core/** agent/** providers/** changed?       │
│   → build both CLI + Server                 │
│ cmd/** only?  → build CLI only              │
│ app/** only?  → build Server only           │
└────────────┬─────────────────────────────────┘
             │
      ┌──────┴──────┐
      ▼              ▼
  build-cli      build-server
  (Dockerfile   (Dockerfile)
   .cli)         3-stage build
      │              │
      └──────┬───────┘
             ▼
       push to ghcr.io
```

### Docker Images

| Image | Registry |
|-------|----------|
| Server | `ghcr.io/David3310273/go-agent-server:latest` |
| CLI | `ghcr.io/David3310273/go-agent-cli:latest` |

Tags: `latest`, `main`, `<git-sha-short>`

### Docker Build Details

#### CLI (`Dockerfile.cli`)
- **Stage 1**: `golang:1.25-alpine` — `go mod download`, `go build ./cmd`
- **Stage 2**: `alpine:3.20` — binary + prompt.md + config template + tools/
- Layer caching: `go.mod`/`go.sum` copied first → `go mod download` cached
- `.dockerignore` excludes `node_modules/`, `.git/`, `bin/`, `logs/`, etc.

#### Server (`Dockerfile`)
- **Stage 1**: `node:20-alpine` — `npm ci`, `npm run build` → dist/
- **Stage 2**: `golang:1.25-alpine` — `go mod download`, `go build ./app`
- **Stage 3**: `alpine:3.20` — binary + dist/ + prompt.md + config + tools/
- Non-root user, health check, static linking (`CGO_ENABLED=0`)

### Manual Build

```bash
make docker-build           # Build both images
make docker-build-server    # Server only
make docker-build-cli       # CLI only
make docker-push            # Push to registry

# Custom tag
IMAGE_TAG=v1.2.0 make docker-build-server
```

---

## Kubernetes Deployment (Helm)

### Helm Chart Structure

```
deploy/helm/go-agent/
├── Chart.yaml
├── values.yaml                     # Default configuration
└── templates/
    ├── _helpers.tpl                # Shared template functions
    ├── namespace.yaml
    ├── configmap.yaml              # Application config (config.json)
    ├── server-deployment.yaml      # Go backend Deployment
    ├── server-service.yaml         # Go backend Service (ClusterIP, sessionAffinity)
    ├── nginx-deployment.yaml       # Nginx reverse proxy + static files
    ├── nginx-service.yaml          # Nginx Service
    ├── nginx-configmap.yaml        # Nginx configuration
    ├── ingress.yaml                # External ingress
    ├── hpa.yaml                    # Horizontal Pod Autoscaler
    └── pvc.yaml                    # Persistent volumes (optional)
```

### Resources Created in K8s

```
helm install go-agent ./deploy/helm/go-agent --namespace go-agent
  │
  ├─ ConfigMap: go-agent-config          # config.json
  ├─ Deployment: go-agent-server         # replicas: 3 (default)
  │    └─ Pod (Go backend, port 8080)
  ├─ Service: go-agent-server            # ClusterIP :8080
  │    └─ sessionAffinity: ClientIP      # SSE sticky sessions
  ├─ Deployment: go-agent-nginx          # replicas: 2 (default)
  ├─ Service: go-agent-nginx             # ClusterIP :80
  ├─ Ingress                             # External traffic → nginx
  └─ HPA                                 # CPU > 70% → scale 2~10
```

### Deploy

```bash
# Install / upgrade
make helm-install

# With custom values
helm upgrade --install go-agent ./deploy/helm/go-agent \
  --namespace go-agent \
  --set server.replicas=5 \
  --set server.image.tag=v1.2.0 \
  --wait

# Status
make helm-status

# Rollback
make helm-rollback     # Interactive: enter revision number

# Uninstall
make helm-uninstall

# Template preview (no deploy)
make helm-template
```

### Runtime Architecture

```
Browser
  │
  ▼
┌─────────────────────────────────────────┐
│ Ingress (TLS termination)               │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│ Nginx Service (ClusterIP :80, 2 pods)   │
│ ├─ / → Static files (dist/)             │
│ ├─ /v1/* → API proxy to server          │
│ └─ proxy_buffering off (SSE friendly)   │
└──────────────┬──────────────────────────┘
               │ upstream: go-agent-server:8080
               │ least_conn balancing
               ▼
┌─────────────────────────────────────────┐
│ Server Service (ClusterIP :8080, 3 pods)│
│ └─ sessionAffinity: ClientIP            │
│    └─ AgentCore worker pool (per pod)   │
└──────────────┬──────────────────────────┘
               │
               ▼
         AI Model API
```

---

## Scale & Operate

```bash
# Manual scale
kubectl scale deployment/go-agent-server -n go-agent --replicas=5

# View logs
kubectl logs -n go-agent -l app.kubernetes.io/component=server --tail=100 -f

# Check HPA status
kubectl get hpa -n go-agent
```

---

## AgentCore Worker Pool

AgentCore uses a configurable goroutine worker pool inside each server pod.

Relevant config in `agent/simple/config.json`:
```json
{
  "workerCount": 4
}
```

Each worker pulls from a shared `Question` channel and routes requests to their target session. Same-session requests are serialized per session via internal mutex channels. This provides parallel dispatch across different sessions without race conditions.

---

## Configuration

### Web Server (`config.json`)

```json
{
  "rootPath": "/app",
  "timeZone": "Asia/Shanghai",
  "language": "zh-CN",
  "logPath": "app/logs/app-%s.log",
  "maxWaitingRequest": 10,
  "maxWaitingSeconds": 30,
  "requestQueueLength": 10,
  "service": {
    "port": 8080,
    "maxCloseWaitingSeconds": 60
  }
}
```

### Agent (`agent/simple/config.json`)

```json
{
  "agent": { ... },
  "session": { ... },
  "model": [{ ... }],
  "workerCount": 4
}
```

---

## Summary

| Scenario | Command |
|----------|---------|
| Local dev (frontend + backend) | `make dev` |
| Build binaries | `make build` |
| Build & push Docker images | `make docker-build && make docker-push` |
| Cross-compile for Linux | `make build-linux` |
| Deploy to K8s | `make helm-install` |
| Scale pods | `kubectl scale deployment/... --replicas=N` |
| Rollback release | `make helm-rollback` |
| View K8s status | `make helm-status` |