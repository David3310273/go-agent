# =============================================================================
# Go-Agent Web Server - Multi-Stage Docker Build
# =============================================================================
# 构建: docker build -t go-agent-server .
# 运行: docker run -p 8080:8080 -v $(pwd)/config.json:/app/config.json go-agent-server
# =============================================================================

# ---- Stage 1: Build Frontend ----
FROM node:20-alpine AS frontend-builder
WORKDIR /src
COPY app/frontend/web/package*.json ./
RUN npm ci --only=production
COPY app/frontend/web/ ./
RUN npm run build

# ---- Stage 2: Build Go Binary ----
FROM golang:1.25-alpine AS go-builder

# 安装基础依赖
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# 先复制依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译 Web 服务 (静态链接)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /go-agent-server ./app

# ---- Stage 3: Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 用户
RUN addgroup -g 1000 goagent && \
    adduser -D -u 1000 -G goagent goagent

WORKDIR /app

# 从构建阶段复制二进制
COPY --from=go-builder /go-agent-server .

# 复制前端静态文件
COPY --from=frontend-builder /src/dist ./app/frontend/web/dist

# 复制 agent 配置模板和 prompt
COPY agent/simple/prompt.md ./agent/simple/prompt.md
COPY agent/simple/config.template.json ./agent/simple/config.template.json
COPY agent/simple/tools/ ./agent/simple/tools/

# 创建数据目录
RUN mkdir -p /agent/simple/sessionLogs /agent/simple/sessionMemories && \
    chown -R goagent:goagent /agent

USER goagent

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/v1/session/list || exit 1

ENTRYPOINT ["./go-agent-server"]