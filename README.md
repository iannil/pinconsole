# marketing-monitor

> 开源 ToB 实时访客监控 + 运营互动 + 录像回放平台。
> 对标某商业竞品，AGPL-3.0 license，支持自托管。

## 项目状态

**v1 主干已交付**(2026-06-18 reality check + 全栈深度审计后):原 1a-1j 切片 + 收尾切片 1k/1l/1h-ui/1m 已完成,1n 进行中。深度分布详见 [`docs/project-status.md`](./docs/project-status.md) §5。

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
| **安全栈**(silent defaults fail-secure + 命令授权 + popup URL 白名单) | ✅(1k) |
| **GDPR 合规**(consent opt-in + 被遗忘权 + IP 截断 + co-browse 横幅) | ✅(1l) |
| **可观测性**(LifecycleTracker + event_type + WS trace_id) | ✅(1m) |

## 快速开始

```bash
git clone https://github.com/iannil/marketing-monitor
cd marketing-monitor

# 1. 复制环境配置
cp .env.example .env

# 2. 启动基础设施
docker compose up -d

# 3. 应用数据库 migration
for f in server/migrations/*.up.sql; do
  docker compose exec -T postgres psql -U mm -d marketing_monitor -f /dev/stdin < "$f"
done

# 4. 安装前端依赖 + 构建嵌入
pnpm install
pnpm --filter @marketing-monitor/admin build
pnpm --filter @marketing-monitor/visitor-sdk build

# 5. 构建 + 启动 release 单二进制
mkdir -p server/cmd/server/embedded/{admin,sdk,landing}
cp -r admin/dist/. server/cmd/server/embedded/admin/
cp -r visitor-sdk/dist/. server/cmd/server/embedded/sdk/
cp -r landing/. server/cmd/server/embedded/landing/
cd server && go build -tags release -o bin/server ./cmd/server
./bin/server

# 6. 访问
# 访客落地页：http://localhost:8080/
# 运营后台：http://localhost:8080/admin
```

## 生产部署

```bash
# docker compose 完整堆栈（含 server 容器）
docker compose --profile prod up -d --build
```

默认 admin 凭据（env var 可配）：
- Email: `admin@marketing-monitor.local`
- Password: `changeme123`

## 文档导航

| 文档 | 用途 |
|---|---|
| [`CLAUDE.md`](./CLAUDE.md) | Claude 工作指南 + 锁定决策 |
| [`PLAN.md`](./PLAN.md) | v1 架构 + 切片拆分（事实来源） |
| [`START.md`](./START.md) | 产品需求 + 竞品分析 |
| [`docs/project-status.md`](./docs/project-status.md) | 当前项目状态快照 |
| [`docs/reports/completed/`](./docs/reports/completed/) | 全部切片完成报告 |

## 技术栈

- **后端**：Go 1.22+ / Gin / coder/websocket / pgx / Redis / MinIO
- **前端**：Vue 3 / TypeScript / Vite / Pinia / Element Plus / Vue I18n / rrweb
- **SDK**：TypeScript / rrweb / MessagePack
- **存储**：PostgreSQL 16 / Redis 7 / MinIO
- **部署**：Docker / docker-compose / GitHub Actions

## License

AGPL-3.0 — 详见 [`LICENSE`](./LICENSE)
