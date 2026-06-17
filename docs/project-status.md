# 项目状态快照

> **此文件是 LLM 进入项目的第一份必读文档**。读完此文件，LLM 应能：
> 1. 在 60 秒内说出项目做什么
> 2. 知道当前进展
> 3. 知道下一步该做什么
> 4. 知道哪些决策不能动、哪些范围不能扩
>
> 状态变化时直接编辑本文件（rolling），不保留历史快照（用 git 历史追溯）。

**最后更新**：2026-06-18（A 阶段 1i + 1j 升级 🟢;1h 拆为 1h-ui 待启动）

---

## 1. 一句话定位

构建一款**实时访客监控 + 运营互动 + 录像回放**的 ToB 工具的**开源替代品**——目标对标某商业竞品，但不考虑客户获取与销售（不做计费、注册、营销页），专注技术核心。License：AGPL-3.0。

## 2. 当前阶段

**v1 切片 1a-1g + 1i + 1j 已深度验证(🟢);1h 拆为 1h-backend(🔴 spec partial) + 1h-ui(⏳ 未启动)**

A 阶段成果(2026-06-18):

- 🟢 verified-deep ×9(1a-1g + 1i + 1j)
- 🔴 implemented-unverified ×1(1h-backend,登录 UI 未实施)
- ⏳ 未启动 ×1(1h-ui,拆出新切片)

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
| [1a](./reports/completed/2026-06-17-slice-1a-implementation.md) | 仓库骨架 | 🟢 | 5 smoke + 端到端 release build |
| [1b](./reports/completed/2026-06-17-slice-1b-implementation.md) | 单向最小 | 🟢 | 4 e2e + 8 Go 单测 |
| [1c](./reports/completed/2026-06-17-slice-1c-implementation.md) | rrweb 接入 | 🟢 | 4 e2e(实时回放真验证) |
| [1d](./reports/completed/2026-06-17-slice-1d-implementation.md) | 录像归档 | 🟢 | 4 e2e(live→historical 真验证) |
| [1e](./reports/completed/2026-06-17-slice-1e-implementation.md) | 双向通道 | 🟢 | 5 e2e(含 PG 审计表) |
| [1f](./reports/completed/2026-06-17-slice-1f-implementation.md) | 表单 + 跳转 | 🟢 | 4 e2e(含 navigate 跨域拒绝) |
| [1g](./reports/completed/2026-06-17-slice-1g-implementation.md) | 弹窗 + 聊天 | 🟢 | 4 e2e(含离线消息持久化) |
| [1h](./reports/completed/2026-06-17-slice-1h-implementation.md) | 认证 + 多运营(后端) | 🔴 | spec 决策 #5 login UI 未实施;已拆为 1h-backend(本报告) + 1h-ui(待启动) |
| [1i](./reports/completed/2026-06-17-slice-1i-implementation.md) | 反爬虫 | 🟢 | A 阶段升级:BehaviorTracker 接线 + Go 单测覆盖深度逻辑 + e2e 真查 PG fingerprint |
| [1j](./reports/completed/2026-06-17-slice-1j-implementation.md) | i18n + 部署 + CI | 🟢 | A 阶段升级:硬编码中文抽 key + 语言切换按钮 + 4 个真 e2e + 修 Dockerfile go 版本 bug |

**累计估时**：solo 全职约 14-17 周（3.5-4 个月）；业余约 9-12 个月。

## 6. 已识别风险

详见 [`PLAN.md`](../PLAN.md) §10 + 2026-06-18 reality check 发现：

- **GDPR / CCPA 合规**：提交前按键监听属敏感处理，需明确访客同意流程
- **co-browsing 接管跳转可能被滥用**：需审计日志 + 访客"紧急退出"
- **rrweb 在动态 SPA 下节点 ID 不稳定**：测试矩阵需覆盖主流框架
- **AGPL 可能劝退部分企业采用**：双 license 路径预留
- **🔴 1h 登录 UI 完全未实施**:spec 决策 #5 "登录 UI + Vue Router 守卫"未做。admin SPA 无 LoginView、无路由守卫。已拆为新切片 1h-ui
- **🔴 1i `BehaviorTracker` 死代码**:`antiscrape/behavior.go` 完整实现 3 个启发式 + FlagSession 调用,但 server/ 中零调用方。1i "行为分析"功能从未运行
- **🔴 1j i18n 子组件未装**:主视图 29 个 `t()` 调用,但 VisitorPanel/Dashboard 仍有硬编码中文(`"运营"` / `"从左侧选择一个访客"` / `"累计事件："` 等)
- **🟡 1h 浅测**:登录/登出/Claim 测试只验证 API 端点存在,未验证 cookie 真设置/真清除/受保护端点真拒绝匿名
- **🟡 1i 浅测**:rate limit 在 prod 模式才生效,e2e 在 dev 模式跑只验证 middleware 注册,未验证 429 真触发
- **🟡 HeadlessChrome UA ban 是死代码**:新版 Playwright chromium UA 是 `Chrome/...` 不含 `HeadlessChrome`,实际绕过
- **migrations 未通过 `migrate` CLI 应用**:`schema_migrations` 表缺失,推测用 `psql -f` 直接应用。`make migrate-down && make migrate-up` 是否干净循环未验证

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
