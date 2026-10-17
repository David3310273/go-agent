# Go-Agent Build System
# =============================================================================
# 独立部署: make build-cmd / make build-server
# 交叉编译: make build-linux / make build-darwin
# Docker:   make docker-build / make docker-up
# =============================================================================

# 变量定义
MODULE        := github.com/David3310273/go-agent
BUILD_DIR     := bin
CMD_ENTRY     := ./cmd
APP_ENTRY     := ./app
SERVER_NAME   := go-agent-server
CLI_NAME      := go-agent-cli

# 版本信息
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME    := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 链接器参数 (去除调试信息减小体积)
LDFLAGS       := -s -w \
	-X '$(MODULE)/core.Version=$(VERSION)' \
	-X '$(MODULE)/core.BuildTime=$(BUILD_TIME)' \
	-X '$(MODULE)/core.GitCommit=$(GIT_COMMIT)'

# =============================================================================
# 默认目标
# =============================================================================
.PHONY: all
all: build

# =============================================================================
# 构建
# =============================================================================
.PHONY: build build-cmd build-server build-all

build: build-cmd build-server

build-cmd:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(CLI_NAME) $(CMD_ENTRY)
	@echo "✅ CLI binary: $(BUILD_DIR)/$(CLI_NAME)"

build-server:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(SERVER_NAME) $(APP_ENTRY)
	@echo "✅ Server binary: $(BUILD_DIR)/$(SERVER_NAME)"

# =============================================================================
# 交叉编译
# =============================================================================
.PHONY: build-linux build-darwin

build-linux: GOOS=linux
build-linux: GOARCH=amd64
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(CLI_NAME)-$(GOOS)-$(GOARCH) $(CMD_ENTRY)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(SERVER_NAME)-$(GOOS)-$(GOARCH) $(APP_ENTRY)
	@echo "✅ Linux binaries built in $(BUILD_DIR)/"

build-darwin: GOOS=darwin
build-darwin: GOARCH=amd64
build-darwin:
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(CLI_NAME)-$(GOOS)-$(GOARCH) $(CMD_ENTRY)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(SERVER_NAME)-$(GOOS)-$(GOARCH) $(APP_ENTRY)
	@echo "✅ macOS binaries built in $(BUILD_DIR)/"

# =============================================================================
# 运行 (开发测试)
# =============================================================================
.PHONY: run-cmd run-server dev dev-quick

run-cmd:
	cd agent/simple && ../../$(BUILD_DIR)/$(CLI_NAME)

run-server:
	cd agent/simple && ../$(BUILD_DIR)/$(SERVER_NAME)

# ---- 一键启动前后端 (Ctrl+C 同时停止) ----
FE_DIR := app/frontend/web

dev:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo " 后端: http://localhost:8080"
	@echo " 前端: http://localhost:5173"
	@echo " Ctrl+C to stop both"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@trap 'kill 0; exit' INT TERM; \
	go run ./app & \
	cd $(FE_DIR) && npm run dev & \
	wait

dev-quick:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo " 后端: http://localhost:8080"
	@echo " 前端: http://localhost:5173"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@trap 'kill 0; exit' INT TERM; \
	go run ./app 2>&1 & \
	cd $(FE_DIR) && npx vite --host & \
	wait

# =============================================================================
# Helm
# =============================================================================
HELM_DIR      := deploy/helm/go-agent
HELM_RELEASE  := go-agent
HELM_NS       := go-agent

.PHONY: helm-lint helm-template helm-install helm-upgrade helm-uninstall helm-status helm-history helm-rollback

helm-lint:
	helm lint $(HELM_DIR)

helm-template:
	helm template $(HELM_RELEASE) $(HELM_DIR) --namespace $(HELM_NS) --debug

helm-install:
	helm upgrade --install $(HELM_RELEASE) $(HELM_DIR) \
		--namespace $(HELM_NS) \
		--create-namespace \
		--wait \
		--timeout 5m
	@echo "✅ Helm release installed. Checking status..."
	@helm status $(HELM_RELEASE) --namespace $(HELM_NS)

helm-upgrade:
	helm upgrade $(HELM_RELEASE) $(HELM_DIR) \
		--namespace $(HELM_NS) \
		--wait \
		--timeout 5m

helm-uninstall:
	helm uninstall $(HELM_RELEASE) --namespace $(HELM_NS)
	@echo "✅ Helm release uninstalled."

helm-status:
	helm status $(HELM_RELEASE) --namespace $(HELM_NS)
	@echo ""
	@echo "=== Pods ==="
	kubectl get pods -n $(HELM_NS) -o wide

helm-history:
	helm history $(HELM_RELEASE) --namespace $(HELM_NS)

helm-rollback:
	@read -p "Rollback to revision: " rev; \
	helm rollback $(HELM_RELEASE) $$rev --namespace $(HELM_NS)

# =============================================================================
# Docker
# =============================================================================
REGISTRY   ?= ghcr.io/David3310273/go-agent
IMAGE_TAG  ?= latest

.PHONY: docker-build docker-build-server docker-build-cli docker-push

docker-build: docker-build-server docker-build-cli

docker-build-server:
	docker build -f Dockerfile -t $(REGISTRY)-server:$(IMAGE_TAG) .
	@echo "✅ Server image: $(REGISTRY)-server:$(IMAGE_TAG)"

docker-build-cli:
	docker build -f Dockerfile.cli -t $(REGISTRY)-cli:$(IMAGE_TAG) .
	@echo "✅ CLI image: $(REGISTRY)-cli:$(IMAGE_TAG)"

docker-push:
	docker push $(REGISTRY)-server:$(IMAGE_TAG)
	docker push $(REGISTRY)-cli:$(IMAGE_TAG)
	@echo "✅ Images pushed to $(REGISTRY)"

# =============================================================================
# 测试 & 代码检查
# =============================================================================
.PHONY: test vet lint

test:
	go test ./... -v -count=1

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found, install: brew install golangci-lint"; exit 1)
	golangci-lint run ./...

# =============================================================================
# 清理
# =============================================================================
.PHONY: clean

clean:
	rm -rf $(BUILD_DIR)
	@echo "✅ Cleaned build artifacts"

# =============================================================================
# 帮助
# =============================================================================
.PHONY: help

help:
	@echo "Go-Agent Build System"
	@echo "====================="
	@echo ""
	@echo "构建:"
	@echo "  make build          构建所有二进制"
	@echo "  make build-cmd      仅构建 CLI 程序"
	@echo "  make build-server   仅构建 Web 服务"
	@echo "  make build-linux    交叉编译 Linux amd64"
	@echo "  make build-darwin   交叉编译 macOS amd64"
	@echo ""
	@echo "运行 (本地开发):"
	@echo "  make dev            一键启动前后端 (go run + vite)"
	@echo "  make dev-quick      同上 (跳过 npm run, 直达 vite)"
	@echo "  make run-cmd        运行 CLI (go run)"
	@echo "  make run-server     运行 Web 服务 (go run)"
	@echo ""
	@echo "Docker (镜像构建):"
	@echo "  make docker-build       构建所有镜像"
	@echo "  make docker-build-server 仅构建 Server 镜像"
	@echo "  make docker-build-cli    仅构建 CLI 镜像"
	@echo "  make docker-push         推送镜像到 Registry"
	@echo ""
	@echo "Helm (K8s 生产部署):"
	@echo "  make helm-lint       检查 Helm Chart 语法"
	@echo "  make helm-template   预渲染模板 (不实际部署)"
	@echo "  make helm-install    安装/升级 Helm Release"
	@echo "  make helm-upgrade    仅升级已存在的 Release"
	@echo "  make helm-uninstall  卸载 Helm Release"
	@echo "  make helm-status     查看 Release 状态"
	@echo "  make helm-history    查看部署历史"
	@echo "  make helm-rollback   回滚到指定版本"
	@echo ""
	@echo "Kubernetes (裸YAML):"
	@echo "  make helm-template  预览生成的 K8s YAML (helm template)"
	@echo "Docker:"
	@echo "  make docker-build       构建所有镜像"
	@echo "  make docker-build-server 仅构建 Server 镜像"
	@echo "  make docker-build-cli    仅构建 CLI 镜像"
	@echo "  make docker-push         推送镜像到 Registry"
	@echo ""
	@echo "质量:"
	@echo "  make test           运行测试"
	@echo "  make vet            代码静态分析"
	@echo ""
	@echo "其他:"
	@echo "  make clean          清理构建产物"