# =============================================================================
# Go-Agent Web Server - Multi-Stage Docker Build
# =============================================================================
# 构建: DOCKER_BUILDKIT=1 docker build -t go-agent-server .
# 运行: docker run -p 8080:8080 -v $(pwd)/config.json:/app/config.json go-agent-server
# =============================================================================

# ---- Stage 1: Build Frontend ----
FROM node:22-alpine AS frontend-builder

# Use domestic mirror for faster downloads in China
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /src
COPY app/frontend/web/package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    --mount=type=cache,target=/src/node_modules \
    npm install --prefer-offline
COPY app/frontend/web/ ./
RUN npm run build

# ---- Stage 2: Build Go Binary ----
FROM golang:1.26-alpine AS go-builder

# Use domestic mirrors: Go module proxy + Alpine package registry
ENV GOPROXY=https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy dependency files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only Go source directories (exclude frontend, CLI, etc. to prevent cache invalidation)
COPY app/ ./app/
COPY core/ ./core/
COPY providers/ ./providers/
COPY mcp/ ./mcp/
COPY embedders/ ./embedders/
COPY knowledgebase/ ./knowledgebase/
COPY storage/ ./storage/
COPY sandbox/ ./sandbox/
COPY tools/ ./tools/
COPY agent/ ./agent/
COPY benchmarker/ ./benchmarker/

# Build web server (static linking, stripped)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /go-agent-server ./app

# ---- Stage 3: Runtime ----
FROM alpine:3.20

# Use domestic mirror: Alpine package registry
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# Create non-root user
RUN addgroup -g 1000 goagent && \
    adduser -D -u 1000 -G goagent goagent

WORKDIR /app

# Copy binary from build stage
COPY --from=go-builder /go-agent-server .

# Copy frontend static assets
COPY --from=frontend-builder /src/dist ./app/frontend/web/dist

# Copy agent config templates and prompt
COPY agent/simple/prompt.md ./agent/simple/prompt.md
COPY agent/simple/config.template.json ./agent/simple/config.template.json
COPY agent/simple/tools/ ./agent/simple/tools/

# Create data directories
RUN mkdir -p /agent/simple/sessionLogs /agent/simple/sessionMemories && \
    chown -R goagent:goagent /agent

USER goagent

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/v1/session/list || exit 1

ENTRYPOINT ["./go-agent-server"]