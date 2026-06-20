# PLAN.md — v1 切片设计

本文件是产品定位 + 架构 + 切片拆分的单一权威(grill-me 访谈后达成)。`START.md` 已并入本文 §1/§5。

## 1. 项目定位

构建竞品的**开源替代品**。不考虑客户获取与销售（不做计费、注册流程、营销页）。专注技术核心：访客实时监控 + 运营互动 + 录像回放 + 低代码页面编辑器（v1 之后）。

## 2. v1 切片范围

**v1 切片目标**：端到端最小可演示——1 个静态落地页上的访客被 1 个运营实时监控 + 全套互动 + 完整录像。覆盖竞品能力密度（除页面编辑器、Tauri 桌面端、自定义域名外）。

完整对标竞品是终局；v1 是其中可独立交付的第一个端到端切片。

## 3. 架构骨架

| 决策 | 选择 | 理由 |
|---|---|---|
| 切片策略 | 纵向切片优先 | 最早暴露集成问题；最早有 demo；强制解决端到端 hardest 问题 |
| 租户模型 | 单租户部署 + schema 预留 `tenant_id` | OSS 自托管数据自然隔离；schema 预留避免后期重构 |
| 实时管道 | 中心化 hub-and-spoke | 架构简单、可录、可审计；带宽压力可通过节流解决 |
| 仓库结构 | Monorepo，Go embed 静态资源 | 单二进制部署；OSS 自托管体验最佳 |
| License | AGPL-3.0 | 防止云厂商 SaaS 化；"OSS 替代商业产品"标准策略 |

## 4. 技术栈

### 后端

- **Go + Gin + coder/websocket + 自定义 hub**
- **PostgreSQL**（元数据：tenant/users/sessions/events 索引/messages/pages）
- **Redis**（presence、hot 缓存、rate limit 计数器）
- **MinIO**（rrweb 事件流归档 + 选择性截图）
- **slog** 结构化 JSON 日志到 stdout（暂不加 metrics/tracing）

### 前端

- **Vue 3 + TypeScript + Vite + Pinia + Element Plus**
- **Vue I18n**（中英双语 from day 1）
- **运营 Web admin**（Tauri 后期复用同一 SPA）

### 访客 SDK

- **TypeScript + Vite**，构建产物由 Go embed 至 `/sdk.js` 同源分发
- **rrweb**（标准采集：DOM 变更 + 鼠标 + 点击 + 滚动 + 失焦表单值）
- **自定义增量**（提交前按键监听 + canvas/iframe 选择性截图）
- **WebSocket** 上行（事件流）+ 下行（运营命令）

## 5. 关键技术决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 多运营并发 | 1:1 锁定（claim/release） | 避免操作冲突；模拟真实销售场景；schema 支持未来扩展 |
| co-browsing 路径 | rrweb 双向 | 访客事件流 → 运营 replayer；运营控制动作 → 访客 SDK 执行；技术栈统一 |
| 元素选择器 | rrweb 节点 ID | 不用 CSS/XPath；rrweb 原生稳定 |
| 运营代填粒度 | 防抖动 300ms | 平衡流量与体验；访客可见"打字"感；冲突简单（LWW） |
| 截图策略 | 选择性（仅 canvas/WebGL/跨域 iframe 触发，1fps WebP q70） | 默认 DOM 重建已够；带宽与存储成本最低 |
| 录像保留 | 默认 30 天，可配置，GDPR 删除接口 | SaaS 常见标准；OSS 自托管可调 |
| Replayer | rrweb-player + 外部透明 overlay 捕获事件 | 不 fork rrweb；升级友好；最干净扩展点 |
| 反爬虫 | rate limit + UA 黑名单 + 行为分析 + canvas/WebGL fingerprint | 安全/反爬虫是一等公民（见 §1）；中等深度 |
| 域名 | v1 仅平台域名 | 验证架构最快；自定义域名作为下一个切片 |
| 认证 | Email/password + bcrypt + HttpOnly cookie；WebSocket 同源依赖 cookie；MFA 可选 | Admin SPA 同源；最简安全方案 |
| 浏览器矩阵 | Modern evergreen desktop + mobile 访客；运营仅桌面 | 中国 H5 营销移动必不可少；运营桌面场景 |

## 6. 仓库布局（提案）

```
pinconsole/
├── server/                # Go monolith
│   ├── cmd/               # main.go
│   ├── internal/
│   │   ├── auth/          # JWT/cookie/session
│   │   ├── api/           # Gin REST handlers
│   │   ├── hub/           # WebSocket hub + rooms
│   │   ├── recording/     # rrweb 归档到 MinIO
│   │   ├── ratelimit/     # Redis-based
│   │   ├── antiscrape/    # UA + behavior + fingerprint
│   │   └── storage/       # PG/Redis/MinIO adapters
│   ├── migrations/        # SQL migrations
│   └── embed.go           # //go:embed admin/ landing/ sdk/
├── admin/                 # Vue3 SPA (运营端)
├── visitor-sdk/           # TypeScript SDK
├── landing/               # 静态落地页模板（后期编辑器输出）
├── docs/                  # 中英双语文档
├── docker-compose.yml     # Go + PG + Redis + MinIO
├── .github/workflows/     # CI/CD
└── LICENSE                # AGPL-3.0
```

## 7. v1 切片内的实施顺序

| # | 子切片 | 内容 | 估时（solo 全职） |
|---|---|---|---|
| 1a | 骨架 | 仓库 + docker-compose + Go "hello world" + admin SPA "hello world" + SDK "hello world" | 1 周 |
| 1b | 单向最小 | SDK 采集鼠标 → WS → 后端 hub → admin 实时显示；验证整条管道 | 2 周 |
| 1c | rrweb 接入 | SDK 集成 rrweb 全量采集 → admin 用 rrweb-player 实时回放 | 2 周 |
| 1d | 录像归档 | 后端把事件流 + 截图写 MinIO + 元数据写 PG + admin 回放历史会话 | 2 周 |
| 1e | 双向通道 | admin overlay 捕获事件 → 命令消息 → SDK 执行（高亮/点击/滚动） | 2-3 周 |
| 1f | 表单 + 跳转 | 代填（防抖动 300ms）+ 跳转接管 + 跨页面会话续接 | 2 周 |
| 1g | 弹窗 + 聊天 | 消息通道 + 持久化 | 1-2 周 |
| 1h | 认证 + 多运营 | 登录/cookie + claim/release 锁 | 1-2 周 |
| 1i | 反爬虫 | rate limit + UA + fingerprint | 1-2 周 |
| 1j | i18n + 部署 + CI | 中英双语 + docker-compose 完善 + GitHub Actions | 1 周 |

**累计估时**
- Solo 全职：约 14-17 周（3.5-4 个月）
- Solo 业余（10-15h/week）：约 9-12 个月

### §7.2 v1 加固阶段（2026-06-18 完成）

`1a-1j` 主干切片完成后，由 [`docs/audits/2026-06-18-deep-audit.md`](./docs/audits/2026-06-18-deep-audit.md) 80 条发现驱动 + 后续 e2e acceptance 反馈。所有切片 ✅ 已交付。

| 切片 | 内容 | 状态 | 报告 |
|---|---|---|---|
| 1h-ui | admin LoginView + Vue Router 守卫 | ✅ | [spec](./docs/reports/completed/2026-06-18-slice-1h-ui-spec.md) + [impl](./docs/reports/completed/2026-06-18-slice-1h-ui-implementation.md) |
| 1k | 安全阻断栈（8 个 P0：silent defaults fail-secure + 命令授权 + popup URL 白名单） | ✅ | [spec](./docs/reports/completed/2026-06-18-slice-1k-spec.md) + [impl](./docs/reports/completed/2026-06-18-slice-1k-implementation.md) |
| 1l | GDPR 合规（consent opt-in + 被遗忘权 + IP 截断 + co-browse 横幅） | ✅ | [spec](./docs/reports/completed/2026-06-18-slice-1l-spec.md) + [impl](./docs/reports/completed/2026-06-18-slice-1l-implementation.md) |
| 1m | 可观测性（LifecycleTracker + event_type + WS trace_id） | ✅ | [spec](./docs/reports/completed/2026-06-18-slice-1m-spec.md) + [impl](./docs/reports/completed/2026-06-18-slice-1m-implementation.md) |
| 1n | 测试深度 + 文档虚标修复 | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1n-implementation.md) |
| 1o | 生产硬化（TrustedProxies + WS timeout + flushSession tx + goroutine 泄漏） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1o-implementation.md) |
| 1p | LLM friendly（proto 共享 + IMPLEMENTATION_PLAN + change-safety） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1p-implementation.md) |
| 1q | 死代码 + 重复清理 | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1q-implementation.md) |
| 1r | i18n + logger 迁移（admin utils + SDK 22 处 `console.*`） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1r-implementation.md) |
| 1s | 可观测性深化（LifecycleTracker 接入关键路径） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1s-implementation.md) |
| 1t | 测试覆盖补全（logging + storage + privacy + migrations） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1t-implementation.md) |
| 1u | god files 拆分（queries.go 771 LOC → 10 文件） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1u-implementation.md) |
| 1v | 审计后续修复（migrator 统一 + GDPR DELETE + e2e webServer） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1v-implementation.md) |
| 1w | flagged session 接入（listSessions + operatorWS + replay） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1w-implementation.md) |
| 1x | 登录暴力破解防护（Redis 计数器 + 锁定 15 分钟） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1x-implementation.md) |
| 1y | visitor WS rate limit（滑动窗口 10s/500 envelope/50 MiB） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1y-visitor-ws-rate-limit.md) |
| 1z | 生产就绪度补全（i18n `@` + trace_id 端到端 + 连接池 + fail-secure） | ✅ | [impl](./docs/reports/completed/2026-06-18-slice-1z-prod-readiness-gaps.md) |
| v1-e2e | 全量 e2e acceptance + 6 regression spec + 4 production bug 修复 | ✅ | [impl](./docs/reports/completed/2026-06-18-v1-e2e-acceptance.md) |
| v1-followups | e2e 后 5 个生产 bug fix（co-browse + visitor-sdk + v1-replay × 3） | ✅ | [impl](./docs/reports/completed/2026-06-18-v1-followups.md) |

### §7.3 v1 测试深化阶段（2026-06-19~20 完成）

由 [`docs/audits/2026-06-19-test-confidence-audit.md`](./docs/audits/2026-06-19-test-confidence-audit.md) 28 T0 + 40 T1 gap + [`docs/audits/2026-06-19-test-health-audit.md`](./docs/audits/2026-06-19-test-health-audit.md) 9 项 P0/P1 驱动。所有切片 ✅ 已交付。

| 切片 | 内容 | 状态 | 报告 |
|---|---|---|---|
| 1aa | TS 测试深化（admin 64 + SDK 48） | ✅ | [impl](./docs/reports/completed/2026-06-19-slice-1aa-ts-test-deepening.md) |
| 1ab | TrustedProxies 加固（P1-5，BEHIND_REVERSE_PROXY env + validate fail-fast） | ✅ | [impl](./docs/reports/completed/2026-06-19-slice-1ab-trusted-proxies.md) |
| 1ac / 1ac-final | 28 T0 关闭 + 2 代码 bug 修复（deleteVisitor admin role + operatorWS auth） | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1ac-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1ac-implementation.md) |
| 1ad | 40 T1 关闭（lifecycle 接线契约 + warn + is_flagged + replay HTTP 行为级） | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1ad-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1ad-implementation.md) |
| 1ae | 测试健康度加固（9 项 P0+P1：mutation score 71.4%→100%, e2e flaky 20%→0%） | ✅ | [audit](./docs/audits/2026-06-19-test-health-audit.md) + [spec](./docs/reports/completed/2026-06-19-slice-1ae-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1ae-implementation.md) |
| 1af | 测试健康度深化（R3 续做 6 group：23 新行为级测试，D1 ~55%→~75%） | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1af-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1af-implementation.md) |
| 1ag | api handler 行为级测试（auth+replay） | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md) |
| 1ah | claim/chat handler 行为级测试 | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-implementation.md) |
| 1aj | 修 parseSince 负数 + ws_ratelimit flaky | ✅ | [spec](./docs/reports/completed/2026-06-19-slice-1aj-followup-bugs-spec.md) + [impl](./docs/reports/completed/2026-06-19-slice-1aj-followup-bugs-implementation.md) |
| 1ai / 1ai-b | storage repo PG 集成测试（user/session + visitor/command/event_blob） | ✅ | [spec 1ai](./docs/reports/completed/2026-06-19-slice-1ai-storage-repo-tests-spec.md) + [impl 1ai](./docs/reports/completed/2026-06-19-slice-1ai-storage-repo-tests-implementation.md) + [spec 1ai-b](./docs/reports/completed/2026-06-20-slice-1aib-storage-repos-b-spec.md) + [impl 1ai-b](./docs/reports/completed/2026-06-20-slice-1aib-storage-repos-b-implementation.md) |
| 1ai-c ~ 1ai-h | api/storage handler 接口化重构 + happy path 测试（AuthHandler / ClaimHandler / ChatHandler / CommandHandler / requireClaimOwnership / SessionHandler） | ✅ | [1aic](./docs/reports/completed/2026-06-20-slice-1aic-auth-handler-interface-implementation.md) + [1aid](./docs/reports/completed/2026-06-20-slice-1aid-me-logout-happy-path-implementation.md) + [1aie](./docs/reports/completed/2026-06-20-slice-1aie-claim-chat-interface-implementation.md) + [1aif](./docs/reports/completed/2026-06-20-slice-1aif-command-handler-interface-implementation.md) + 1ai-g/1ai-h（仅 daily 记录，无独立 impl） |

> **接口化模式 PoC**：1ai-c ~ 1ai-h 在 5 个 handler 验证了"接口字段 + mock + happy path"模式，累计 +75 测试，api 包覆盖实测 28.7%（commit 自报 38.2% 是更高口径）。

## 8. v1 之后的切片

按优先级排序：

1. **页面编辑器**（拖拽 / 低代码 / JSON schema → Go 模板渲染）—— 切片 2-3
2. **自定义域名**（DNS 验证 + Let's Encrypt ACME + Host-header 路由）—— 1-2 周
3. **Tauri 桌面端**（Win + Mac，复用 admin SPA）—— 1 个月
4. **反爬加固**（CAPTCHA + honeypot + 动态类名/ID）—— 2-3 周
5. **SSO / SAML / OIDC**（企业）—— 2 周
6. **分析仪表盘**（漏斗 / 热力图 / 停留）—— 1-2 个月
7. **多租户**（激活预留的 tenant_id，配额/隔离）—— 取决于商业化方向

**已完成收尾**：

- ✅ **1h-ui admin 登录 UI**（v1 收尾，2026-06-18 拆出并完成）：LoginView + Vue Router 守卫 + useAuth composable + 401 拦截重定向 + `/api/auth/me` 集成。spec 来源：`docs/reports/completed/2026-06-17-slice-1h-spec.md` 决策 #5。

## 9. 未敲定的实施层细节

切片推进过程中再决定，不影响架构骨架：

- DB schema 字段级设计（visitor/session/event/message 各字段）
- 截图压缩参数微调
- 运营 replayer 的 overlay 实现（CSS pointer-events / 自定义事件代理）
- WebSocket 重连策略（指数退避 + 本地缓冲 + size limit）
- 多运营"接管排队"UX（transfer 优先 vs 抢占式）
- 文档策略（docs as code + 中英双语生成）
- 测试策略（unit + integration + e2e 比例）
- 录像回放的性能优化（virtual list / seek 索引）
- SDK 初始化握手协议（session_id 分配 / capabilities 协商）
- 后端 hub 水平扩展（Redis pub/sub 触发条件）

## 10. 已识别的风险

| 风险 | 缓解 |
|---|---|
| 提交前按键监听在 GDPR / CCPA 下属敏感处理 | 需要访客同意流程；明确告知；提供关闭开关 |
| co-browsing 接管跳转可能被滥用（强制跳转至付款页） | 审计日志；运营权限分级；访客"紧急退出"快捷键 |
| MinIO 存储成本（录像 30 天 × 500 并发） | 选择性截图降低压力；保留期可配置；分区冷存储 |
| rrweb 在复杂 SPA（React/Vue 动态重渲染）下节点 ID 不稳定 | 测试矩阵覆盖主流框架；fallback 到 CSS selector |
| 1:1 锁定在多运营协作场景下不够 | v1 后补"角色分层"（主运营 + 观察者） |
| AGPL 可能劝退部分企业采用 | 双 license 路径预留 |
