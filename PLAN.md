# PLAN.md — v1 切片设计

本文件是 grill-me 访谈后达成的共识。`START.md` 是产品需求来源（来自竞品拆解）；本文是 v1 实施的技术决策与切片拆分。

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
| 反爬虫 | rate limit + UA 黑名单 + 行为分析 + canvas/WebGL fingerprint | START.md "一等公民"；中等深度 |
| 域名 | v1 仅平台域名 | 验证架构最快；自定义域名作为下一个切片 |
| 认证 | Email/password + bcrypt + HttpOnly cookie；WebSocket 同源依赖 cookie；MFA 可选 | Admin SPA 同源；最简安全方案 |
| 浏览器矩阵 | Modern evergreen desktop + mobile 访客；运营仅桌面 | 中国 H5 营销移动必不可少；运营桌面场景 |

## 6. 仓库布局（提案）

```
marketing-monitor/
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

## 8. v1 之后的切片

按优先级排序：

1. **页面编辑器**（拖拽 / 低代码 / JSON schema → Go 模板渲染）—— 切片 2-3
2. **自定义域名**（DNS 验证 + Let's Encrypt ACME + Host-header 路由）—— 1-2 周
3. **Tauri 桌面端**（Win + Mac，复用 admin SPA）—— 1 个月
4. **反爬加固**（CAPTCHA + honeypot + 动态类名/ID）—— 2-3 周
5. **SSO / SAML / OIDC**（企业）—— 2 周
6. **分析仪表盘**（漏斗 / 热力图 / 停留）—— 1-2 个月
7. **多租户**（激活预留的 tenant_id，配额/隔离）—— 取决于商业化方向

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
