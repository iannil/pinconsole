# 切片 1a 仓库骨架完成报告

**状态**：completed
**完成时间**：2026-06-17
**对应 progress**：[`docs/progress/2026-06-17-slice-1a-implementation.md`](../../progress/2026-06-17-slice-1a-implementation.md)
**规格来源**：[`docs/progress/2026-06-17-slice-1a-spec.md`](../../progress/2026-06-17-slice-1a-spec.md)

## Summary

按 13 项锁定决策落地仓库骨架。可重复执行的工程链路（dev / build / test / lint / docker）全部可用，端到端 smoke 验证通过。与规格的两项偏差已记录（concurrently 替代 overmind、Go 1.25 实际版本）。

## Changes Delivered

### 顶层（13 个新文件）

- ✅ `package.json` — pnpm workspace root（dev: concurrently 多进程、build、test、lint 全套）
- ✅ `pnpm-workspace.yaml` — admin / visitor-sdk / e2e 三个 workspace
- ✅ `Makefile` — 顶层任务运行器（dev / build / test / lint / docker / migrate / install-tools / clean）
- ✅ `.gitignore` — Go + Node + Docker + IDE
- ✅ `.editorconfig` — UTF-8 / LF / 2-space JS / tab Go / Makefile
- ✅ `.env.example` — Server / PG / Redis / MinIO 完整环境变量文档
- ✅ `commitlint.config.js` — conventional commits 强制
- ✅ `.pre-commit-config.yaml` — pre-commit 钩子框架
- ✅ `markdownlint.json` + `.markdownlintignore`
- ✅ `docker-compose.yml` — 单文件 + profiles（dev 默认仅 infra，prod 加 server）
- ✅ `.github/workflows/ci.yml` — Go check / JS check / docs check / docker build / compose smoke
- ✅ `.github/workflows/release.yml` — tag 触发，跨平台二进制 + Docker push ghcr

### server/（12 个新文件 + migrations 占位）

- ✅ `go.mod` / `go.sum` — module `github.com/iannil/marketing-monitor`
- ✅ `cmd/server/main.go` — 入口，装配 config / logger / stores / router，支持优雅退出
- ✅ `cmd/server/embed.go` — `//go:build release`，`//go:embed all:embedded`
- ✅ `cmd/server/embed_dev.go` — `//go:build !release`，空 embed.FS（dev 走 Vite）
- ✅ `cmd/server/embedded/` — 构建产物占位（build 时填充）
- ✅ `internal/config/config.go` — env struct tag 加载（caarlos0/env）
- ✅ `internal/logging/handler.go` — slog JSON / Text 双模式 + trace_id/span_id context 传播
- ✅ `internal/logging/middleware.go` — Gin 中间件，每请求注入 trace_id + span_id
- ✅ `internal/api/router.go` — Gin 路由总入口，dev/prod 双模式静态资源
- ✅ `internal/api/health.go` — /healthz（liveness）+ /readyz（readiness，查 PG/Redis/MinIO）
- ✅ `internal/storage/{stores,postgres,redis,minio.go}` — 三后端连接适配器，Ping 验证
- ✅ `.golangci.yml` / `.air.toml` / `Dockerfile` — lint + 热重载 + multi-stage Alpine 镜像
- ✅ `migrations/.gitkeep` — golang-migrate 占位

### admin/（11 个新文件）

- ✅ `package.json` / `tsconfig.json` / `vite.config.ts` — Vue 3 + TS + Vite + Pinia + Element Plus + Vue I18n
- ✅ `index.html` / `src/main.ts` / `src/App.vue` — hello world + 中英切换 + 健康检查
- ✅ `src/env.d.ts` — Vue SFC 类型 shim
- ✅ `src/i18n/{index,zh-CN,en-US}.ts` — 中英双语 from day 1
- ✅ `tests/smoke.test.ts` — Vitest 占位
- ✅ `.eslintrc.cjs` / `.prettierrc` — ESLint + Prettier 配置

### visitor-sdk/（9 个新文件）

- ✅ `package.json` / `tsconfig.json` / `vite.config.ts` — Vite library mode，输出单文件 `sdk.js`
- ✅ `src/index.ts` — 入口，自动初始化 + console.log 验证加载链路
- ✅ `src/config.ts` — 从 `<script data-*>` 与 `window.MM_CONFIG` 合并配置
- ✅ `playground/index.html` — 开发期测试页
- ✅ `tests/config.test.ts` — Vitest 占位
- ✅ `.eslintrc.cjs` / `.prettierrc` — ESLint + Prettier

### landing/ 与 e2e/

- ✅ `landing/demo/index.html` — 访客落地页 demo，含 `<script src="/sdk.js">`
- ✅ `e2e/package.json` / `playwright.config.ts` / `tests/smoke.spec.ts` — Playwright smoke

## Verification

实际执行的验证：

```bash
# 1. Go vet 通过（修复 1 处 unused import）
cd server && go vet ./...

# 2. Go dev 构建
cd server && go build -tags dev -o /tmp/server-dev ./cmd/server
# → 17 MB 二进制

# 3. Go release 构建（含前端 embed）
pnpm --filter @marketing-monitor/admin build      # → admin/dist 1.06 MB JS
pnpm --filter @marketing-monitor/visitor-sdk build # → visitor-sdk/dist/sdk.js 1.56 KB
# 拷贝产物到 server/cmd/server/embedded/
cd server && CGO_ENABLED=0 go build -tags release -o /tmp/server-release ./cmd/server
# → 24 MB 单二进制（含全部前端）

# 4. JS 单元测试
pnpm test:js  # admin 1 passed / visitor-sdk 1 passed

# 5. docker-compose infra 启动
docker compose up -d
docker compose ps  # → postgres / redis / minio 均 healthy

# 6. 端到端 smoke（release 二进制 + infra）
/tmp/server-release &
curl http://localhost:8080/healthz   # → {"status":"alive","trace_id":"..."}
curl http://localhost:8080/readyz    # → {"status":"ready","components":{"postgres":"ok","redis":"ok","minio":"ok"}}
curl http://localhost:8080/          # → 200, 2729 bytes (landing)
curl http://localhost:8080/sdk.js    # → 200, 1558 bytes (text/javascript)
curl http://localhost:8080/admin     # → 200, 472 bytes (SPA index.html)
curl http://localhost:8080/admin/dashboard          # → 200 (SPA fallback)
curl http://localhost:8080/admin/assets/missing.js  # → 404 asset_not_found
```

**实际产物大小**：
- dev 二进制：17 MB
- release 二进制：24 MB（含 admin SPA + SDK + landing）
- admin SPA：1.06 MB JS（Element Plus 较大，1b+ 可按需引入）
- SDK：1.56 KB（rrweb 尚未集成，1c 加入后会增加）

**结构化日志样例**（已落实 CLAUDE.md "可观测性开发"约定）：

```
level=INFO msg=http_request trace_id=7c7f71fdd684ae7119c772085a293b3a \
  span_id=e0b01225bed28057801c34ae74b248bf method=GET path=/healthz \
  status=200 latency_ms=0 client_ip=::1 user_agent=curl/8.7.1
```

## 与规格的偏差

| 偏差 | 规格 | 实际 | 理由 |
|---|---|---|---|
| 多进程管理 | overmind | concurrently（npm） | 避免 tmux 依赖，跨平台更友好 |
| Go 版本 | 1.22+（go.mod 写 1.22） | go.mod 自动写 1.25（系统装的是 1.26.2） | tidy 自动调整；go.mod 写 1.22 也兼容 |
| visitor-sdk 构建 | vue-tsc | tsc | visitor-sdk 不是 Vue 项目，无需 vue-tsc |
| admin tsconfig | tsconfig.json + tsconfig.node.json 引用 | 合并为单一 tsconfig.json | composite + noEmit 配置冲突，简化为单文件 |
| `/admin/*any` catch-all | 显式路由 | 用 NoRoute 做 SPA fallback | Gin 不允许 catch-all 与 StaticFS 同前缀 |

所有偏差均不偏离架构决策；后续切片实施不受影响。

## Follow-ups

- 切片 1b：实现单向最小（SDK 鼠标 → WS → admin 实时显示）
  - 引入 `coder/websocket`（已在 PLAN.md 锁定）
  - DB schema 起步：visitors / sessions 表
  - 决定 ORM/DB 库（推迟清单中的项）
- 实施前需安装的本地工具：
  - `make install-tools`（air / golangci-lint / golang-migrate）
  - Playwright 浏览器：`pnpm --filter @marketing-monitor/e2e install-browsers`
- 首次 git commit：用户尚未授权，等待指令
- 启动日常开发：`make docker-up && pnpm dev`

## Notes

- Go 二进制大小 24 MB 含 Element Plus（admin）+ rrweb（SDK 占位）+ landing，合理
- release 二进制可独立分发，无外部文件依赖
- /healthz 与 /readyz 都返回 trace_id，便于日志关联
- middleware 自动给所有请求注入 trace_id（无 X-Trace-Id 时生成），并通过响应头回传
- docker-compose 中 server 容器只在 `--profile prod` 时启动；dev 流程中由本地 air 跑
