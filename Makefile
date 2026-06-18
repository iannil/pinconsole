.PHONY: help dev dev-server build build-server build-frontend test test-go test-js test-e2e \
       lint lint-go lint-frontend lint-md format \
       docker-up docker-down docker-prod docker-logs \
       migrate-up migrate-down migrate-new \
       install-tools clean

# 默认目标
.DEFAULT_GOAL := help

# 路径变量
SERVER_DIR  := server
SERVER_BIN  := $(SERVER_DIR)/bin/server
FRONTEND_DIRS := admin visitor-sdk

# Go 工具
GO          := go
GOLANGCI    := golangci-lint
AIR         := air
MIGRATE     := migrate

# Docker
DOCKER      := docker
COMPOSE     := $(DOCKER) compose

# 颜色（仅在 TTY 时启用）
ifneq (,$(findstring xterm,$(TERM)))
	C_RESET  := \033[0m
	C_GREEN  := \033[32m
	C_CYAN   := \033[36m
	C_YELLOW := \033[33m
else
	C_RESET  :=
	C_GREEN  :=
	C_CYAN   :=
	C_YELLOW :=
endif

help: ## 显示所有可用目标
	@echo "$(C_GREEN)marketing-monitor$(C_RESET) - 切片 1a 仓库骨架"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "$(C_CYAN)开发$(C_reset):"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(C_YELLOW)%-20s$(C_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "提示: 首次运行先 'make install-tools' 安装 Go 工具链。"

# ============================================================
# 开发
# ============================================================

dev: ## 启动开发模式（Go + admin + sdk playground）
	pnpm dev

dev-server: ## 仅启动 Go server（air 热重载）
	cd $(SERVER_DIR) && $(AIR)

# ============================================================
# 构建
# ============================================================

build: build-frontend build-server ## 构建全部（前端 + Go 二进制）

build-frontend: ## 构建前端（admin + sdk）
	pnpm -r --if-present build

build-server: ## 构建 Go 单二进制（含前端 embed）
	pnpm build:admin && pnpm build:sdk
	cd $(SERVER_DIR) && CGO_ENABLED=0 $(GO) build -o bin/server -tags release ./cmd/server
	@echo "$(C_GREEN)✓$(C_RESET) 二进制产出: $(SERVER_BIN)"

build-server-dev: ## 构建 Go dev 二进制（不含前端 embed，仅 API）
	cd $(SERVER_DIR) && $(GO) build -o bin/server-dev -tags dev ./cmd/server

# ============================================================
# 测试
# ============================================================

test: test-go test-js ## 运行全部单元测试（Go + JS）

test-go: ## Go 单元测试
	cd $(SERVER_DIR) && $(GO) test -race -cover ./...

test-js: ## JS 单元测试（Vitest）
	pnpm -r --if-present test

test-e2e: ## E2E 测试（Playwright）
	pnpm --filter @marketing-monitor/e2e test

# ============================================================
# Lint 与格式化
# ============================================================

lint: lint-go lint-frontend lint-md ## 运行全部 lint

lint-go: ## Go lint（golangci-lint）
	cd $(SERVER_DIR) && $(GOLANGCI) run ./...

lint-frontend: ## 前端 lint（ESLint + Prettier check）
	pnpm -r --if-present lint

lint-md: ## Markdown lint
	pnpm lint:md

format: ## 格式化（gofmt + goimports + Prettier）
	cd $(SERVER_DIR) && $(GO) fmt ./...
	pnpm -r --if-present format

# ============================================================
# Docker
# ============================================================

docker-up: ## 启动基础设施（PG + Redis + MinIO）
	$(COMPOSE) up -d
	@echo "$(C_GREEN)✓$(C_RESET) 基础设施已启动。检查: $(COMPOSE) ps"

docker-down: ## 停止并清理容器
	$(COMPOSE) down

docker-prod: ## 启动完整生产堆栈（含 server 容器）
	$(COMPOSE) --profile prod up -d --build

docker-logs: ## 跟踪容器日志
	$(COMPOSE) logs -f

# ============================================================
# DB Migration
# ============================================================
# 1v:server 启动时自动跑内嵌 migrator（migrations.go），
# 单一事实源 + schema_migrations 表 + pg_advisory_lock。
# 不再用 golang-migrate CLI（其 schema_migrations 形状 version+dirty 与 server 不兼容）。
# 详见 docs/audits/2026-06-18-1k-1u-regression.md §4 新-2。

PG_URL ?= postgres://mm:mm_dev@localhost:5432/marketing_monitor?sslmode=disable

migrate-up: ## [DEPRECATED] 由 server 启动时自动跑；保留入口仅为兼容
	@echo "$(C_YELLOW)⚠️  migrate-up 已废弃：server 启动时自动应用 migrations。$(C_RESET)"
	@echo "  正确做法：./ops.sh start  或  ./ops.sh restart"
	@echo "  强制用 golang-migrate CLI：make migrate-up-legacy"
	@exit 1

migrate-up-legacy: ## [INTERNAL] 强制用 golang-migrate CLI（与 server 内嵌 migrator 不兼容）
	$(MIGRATE) -path $(SERVER_DIR)/migrations -database "$(PG_URL)" up

migrate-down: ## [DEPRECATED] 由 server 启动时自动跑；破坏性，需逃生门
	@echo "$(C_YELLOW)⚠️  migrate-down 已废弃：破坏性操作。$(C_RESET)"
	@echo "  重置 dev DB：./ops.sh reset  （DROP + server 自动 re-migrate）"
	@echo "  强制用 golang-migrate CLI down：make migrate-down-legacy MM_ALLOW_DESTRUCTIVE_MIGRATE=1"
	@exit 1

migrate-down-legacy: ## [INTERNAL] 强制用 golang-migrate CLI down（破坏性！）
	@if [ "$$MM_ALLOW_DESTRUCTIVE_MIGRATE" != "1" ]; then \
		echo "$(C_YELLOW)⚠️  migrate-down-legacy 会 DROP TABLE 并丢失数据！$(C_RESET)"; \
		echo "5 秒内按 Ctrl+C 取消，或设 MM_ALLOW_DESTRUCTIVE_MIGRATE=1 跳过此提示。"; \
		for i in 5 4 3 2 1; do echo -n "$$i "; sleep 1; done; \
		echo ""; \
	fi
	$(MIGRATE) -path $(SERVER_DIR)/migrations -database "$(PG_URL)" down 1

migrate-new: ## 创建新迁移：make migrate-new NAME=add_users
	@test -n "$(NAME)" || (echo "用法: make migrate-new NAME=<kebab-case-name>"; exit 1)
	@cd $(SERVER_DIR)/migrations && migrate create -ext sql -dir . -seq $(NAME)

# ============================================================
# 工具安装
# ============================================================

install-tools: ## 安装 Go 工具链（air, golangci-lint, migrate）
	@echo "$(C_CYAN)安装 Go 工具...$(C_RESET)"
	$(GO) install github.com/air-verse/air@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "$(C_GREEN)✓$(C_RESET) 工具已安装到 $$($$(go env GOPATH)/bin)。确保 PATH 含该路径。"

# ============================================================
# 清理
# ============================================================

clean: ## 清理构建产物
	rm -rf $(SERVER_DIR)/bin
	rm -rf admin/dist visitor-sdk/dist
	find . -type d -name node_modules -prune -exec rm -rf {} + 2>/dev/null || true
	@echo "$(C_GREEN)✓$(C_RESET) 已清理构建产物"
