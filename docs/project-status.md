# 项目状态快照

> **此文件是 LLM 进入项目的第一份必读文档**。读完此文件，LLM 应能：
> 1. 在 60 秒内说出项目做什么
> 2. 知道当前进展
> 3. 知道下一步该做什么
> 4. 知道哪些决策不能动、哪些范围不能扩
>
> 状态变化时直接编辑本文件（rolling），不保留历史快照（用 git 历史追溯）。

**最后更新**：2026-06-18(全栈审计 → 1k 安全 + 1l GDPR 合规已交付 🟢;1h-ui / 1m / 1n 待启动)

---

## 1. 一句话定位

构建一款**实时访客监控 + 运营互动 + 录像回放**的 ToB 工具的**开源替代品**——目标对标某商业竞品，但不考虑客户获取与销售（不做计费、注册、营销页），专注技术核心。License：AGPL-3.0。

## 2. 当前阶段

**v1 主干已交付:T0 安全栈 + GDPR 合规已修复(1k + 1l 🟢),T1+ 部分待补**

最新进展(2026-06-18):

- ✅ 全栈深度审计完成 → [`docs/audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)(80 条发现,P0:13)
- ✅ 1k-security-blockers 🟢 verified-deep:修 8 个 P0(silent defaults 全套 fail-secure + command/chat/claim 授权 + claim TOCTOU/Lua + popup URL 白名单 + migrations embed + auto up)
- ✅ **1l-privacy-gdpr 🟢 verified-deep**:修 P0-5/P0-6(consent opt-in + 被遗忘权级联删除 + IP /24+64 截断 + co-browse 横幅 + GC 扩展);**部署欧盟/加州阻断项已解除**
- ⚠️ 审计 badge 复核:7 切片原 🟢 实际 🟡(1a/1b/1c/1e/1f/1g/1i),仅 1d/1j 真到 🟢;**未修改 project-status.md §5 的 badge 表**(留给 1n-test-depth 切片统一处理)
- ⏳ 1h-ui 待启动(admin LoginView + Vue Router 守卫)
- ⏳ 1m-observability / 1n-test-depth 待启动

切片深度分布(v1 主干,基于审计实测 + 1n/1o/1p/1q 修复):

- 🟢 verified-deep ×10(1d, 1j, 1k, 1l, 1h-ui, 1m, 1n, 1o, 1p, **1q**)
- 🟡 verified-shallow ×7(1a, 1b, 1c, 1e, 1f, 1g, 1i)
- 🔴 implemented-unverified ×1(1h-backend)
- 全部切片已交付

A 阶段升级详情:

- **1i 🟡→🟢**:接线 BehaviorTracker(原死代码)、加 Go 单测覆盖 rate limit 429 + 3 个启发式 + UA 黑名单、扩展 builtinBannedUAs、e2e 真查 PG fingerprint 持久化
- **1j 🔴→🟢**:抽硬编码中文 20+ 处为 i18n key、加语言切换按钮(原无 UI)、加 i18n/docker-prod/CI/README 4 个真 e2e、修 Dockerfile go 版本 bug(1.22 → 1.25)
- **1h U2**:拆为 1h-backend(本已 done 的后端部分) + 1h-ui(待启动,登录 UI + 路由守卫)

深度判定标准详见 [`standards/verification-depth.md`](./standards/verification-depth.md)。

下一步优先级:**B 文档对齐(✅)→ A 浅测补深 + 实施补全(✅)→ C 设计漏洞(待启动)**。

| 项 | 状态 |
|---|---|
| 产品事实来源 | ✅ [`START.md`](../START.md) |
| 架构事实来源 | ✅ [`PLAN.md`](../PLAN.md) |
| Claude 工作指南 | ✅ [`CLAUDE.md`](../CLAUDE.md) |
| 文档规范 | ✅ [`docs/standards/`](./standards/) |
| LICENSE | ✅ AGPL-3.0 |
| README | ✅ |
| 切片 1a(仓库骨架) | 🟢 [报告](./reports/completed/2026-06-17-slice-1a-implementation.md) |
| 切片 1b(单向最小) | 🟢 [报告](./reports/completed/2026-06-17-slice-1b-implementation.md) |
| 切片 1c(rrweb 接入) | 🟢 [报告](./reports/completed/2026-06-17-slice-1c-implementation.md) |
| 切片 1d(录像归档) | 🟢 [报告](./reports/completed/2026-06-17-slice-1d-implementation.md) |
| 切片 1e(双向通道) | 🟢 [报告](./reports/completed/2026-06-17-slice-1e-implementation.md) |
| 切片 1f(表单 + 跳转) | 🟢 [报告](./reports/completed/2026-06-17-slice-1f-implementation.md) |
| 切片 1g(弹窗 + 聊天) | 🟢 [报告](./reports/completed/2026-06-17-slice-1g-implementation.md) |
| 切片 1h(认证 + 多运营) | 🔴 [报告](./reports/completed/2026-06-17-slice-1h-implementation.md) |
| 切片 1i(反爬虫) | 🟢 [报告](./reports/completed/2026-06-17-slice-1i-implementation.md) |
| 切片 1j(i18n + 部署 + CI) | 🟢 [报告](./reports/completed/2026-06-17-slice-1j-implementation.md) |

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

## 5. v1 切片状态(2026-06-18 reality check 后)

详见 [`PLAN.md`](../PLAN.md) §7。**深度判定标准**: [`standards/verification-depth.md`](./standards/verification-depth.md)。

图例:🟢 verified-deep / 🟡 verified-shallow / 🔴 implemented-unverified

| 子切片 | 内容 | 深度 | 备注 |
|---|---|---|---|
| [1a](./reports/completed/2026-06-17-slice-1a-implementation.md) | 仓库骨架 | 🟡 | 1n 降级:vacuous assertion + test.skip 静默跳过 |
| [1b](./reports/completed/2026-06-17-slice-1b-implementation.md) | 单向最小 | 🟡 | 1n 降级:SDK 重连 / MinIO checksum 未覆盖 |
| [1c](./reports/completed/2026-06-17-slice-1c-implementation.md) | rrweb 接入 | 🟡 | 1n 降级:脱敏 vacuous truth(已修)+ 截图/韧性未覆盖 |
| [1d](./reports/completed/2026-06-17-slice-1d-implementation.md) | 录像归档 | 🟢 | live→historical 真验证 |
| [1e](./reports/completed/2026-06-17-slice-1e-implementation.md) | 双向通道 | 🟡 | 1n 降级:3 处静默跳过(已修)+ cursor/ESC/审计表未真验 |
| [1f](./reports/completed/2026-06-17-slice-1f-implementation.md) | 表单 + 跳转 | 🟡 | 1n 降级:4 处静默跳过(已修) |
| [1g](./reports/completed/2026-06-17-slice-1g-implementation.md) | 弹窗 + 聊天 | 🟡 | 1n 降级:4 处静默跳过(已修)+ 双向聊天未真验 |
| [1h](./reports/completed/2026-06-17-slice-1h-implementation.md) | 认证 + 多运营(后端) | 🔴 | spec 决策 #5 login UI 未实施;1h-ui 已补 |
| [1h-ui](./reports/completed/2026-06-18-slice-1h-ui-implementation.md) | admin LoginView + 守卫 | 🟢 | 修 1h spec 决策 #5 |
| [1i](./reports/completed/2026-06-17-slice-1i-implementation.md) | 反爬虫 | 🟡 | 1n 降级:关键 Go 测试曾 flaky(已修)+ e2e 仅 dev 模式 |
| [1j](./reports/completed/2026-06-17-slice-1j-implementation.md) | i18n + 部署 + CI | 🟢 | A 阶段升级真验证 |
| [1k](./reports/completed/2026-06-18-slice-1k-implementation.md) | 安全阻断栈 | 🟢 | 修审计 T0:8 个 P0 |
| [1l](./reports/completed/2026-06-18-slice-1l-implementation.md) | GDPR 合规 | 🟢 | 修 P0-5/P0-6 |
| [1m](./reports/completed/2026-06-18-slice-1m-implementation.md) | 可观测性 | 🟢 | LifecycleTracker + WS trace_id |
| [1n](./reports/completed/2026-06-18-slice-1n-implementation.md) | 测试深度 + 文档虚标修复 | 🟢 | 修 P0-9/10/11/12 + 7 切片 badge 降级 + e2e 静默跳过改 strict assertion |
| [1o](./reports/completed/2026-06-18-slice-1o-implementation.md) | 生产硬化 | 🟢 | 修 P1-5/6/7/8:TrustedProxies + WS WriteTimeout=0 + flushSession 补偿事务 + operatorWS goroutine 泄漏 |
| [1p](./reports/completed/2026-06-18-slice-1p-implementation.md) | LLM friendly | 🟢 | packages/proto 共享包 + IMPLEMENTATION_PLAN.md + change-safety.md + naming-conventions 语言惯例差异 |
| [1q](./reports/completed/2026-06-18-slice-1q-implementation.md) | 死代码 + 重复清理 | 🟢 | 删 6 处死代码 + queries.sql + Element Plus(bundle -940KB)+ e2e/helpers + room.publish 加日志 |

> 1n 完成后 badge 复核:1a/1b/1c/1e/1f/1g/1i 已降级 🟡(基于审计 §5 实测);1d/1j/1k/1l/1h-ui/1m/1n 维持 🟢;1h-backend 维持 🔴。深度细节见 [`audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)。

**累计估时**：solo 全职约 14-17 周（3.5-4 个月）；业余约 9-12 个月。

## 6. 已识别风险

详见 [`PLAN.md`](../PLAN.md) §10 + [2026-06-18 全栈深度审计](./audits/2026-06-18-deep-audit.md)(80 条发现,P0:13/P1:27/P2:26/P3:14):

- **🔴 P0 文档虚标**:README/docs/README/1j 报告 三处独立虚标 v1 已完成(P0-10/11/12)→ 待文档对齐批次
- **🔴 P0 1i 测试 flaky**:`TestRateLimitMiddleware_Triggers429` 包内跑时 FAIL(P0-9)→ 1n-test-depth 切片
- **🟡 P1 7 切片 badge 虚标**:1a/1b/1c/1e/1f/1g/1i 实际 🟡 但 §5 标 🟢 → 1n-test-depth 切片
- **🟡 P1 可观测性合规度 ~25%**:无 LifecycleTracker、event_type 未实现、trace_id 在 WS 链路断裂(P1-15/16)→ 1m-observability 切片
- **🟡 P1 LLM Friendly 欠债**:`IMPLEMENTATION_PLAN.md` 缺失、三方 proto 手写无 codegen、变更安全策略零落地 → 1m 或独立切片
- **rrweb 在动态 SPA 下节点 ID 不稳定**:测试矩阵需覆盖主流框架
- **AGPL 可能劝退部分企业采用**:双 license 路径预留

**已修复(1k-security-blockers 🟢 + 1l-privacy-gdpr 🟢,2026-06-18)**:
- ~~P0-1 SERVER_ENV=dev 默认 + AuthMiddleware dev bypass~~ → 默认改 prod + 编译 tag 隔离
- ~~P0-2 默认 admin changeme123~~ → AdminPassword required + prod 拒绝 changeme123
- ~~P0-3 command/chat/claim 无 user_id 授权~~ → handler 层 requireClaimOwnership
- ~~P0-4 claim TOCTOU race + UUID parse~~ → SET NX + uuid.Parse + Lua release
- ~~P0-5 GDPR 按键监听无 consent flow~~ → SDK opt-in banner + mm.setConsent() + PG visitor_consents 表
- ~~P0-6 GDPR 缺 erasure + GC 只清 1/5 表~~ → DELETE /api/privacy/visitor/:fingerprint 级联删 + GC 扩展清 5/6 表
- ~~P0-7 docker-compose prod 回退 dev 凭证~~ → `${VAR:?required}` 必填
- ~~P0-8 popup action_url javascript: 注入~~ → 双重 scheme 白名单
- ~~P0-13 migrate-down 无保护~~ → prompt + 5s 倒计时 + 逃生门
- ~~P0-14 schema_migrations 未在部署路径使用~~ → embed + auto up + advisory lock
- ~~P1-1 bcrypt cost 10 < 12~~ → BCRYPT_COST 默认 12
- ~~P1-2 session cookie Secure=false~~ → prod 模式 Secure=true
- ~~P1-9 PG sslmode=disable 硬编码~~ → PG_SSLMODE env(默认 prefer)
- ~~GDPR Art.22 co-browsing 不透明~~ → SDK co-browse 接管横幅 + 退出按钮
- ~~IP 数据最小化~~ → IPv4 /24 + IPv6 /64 截断,GDPR Recital 26 不再是个人数据

## 7. 下一步动作

按优先级 B → A → C:

### B 文档对齐(✅ 完成,2026-06-18)

- ✅ reality check(静态 + 单测 + e2e 39/39)
- ✅ 锁定深度判定 R2 rubric → `docs/standards/verification-depth.md`
- ✅ 锁定切片状态三级标定(🟢/🟡/🔴)
- ✅ 1a-1j spec + implementation 全部移到 `docs/reports/completed/`
- ✅ 完成报告加深度 badge + 叙述免责 disclaimer
- ✅ e2e 文件按切片拆分(realtime.spec.ts 824 行 → 10 个 per-slice 文件)
- ✅ 补全 1c-1j daily 笔记 retroactive 总结(2026-06-18.md)
- ✅ 更新 MEMORY.md(当前阶段从 Pre-code 改为 v1 delivered)

### A 浅测补深 + 实施补全(✅ 完成,2026-06-18)

2026-06-18 spec vs 实施对照发现 1h/1i/1j 都有重大 gap(不止测试浅)。三处分别处理:

- **1h U2(文档拆分,✅ 完成)**:
  - 1h 拆为 1h-backend(🔴 spec partial) + 新切片 1h-ui(⏳ 未启动)
  - 1h-ui 已加入 v1 收尾路线(见 §8)
- **1i 🟡→🟢(✅ 完成)**:
  - 接线 `BehaviorTracker` 到 ws.go visitor read loop(每事件 Observe,每 100 事件 CheckAndFlag)
  - 加 Go 单测覆盖 rate limit 429 真触发 + 3 个启发式 + UA 黑名单 7 case
  - e2e 场景3 真查 PG visitors.fingerprint 持久化
  - 扩展 builtinBannedUAs(加 8 个现代 bot UA)
- **1j 🔴→🟢(✅ 完成)**:
  - 抽出 VisitorPanel/ChatPanel/ReplayPlayer/FloatingInput/CoBrowseOverlay/ReplayViewer/ReplayList/Dashboard 硬编码中文 20+ 处为 i18n key
  - 加 `app.switch_lang` 语言切换按钮到 Dashboard.vue(原 i18n key 存在但无 UI)
  - 加 i18n 切换/docker-prod 启动/CI workflow lint/README 命令 4 个真 e2e
  - 修产品代码 bug:Dockerfile 用 golang:1.22-alpine 但 go.mod 1.25.0 → 升级到 1.25-alpine;
    CI workflow setup-go 同步升级

### C 设计漏洞(A 完成后启动)

- migrations 不可重放:验证 `make migrate-down && make migrate-up` 干净循环,补 schema_migrations 表
- 5 个 Go 包零单测(api/storage/antiscrape/config/logging):补契约级单测,尤其 storage/queries.go(569 LOC)
- TS 测试仅 2 个 trivial smoke:补 SDK transport + admin stores 集成测试

## 8. LLM 协作提示

**进入新会话时**：

1. 先读本文件（项目状态快照）
2. 再按需读 [`CLAUDE.md`](../CLAUDE.md)（工作指南）、[`PLAN.md`](../PLAN.md)（架构详情）
3. 读当日的 `memory/daily/{date}.md`（如有）
4. 读相关的 `docs/reports/completed/`（注意每份报告顶部有**深度 badge + 叙述免责 disclaimer**）

**遇到冲突时**：

- 范围扩张请求（"加 X 功能"、"也支持 Y"）→ 检查是否在 v1 切片或后续切片路线（PLAN.md §8）。不在则停下来与用户确认是否调整 PLAN.md
- 架构决策冲突 → §4 列出的决策不能擅自动，先停下来
- "用户说 X 是 SaaS 多租户" → 不对，本项目明确不做多租户（START.md 描述的是竞品能力，不是本项目决策）
- 提到不存在的命令/服务/脚本 → 检查是否真的存在

**写代码时**：

- 遵循 [`CLAUDE.md`](../CLAUDE.md) "LLM Friendly" 与 "可观测性开发" 章节
- 所有用户可见文案走 Vue I18n key（中英双语 day 1）
- 所有日志结构化 JSON（含 timestamp / trace_id / span_id / event_type / payload）
- 函数命名遵循 [`docs/standards/naming-conventions.md`](./standards/naming-conventions.md) §4
- 测试深度判定遵循 [`docs/standards/verification-depth.md`](./standards/verification-depth.md)

**写文档时**：

- 模板见 [`docs/templates/`](./templates/)
- 结构规范见 [`docs/standards/doc-structure.md`](./standards/doc-structure.md)
- 修改即记录到 `docs/progress/`，完成**spec + impl 一起**移到 `docs/reports/completed/`
- 完成报告必须带深度 badge + 叙述免责 disclaimer
