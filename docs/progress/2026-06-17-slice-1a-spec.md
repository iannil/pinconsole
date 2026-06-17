# 切片 1a 规格说明（仓库骨架）

**状态**：in_progress
**开始**：2026-06-17
**完成**：（未完成，规格说明已完成，实施待启动）
**关联**：[`PLAN.md`](../../PLAN.md) §7（v1 切片顺序）、[`docs/project-status.md`](../project-status.md) §5（v1 切片状态）

## Context

PLAN.md §7 把 v1 切片拆为 10 个子切片（1a-1j）。本文档钉死切片 1a（仓库骨架）的具体范围、技术决策、目录布局、端口与环境变量约定，作为实施的唯一事实来源。

切片 1a 的目标：**打通"仓库骨架 + 三 app 端到端 hello world"**。无业务逻辑，但 build/dev/test/lint/CI 全套工程链路要可用，让切片 1b 起可以直接专注业务实现。

## 范围（acceptance criteria）

实施完成的判定标准：

- [ ] 仓库结构按 §"仓库布局"小节落地（所有目录与占位文件存在）
- [ ] `docker compose up -d`（默认 profile）启动 PG + Redis + MinIO + 健康检查通过
- [ ] `make dev` 同时启动 Go server（air）+ admin SPA（Vite）+ SDK playground（Vite），由 overmind 管理多进程
- [ ] `make build` 产出单二进制 `server/bin/server`（含 admin/dist + sdk/dist + landing/ embed）
- [ ] `docker compose --profile prod up -d` 启动完整堆栈（含 server 容器从 Dockerfile 构建）
- [ ] 访客访问 `http://localhost:8080/` 看到静态落地页（含 `<script src="/sdk.js">`）
- [ ] SDK 加载后 console.log 一条 hello 信息
- [ ] 运营访问 `http://localhost:8080/admin` 看到 Vue SPA hello world
- [ ] `GET /healthz` 返回 200 + JSON `{"status":"alive"}`
- [ ] `GET /readyz` 检查 PG/Redis/MinIO 连接，全部就绪返回 200，否则 503
- [ ] `make lint` 通过（golangci-lint + ESLint + Prettier + markdownlint）
- [ ] `make test` 通过（Go unit + Vitest unit + Playwright smoke）
- [ ] GitHub Actions CI（PR check）在 main 分支通过
- [ ] `commitlint` 在 commit message 上生效

## 锁定的决策（13 项）

| # | 维度 | 选择 | 备注 |
|---|---|---|---|
| 1 | JS 包管理 | pnpm workspaces | 跨 admin/visitor-sdk 共享依赖 |
| 2 | 任务运行器 | Make（顶层 Makefile） | 跨语言调度 |
| 3 | Go 版本 | 1.22+ | slog + embed + 1.22 ServeMux |
| 4 | Embed 策略 | 构建产物 embed + dev/prod 双模式 | dev 走 Vite proxy，prod 走 embed |
| 5 | server/ 结构 | `cmd/server/main.go` + `internal/` 多包 | internal 强封装 |
| 6 | 热重载 | air（Go）+ Vite HMR（admin + SDK）+ overmind | 多进程协调 |
| 7 | 配置 | env vars + .env + struct tag | caarlos0/env |
| 8 | docker-compose | 单文件 + profiles | dev 默认仅 infra，prod 加 server |
| 9 | Module path | `github.com/iannil/marketing-monitor` | 与 git user 一致 |
| 10 | DB migration | golang-migrate | CLI + lib 双形态 |
| 11 | 测试 | testify + Vitest + Playwright | 1b 起加 testcontainers |
| 12 | Linting | golangci-lint + ESLint + Prettier + markdownlint + commitlint + pre-commit | 完整工程化 |
| 13 | Docker 镜像 | multi-stage + alpine:3.19 | 约 30MB |

## 仓库布局（具体到文件）

```
marketing-monitor/
├── CLAUDE.md                          # 已存在
├── PLAN.md                            # 已存在
├── START.md                           # 已存在
├── README.md                          # 已存在
├── LICENSE                            # 已存在
├── Makefile                           # 新增：顶层任务运行器
├── docker-compose.yml                 # 新增：单文件 + profiles
├── package.json                       # 新增：pnpm workspace root
├── pnpm-workspace.yaml                # 新增：声明 admin + visitor-sdk
├── .env.example                       # 新增：环境变量文档
├── .env                               # gitignore（本地实际值）
├── .gitignore                         # 新增/更新
├── .editorconfig                      # 新增
├── .pre-commit-config.yaml            # 新增
├── commitlint.config.js               # 新增
├── markdownlint.json                  # 新增
│
├── server/                            # Go monolith
│   ├── go.mod                         # module github.com/iannil/marketing-monitor
│   ├── go.sum
│   ├── Dockerfile                     # multi-stage + alpine
│   ├── .golangci.yml
│   ├── .air.toml
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/                    # env struct tag 加载
│   │   │   └── config.go
│   │   ├── api/                       # Gin router + handlers
│   │   │   ├── router.go              # 路由注册总入口
│   │   │   ├── health.go              # /healthz /readyz
│   │   │   ├── visitor.go             # GET / GET /page/:id 服务落地页
│   │   │   ├── admin.go               # GET /admin/* 服务 admin SPA
│   │   │   └── sdk.go                 # GET /sdk.js
│   │   ├── storage/                   # PG/Redis/MinIO 适配器（连接池占位）
│   │   │   ├── postgres.go
│   │   │   ├── redis.go
│   │   │   └── minio.go
│   │   └── logging/                   # slog handler + trace_id 中间件
│   │       ├── handler.go
│   │       └── middleware.go
│   ├── migrations/                    # golang-migrate（空目录，1b 起加表）
│   │   └── .gitkeep
│   ├── embed.go                       # //go:embed admin/dist sdk/dist landing
│   └── embed_dev.go                   # dev build tag，跳过 embed
│
├── admin/                             # Vue3 SPA
│   ├── package.json
│   ├── vite.config.ts                 # HMR + proxy /api → :8080
│   ├── tsconfig.json
│   ├── index.html
│   ├── .eslintrc.cjs
│   ├── .prettierrc
│   └── src/
│       ├── main.ts                    # Vue + Pinia + Element Plus + i18n
│       ├── App.vue                    # hello world
│       ├── i18n/
│       │   ├── index.ts
│       │   ├── zh-CN.ts
│       │   └── en-US.ts
│       └── stores/                    # Pinia（空）
│           └── .gitkeep
│
├── visitor-sdk/                       # TypeScript SDK
│   ├── package.json                   # name: @marketing-monitor/visitor-sdk
│   ├── vite.config.ts                 # library mode + dev playground
│   ├── tsconfig.json
│   ├── playground/
│   │   └── index.html                 # dev playground 页面
│   └── src/
│       ├── index.ts                   # 入口 + 自动初始化
│       └── config.ts                  # 从 <script> data-* 读取
│
├── landing/                           # 静态落地页模板（v1 demo）
│   └── demo/
│       └── index.html                 # hello world + <script src="/sdk.js">
│
├── e2e/                               # Playwright 测试
│   ├── playwright.config.ts
│   └── smoke.spec.ts                  # 访客页加载 + SDK console.log
│
├── scripts/
│   ├── dev.sh                         # overmind Procfile 启动
│   └── release.sh                     # tag 触发的本地 release 助手
│
└── .github/workflows/
    ├── ci.yml                         # PR check：lint + test + build
    └── release.yml                    # tag 触发：cross-compile + Docker push ghcr
```

## 端口分配（约定）

| 服务 | 端口 |
|---|---|
| Go server | 8080 |
| Admin Vite dev | 5173 |
| SDK playground | 5174 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| MinIO API | 9000 |
| MinIO Console | 9001 |

## 环境变量（.env.example）

```bash
# Server
SERVER_PORT=8080
SERVER_ENV=dev
LOG_LEVEL=info

# PostgreSQL
PG_HOST=localhost
PG_PORT=5432
PG_USER=mm
PG_PASSWORD=mm_dev
PG_DB=marketing_monitor

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=mm_dev
MINIO_SECRET_KEY=mm_dev_secret
MINIO_BUCKET=marketing-monitor
MINIO_USE_SSL=false
```

docker-compose 中的容器版本：
- `postgres:16-alpine`
- `redis:7-alpine`
- `minio/minio:latest`

## dev 工作流（最终命令）

```bash
# 首次初始化
pnpm install
cd server && go mod download && cd ..
cp .env.example .env

# 日常开发
docker compose up -d                  # 起 infra（PG/Redis/MinIO）
make dev                              # 起 Go + admin + SDK playground（air + Vite + overmind）

# 测试
make test                             # 所有
make test-go                          # 仅 Go unit
make test-js                          # 仅 Vitest unit
make test-e2e                         # Playwright

# Lint
make lint                             # 全套
make lint-fix                         # 自动修复

# 生产构建
make build                            # 单二进制 server/bin/server
docker compose --profile prod up -d   # 完整堆栈
```

## Procfile（overmind/multinode 管理多进程）

```
server: air -server.cd server -- build/bin/server
admin: pnpm --filter @marketing-monitor/admin dev
sdk: pnpm --filter @marketing-monitor/visitor-sdk dev
```

## 推迟到 1b 的次要决策

以下决策不影响 1a 骨架落地，留给切片 1b 实施时再定：

- API 路由分组策略（按业务域 vs 按 HTTP 方法）
- Go 错误处理模式（sentinel errors vs custom error types vs errors.Is/As 链）
- SDK 初始化握手协议（session_id 分配、capabilities 协商、断线重连）
- ORM/DB 库选择（GORM / sqlc / sqlx / pgx 裸用）
- WebSocket hub 的具体路由模型（per-tenant room / per-visitor channel）
- pgx 连接池参数（max conns / idle timeout）
- Redis key 命名约定
- MinIO object key 命名约定
- trace_id 生成方式（UUID v4 / ULID / Snowflake）

## Verification

实施完成后用以下方式验证：

```bash
# 1. 仓库结构检查
find server admin visitor-sdk landing e2e scripts -type f | wc -l   # 应有非零数量

# 2. Go 编译
cd server && go build ./... && cd ..

# 3. 前端构建
pnpm install
pnpm -r build

# 4. docker-compose 启动
docker compose up -d
docker compose ps                    # 三个 infra 服务 healthy

# 5. 端到端 smoke
curl http://localhost:8080/healthz   # {"status":"alive"}
curl http://localhost:8080/sdk.js | head -1   # JS bundle 第一行
curl -I http://localhost:8080/admin  # 200
curl -I http://localhost:8080/        # 200 (landing demo)

# 6. 测试与 lint
make test
make lint

# 7. 完整生产构建
make build
ls -lh server/bin/server             # 二进制存在
docker compose --profile prod up -d --build
docker compose --profile prod ps     # 4 个服务 healthy
```

**预期**：所有命令成功退出，curl 返回期望状态码，二进制大小约 30-50MB（含前端 embed）。

## 估时

| 模式 | 估时 |
|---|---|
| Solo 全职 | 约 5-7 天（5 个工作日） |
| Solo 业余（10-15h/week） | 约 2-3 周 |

详细分阶段预算（solo 全职）：
- Day 1：仓库结构 + pnpm workspace + go.mod + Makefile + .env.example
- Day 2：docker-compose + 三 app 各自能独立 build/run
- Day 3：Go embed + admin/SDK build → 单二进制
- Day 4：health check + Playwright smoke + 三 app 联调
- Day 5：linting（golangci-lint + ESLint + Prettier + markdownlint + commitlint + pre-commit）+ GitHub Actions CI
- Day 6：Dockerfile + docker-compose prod profile + release workflow
- Day 7：buffer + 文档更新（START.md/PLAN.md 不动，更新 project-status.md 标记 1a 完成）

## Next

实施完切片 1a 后：

1. 在 `docs/progress/{date}-slice-1a-skeleton.md` 记录实施过程（按 [progress 模板](../templates/progress.md)）
2. 在 `docs/reports/completed/{date}-slice-1a-skeleton.md` 标记完成（按 [report 模板](../templates/report.md)）
3. 更新 `docs/project-status.md` §5 切片状态表（1a → completed）
4. 更新 `memory/daily/{date}.md` 追加实施日志
5. 启动切片 1b（单向最小：SDK 鼠标 → WS → admin 实时显示）

## Blockers

无。所有架构决策已锁定，可立即开工。

## Notes

- 切片 1a 不连 DB、不开 WebSocket、不集成 rrweb、不做认证——这些都在 1b 起
- 1a 的 hello world 是工程链路的"全链路 smoke"，不是产品功能
- 实施时如发现某项决策不可行，停下来与用户确认是否更新本规格——不要擅自改方案
- 本规格与 PLAN.md §7 一致；如本规格与 PLAN.md 出现冲突，PLAN.md 优先，本规格需修正
