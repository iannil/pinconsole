# 项目状态快照

> **此文件是 LLM 进入项目的第一份必读文档**。读完此文件，LLM 应能：
> 1. 在 60 秒内说出项目做什么
> 2. 知道当前进展
> 3. 知道下一步该做什么
> 4. 知道哪些决策不能动、哪些范围不能扩
>
> 状态变化时直接编辑本文件（rolling），不保留历史快照（用 git 历史追溯）。

**最后更新**：2026-06-17（切片 1b 规格已定，待实施）

---

## 1. 一句话定位

构建一款**实时访客监控 + 运营互动 + 录像回放**的 ToB 工具的**开源替代品**——目标对标某商业竞品，但不考虑客户获取与销售（不做计费、注册、营销页），专注技术核心。License：AGPL-3.0。

## 2. 当前阶段

**Slice 1a 已交付，Slice 1b 规格已定**

| 项 | 状态 |
|---|---|
| 产品事实来源 | ✅ [`START.md`](../START.md) |
| 架构事实来源 | ✅ [`PLAN.md`](../PLAN.md) |
| Claude 工作指南 | ✅ [`CLAUDE.md`](../CLAUDE.md) |
| 文档规范 | ✅ [`docs/standards/`](./standards/) |
| LICENSE | ✅ AGPL-3.0 |
| README | ✅ |
| 切片 1a（仓库骨架） | ✅ [完成报告](./reports/completed/2026-06-17-slice-1a-implementation.md) |
| 切片 1b（单向最小） | 📋 [规格](./progress/2026-06-17-slice-1b-spec.md)，待实施 |

## 3. 事实来源优先级（冲突时按此解析）

```
1. PLAN.md     — 架构、技术栈、切片拆分、决策理由
2. START.md    — 产品需求、竞品能力、业务上下文
3. CLAUDE.md   — Claude 工作指南（含文档/记忆/可观测性约定）
4. 本文件       — 当前状态与下一步
```

冲突场景示例：
- PLAN.md 说用 Gin，START.md 提到 "Gin 或 Go-Zero" → 用 Gin（PLAN.md 优先）
- START.md 描述"同时支持 SaaS 多租户"，CLAUDE.md 说"不做多租户" → 不做（CLAUDE.md 优先，因 START.md 是描述竞品能力而非本项目决策）

## 4. 架构决策清单（不可重新讨论）

详见 [`PLAN.md`](../PLAN.md) §3-§5。简表：

| 维度 | 决策 |
|---|---|
| 范围 | v1 切片 = 端到端最小可演示；完整对标是终局 |
| 切片策略 | 纵向切片优先（最小端到端 → 横向扩展） |
| 租户 | 单租户部署 + schema 预留 `tenant_id`（**不做多租户 SaaS**） |
| 管道 | 中心化 hub-and-spoke |
| 仓库 | Monorepo，Go embed 静态资源，单二进制 |
| License | AGPL-3.0 |
| 后端 | Go + Gin + coder/websocket + 自定义 hub |
| 存储 | PostgreSQL + Redis + MinIO |
| 前端 | Vue 3 + TypeScript + Vite + Pinia + Element Plus + Vue I18n（中英 day 1） |
| SDK | TypeScript + rrweb，构建产物 Go embed `/sdk.js` 同源分发 |
| co-browsing | rrweb 双向；rrweb 节点 ID 选择器；代填防抖 300ms |
| 截图 | 选择性（仅 canvas/WebGL/跨域 iframe 触发） |
| 录像 | 默认 30 天，可配置，GDPR 删除接口 |
| 多运营 | 1:1 锁定（claim/release） |
| 认证 | Email/password + bcrypt + HttpOnly cookie |
| 域名 | v1 仅平台域名（自定义域名是后续切片） |
| 浏览器 | Modern evergreen desktop + mobile 访客 |
| 可观测 | 仅 slog 结构化日志（暂不加 metrics/tracing/Sentry） |

如需变更任一决策，先与用户确认 → 更新 PLAN.md → 更新此文件。

## 5. v1 切片状态

详见 [`PLAN.md`](../PLAN.md) §7。

| 子切片 | 内容 | 状态 |
|---|---|---|
| [1a](./reports/completed/2026-06-17-slice-1a-implementation.md) | 仓库骨架（Go + admin SPA + SDK hello world + docker-compose） | ✅ completed |
| 1b | 单向最小（SDK 鼠标 → WS → admin 实时显示） | 📋 已规格化（[spec](./progress/2026-06-17-slice-1b-spec.md)），待实施 |
| 1c | rrweb 接入（全量采集 + 实时回放） | ⏳ pending |
| 1d | 录像归档（MinIO + PG 元数据 + 历史回放） | ⏳ pending |
| 1e | 双向通道（admin overlay → 命令 → SDK 执行） | ⏳ pending |
| 1f | 表单 + 跳转（代填 + 跳转接管 + 跨页面会话续接） | ⏳ pending |
| 1g | 弹窗 + 聊天 | ⏳ pending |
| 1h | 认证 + 多运营（login + claim/release） | ⏳ pending |
| 1i | 反爬虫（rate limit + UA + fingerprint） | ⏳ pending |
| 1j | i18n + 部署 + CI（中英双语 + docker-compose 完善 + GitHub Actions） | ⏳ pending |

**累计估时**：solo 全职约 14-17 周（3.5-4 个月）；业余约 9-12 个月。

## 6. 已识别风险

详见 [`PLAN.md`](../PLAN.md) §10。重点关注：

- **GDPR / CCPA 合规**：提交前按键监听属敏感处理，需明确访客同意流程
- **co-browsing 接管跳转可能被滥用**：需审计日志 + 访客"紧急退出"
- **rrweb 在动态 SPA 下节点 ID 不稳定**：测试矩阵需覆盖主流框架
- **AGPL 可能劝退部分企业采用**：双 license 路径预留

## 7. 下一步动作

**立即可执行**：启动切片 1b 实施（规格已定，见 [`progress/2026-06-17-slice-1b-spec.md`](./progress/2026-06-17-slice-1b-spec.md)）

具体步骤：

1. 写 3 表 migration（visitors + sessions + event_blobs）
2. 配置 sqlc，写 queries.sql 生成类型安全查询
3. 实现 hub + WS handler（`/ws/visitor` + `/ws/operator`）+ MessagePack envelope
4. SDK 实现 4 类采集器 + transport + 重连
5. admin 实现 Pinia store + useWs + 两栏 UI
6. Redis Stream flusher + MinIO 快照
7. testcontainers 集成测试 + Playwright e2e
8. 验证 4 个验收场景

切片 1a 已完成（[完成报告](./reports/completed/2026-06-17-slice-1a-implementation.md)）。

可立即开工的日常命令：

```bash
docker compose up -d      # 启动 infra（PG + Redis + MinIO）
make install-tools        # 安装 air / golangci-lint / golang-migrate（一次性）
pnpm dev                  # 启动 Go + admin + SDK playground（热重载）
make build                # release 单二进制
```

## 8. LLM 协作提示

**进入新会话时**：

1. 先读本文件（项目状态快照）
2. 再按需读 [`CLAUDE.md`](../CLAUDE.md)（工作指南）、[`PLAN.md`](../PLAN.md)（架构详情）
3. 读当日的 `memory/daily/{date}.md`（如有）
4. 读相关的 `docs/progress/` 与 `docs/reports/completed/`（最近 3 个）

**遇到冲突时**：

- 范围扩张请求（"加 X 功能"、"也支持 Y"）→ 检查是否在 v1 切片或后续切片路线（PLAN.md §8）。不在则停下来与用户确认是否调整 PLAN.md
- 架构决策冲突 → §4 列出的决策不能擅自动，先停下来
- "用户说 X 是 SaaS 多租户" → 不对，本项目明确不做多租户（START.md 描述的是竞品能力，不是本项目决策）
- 提到不存在的命令/服务/脚本（如 `npm test`、`make build`）→ 检查是否真的存在，pre-code 阶段无 build/test 命令

**写代码时**：

- 遵循 [`CLAUDE.md`](../CLAUDE.md) "LLM Friendly" 与 "可观测性开发" 章节
- 所有用户可见文案走 Vue I18n key（中英双语 day 1）
- 所有日志结构化 JSON（含 timestamp / trace_id / span_id / event_type / payload）
- 函数命名遵循 [`docs/standards/naming-conventions.md`](./standards/naming-conventions.md) §4

**写文档时**：

- 模板见 [`docs/templates/`](./templates/)
- 结构规范见 [`docs/standards/doc-structure.md`](./standards/doc-structure.md)
- 修改即记录到 `docs/progress/`，完成移到 `docs/reports/completed/`
