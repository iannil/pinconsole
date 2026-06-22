# · pinconsole

> **你的访客，你的数据。** / *Your visitors, your data.*

开源 ToB 实时访客监控 + 运营互动 + 录像回放平台。AGPL-3.0，自托管，数据从不出门——竞品 SaaS 的开源替代。

[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-0F766E.svg)](./LICENSE)
[![e2e: 65 passed](https://img.shields.io/badge/e2e-65%20passed%20%2F%200%20failed-15803D.svg)](./docs/reports/completed/2026-06-18-v1-e2e-acceptance.md)
[![v1: shipped](https://img.shields.io/badge/v1-shipped-0F766E.svg)](./docs/project-status.md)
[![i18n: zh/en](https://img.shields.io/badge/i18n-zh%20%2F%20en-0E7490.svg)](#)

---

## 这是什么

**pinconsole** 是 ToB 实时访客互动平台的 AGPL-3.0 开源替代品。三个不可妥协的设计决策：

- **数据主权**：所有访客行为、运营对话、录像会话存在**你自己的** PostgreSQL / Redis / MinIO。无第三方调用、无外部依赖。
- **AGPL-3.0 强 copyleft**：任何修改必须开源，**云厂商无法拿去做 SaaS**——license 层的硬保护。
- **标准栈，无锁定**：Go 1.22 + Vue 3 + PostgreSQL 16 + Redis 7 + MinIO。每一层都是行业标准，Schema 在你手里。

**v1 已交付**：实时监控（rrweb 全量）+ 双向协同（cursor/click/scroll/fill/navigate）+ 录像回放 + 弹窗聊天 + 多运营 claim 锁 + 反爬虫 + GDPR + 中英双语 + Docker 一键部署。90+ commits，65 e2e 测试全绿。

> **决策者？** 看官网 → [pinconsole.example.com](https://pinconsole.example.com)（占位——部署后替换为真实域名）。
> **咨询**（评估替代 / 自托管 / 定制 / 合规）→ [官网表单](https://pinconsole.example.com#consult) 或 [GitHub Issues](https://github.com/iannil/pinconsole/issues)。
> **工程师？** 下面是快速开始。

---

## 项目状态

**v1 主干完全收口 + 测试信心补全**(2026-06-20):1a-1z 全切片 + 1k-1z 安全/GDPR/可观测加固 + 1aa-1ai-h 测试深化与接口化重构 + e2e acceptance(65 测试全绿)+ 5 个 followup fix + admin flagged UI/prod-mode CI + marketing-monitor → pinconsole 全量重命名,共 90+ commits 交付。深度分布与下一步详见 [`docs/project-status.md`](./docs/project-status.md)。

| 能力 | 状态 |
|---|---|
| 实时访客监控(rrweb 全量采集) | ✅ |
| 运营 Web admin(Vue3 + Element Plus + LoginView + Vue Router 守卫) | ✅ |
| co-browsing 双向控制(cursor/click/scroll/fill/navigate) | ✅ |
| 录像归档 + 历史回放(MinIO + rrweb-player) | ✅ |
| 弹窗推送 + 双向即时聊天 | ✅ |
| 认证 + 多运营 claim/release 锁(后端 claim SET NX + Lua release) | ✅ |
| 反爬虫(rate limit + UA 黑名单 + 行为分析 + fingerprint) | ✅ |
| 中英双语 i18n | ✅ |
| Docker Compose 一键部署(prod profile fail-secure 凭证) | ✅ |
| GitHub Actions CI/CD | ✅ |
| **安全栈**(silent default fail-secure + 命令授权 + popup URL 白名单) | ✅(1k) |
| **GDPR 合规**(consent opt-in + 被遗忘权 + IP 截断 + co-browse 横幅) | ✅(1l) |
| **可观测性**(LifecycleTracker + event_type + WS trace_id) | ✅(1m) |
| **测试深化与接口化重构**(api/storage handler 接口化 + happy path 覆盖) | ✅(1ai-c~1ai-h) |

## 快速开始

```bash
git clone https://github.com/iannil/pinconsole
cd pinconsole

# 1. 复制环境配置
cp .env.example .env

# 2. 启动基础设施（用 Make 一键启动 + 等 PG ready）
make docker-up
# 或手动:docker compose up -d postgres redis minio

# 3. 安装前端依赖 + 构建嵌入
make build-frontend
# 或手动:pnpm install && pnpm --filter @pinconsole/admin build && ...

# 4. 构建 + 启动 release 单二进制(server 启动时自动应用 migrations)
make build
./server/bin/pinconsole-server
# 或手动:cd server && go build -tags release -o bin/pinconsole-server ./cmd/server

# 5. 访问
# 访客落地页：http://localhost:8080/
# 运营后台：http://localhost:8080/admin
```

**Make 命令清单**(详见 `Makefile`):`make help` / `make docker-up` / `make test-go` / `make coverage-go` / `make check`(lint + 单测) / `make verify`(含 e2e)。

### 老开发者迁移(2026-06-20 rename 重构遗漏修复)

如果本地 PG volume 已有 `marketing_monitor` 数据库(rename 重构前的旧名),改 `.env` 后 docker compose up **不会**自动迁移。两种迁移方式:

```bash
# 方式 A:清空 volume 重来(数据丢失,仅开发环境)
make docker-down-v
make docker-up
make migrate

# 方式 B:保留数据,手动迁移
docker compose exec -T postgres psql -U mm -d postgres -c "CREATE DATABASE pinconsole OWNER mm;"
make migrate
# 然后可选:导出 marketing_monitor 数据 → 导入 pinconsole → DROP DATABASE marketing_monitor
```

详见 [`docs/reports/completed/2026-06-20-slice-0-env-rename-fix-implementation.md`](./docs/reports/completed/2026-06-20-slice-0-env-rename-fix-implementation.md)。

## 生产部署

```bash
# docker compose 完整堆栈（含 server 容器）
docker compose --profile prod up -d --build
```

默认 admin 凭据（env var 可配）：
- Email: `admin@pinconsole.local`
- Password: 部署时由 `ADMIN_PASSWORD` 强制要求(prod 模式拒绝 `changeme123`)

## v1 已知限制(部署前必读)

v1 是端到端最小可演示切片,以下限制在生产部署前需自行评估:

1. **单实例 hub(不支持横向扩展)**
   WebSocket 路由基于进程内 `map`(`server/internal/hub/hub.go`)。
   多实例部署(2+ server behind LB)会导致 visitor 和 operator 连到不同实例后互不可见,
   系统不会报错(静默表现坏)。如需多实例,需引入 Redis Pub/Sub 或 NATS 作为消息总线。

2. **500 WS/房间并发未压测**
   PLAN.md 把"500 WS/房间"作为设计目标驱动单租户/hub-and-spoke/1:1 锁定决策,
   但 v1 **未做实际压测**。默认 `PG_MAX_CONNS=25` / `REDIS_POOL_SIZE=50`
   是经验值,实际容量需部署方自行压测验证。

3. **OSS 项目不提供生产拓扑**
   docker-compose `prod` profile 仅作为参考,实际生产部署(VM / k8s / 反代 / TLS /
   备份 / 监控 / 日志聚合 / 资源限制)由部署方自行决定。本仓库只保证:
   - dev/CI 路径可重复运行
   - release 二进制 fail-secure(默认拒绝弱配置,详见 [`docs/audits/`](./docs/audits/))
   - `/healthz` + `/readyz` 提供依赖健康检查

4. **Trace_id 端到端传播(1z 已补全)**
   operator browser → server → visitor SDK → server → operator 形成完整 trace_id 闭环:
   - admin SPA 每次 REST 调用注入 `X-Trace-Id` 头(`admin/src/api/client.ts`)
   - visitor SDK 收到 operator command 时缓存 trace_id,后续 10 个事件或 5 秒内继承(`visitor-sdk/src/transport/ws.ts`)
   - server 端 TraceMiddleware + WS handler 还原 ctx trace_id

   详见 [`docs/reports/completed/2026-06-18-slice-1z-prod-readiness-gaps.md`](./docs/reports/completed/2026-06-18-slice-1z-prod-readiness-gaps.md)。

## 文档导航

| 文档 | 用途 |
|---|---|
| [`docs/project-status.md`](./docs/project-status.md) | **LLM 入口** — 当前状态、架构决策、下一步、协作提示 |
| [`docs/README.md`](./docs/README.md) | docs/ 目录与切片报告索引 |
| [`CLAUDE.md`](./CLAUDE.md) | Claude 工作指南 + 锁定决策 |
| [`PLAN.md`](./PLAN.md) | v1 产品定位 + 架构 + 切片拆分（事实来源） |

## 技术栈

- **后端**：Go 1.22+ / Gin / coder/websocket / pgx / Redis / MinIO
- **前端**：Vue 3 / TypeScript / Vite / Pinia / Element Plus / Vue I18n / rrweb
- **SDK**：TypeScript / rrweb / MessagePack
- **存储**：PostgreSQL 16 / Redis 7 / MinIO
- **部署**：Docker / docker-compose / GitHub Actions

## License

AGPL-3.0 — 详见 [`LICENSE`](./LICENSE)
