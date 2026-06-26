# 项目状态快照

> **此文件是 LLM 进入项目的第一份必读文档**。读完此文件，LLM 应能：
> 1. 在 60 秒内说出项目做什么
> 2. 知道当前进展
> 3. 知道下一步该做什么
> 4. 知道哪些决策不能动、哪些范围不能扩
>
> 状态变化时直接编辑本文件（rolling），不保留历史快照（用 git 历史追溯）。

**最后更新**:2026-06-26(page-editor 全部 3 切片完成 + fork-3b 测试债补全 + Go 后端三包 90%+)

---

## 1. 一句话定位

构建一款**实时访客监控 + 运营互动 + 录像回放**的 ToB 工具的**开源替代品**——目标对标某商业竞品,但不考虑客户获取与销售(不做计费、注册、营销页),专注技术核心。License:AGPL-3.0。

## 2. 当前阶段

**v1 主干完全收口 + rrweb 硬分叉自维护 + 测试债补全 + ReplayPlayer 实时修复 + page-editor 全 3 切片 + Go 覆盖 90%+**。

最新进展(2026-06-26):

- ✅ **page-editor 全 3 切片**（3 commits: `19eca36` `42a5625` `1dcda74`）: post-v1 页面编辑器。pe-1 proto 类型 + PG 迁移 + Go CRUD + API handler；pe-2 admin WidgetsView.vue + 路由 + i18n；pe-3 SDK widget-config 获取器 + 配置驱动渲染。详见 [`progress/2026-06-26-page-editor-spec.md`](./progress/2026-06-26-page-editor-spec.md)
- ✅ **fork-3b 上游测试转译 + record 模块测试深化**（+88+29=+117 测试）: replay-core 从 57→146 测试，0 failed。Timer/machine/Replayer/record-helpers/mutation-buffer 全覆盖。
- ✅ **Go 后端覆盖三包 90%+**: recording 90.7% / api 90.0% / storage 91.5%。修复 snapshot.go Meta 缓存 0%→100%、authenticateOperatorWS 81%→90.5%、truncate 75%→100%。
- ✅ **开发端口迁移 7000-7100**: 30 文件变更。7080(server)/7073(admin)/7074(SDK)/7032(PG)/7079(Redis)/7020(MinIO)/7021(Console)。修复 macOS AirPlay 与 7000 端口冲突。
- ✅ **vendor-rrweb 硬分叉**（21 commits, feat 分支 → merged to master）
- ✅ **ReplayPlayer 实时事件修复 + cover sizing**
- ✅ **docs 全量整理归档**

最新进展(2026-06-25~26):

- ✅ **vendor-rrweb 硬分叉**（21 commits, `feat/vendor-rrweb` 分支 → merged to master `82a7a355`）: 将 rrweb alpha.20 TS 源码拷贝至 `packages/replay-core`，经 fork-0~4 逐步替换 SDK record / admin replay / 简化文件 / 实现 nodeID 跨端寻址。删除 Svelte rrweb-player 依赖（-600 行 hack）。详见 [`reports/completed/2026-06-25-vendor-rrweb-implementation.md`](./reports/completed/2026-06-25-vendor-rrweb-implementation.md)
- ✅ **fork-3b 上游测试转译 + record 模块测试深化**（2026-06-26, `e40d8f7`）: 新增 5 测试文件 / +88 测试，覆盖 Timer 类、状态机、Replayer 核心 API、error-handler、ProcessedNodeManager、StylesheetManager、MutationBuffer。replay-core 从 57→145 测试，**0 failed**。详见 [`reports/2026-06-26-fork-3b-record-test-verification.md`](./reports/2026-06-26-fork-3b-record-test-verification.md)
- ✅ **ReplayPlayer 实时事件修复 + cover sizing**（2026-06-26, 11 文件未提交改动）: 创建 Replayer 后立即 `startLive(farFuture)` 而非等待 `finish` 事件；sizing 从 contain 改为 cover 模式；store cap 500→5000；iframe sandbox `allow-scripts` 全开。详见 [`progress/2026-06-26-replay-live-mode-and-sizing-fix.md`](./progress/2026-06-26-replay-live-mode-and-sizing-fix.md)
- ✅ **全量 T2/T3 组件测试补全**（2026-06-26, `{daily}/2026-06-26.md`）: 新增 11 个测试文件 / +70 测试用例。admin 24 files / 203 passed（原 14 files / 146）；visitor-sdk 14 files / 219 passed（原 13 files / 206）
- ✅ **docs 全量整理归档**（2026-06-26）: vendor-rrweb spec+impl 归档、live-input-render 归档、replay-sizing 标记 superseded、project-status/daily/MEMORY 全量更新

前序进展(2026-06-18~19):

- ✅ 全栈深度审计 → [`docs/audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)(80 条发现,P0:13 / P1:27 / P2:26 / P3:14)
- ✅ 全部 13 个 P0 闭环(11 真修 + 2 已文档化 workaround,详见 1k/1l/1v 报告)
- ✅ 1k-1u 回归审计的 8 个新发现全部闭环(1v)
- ✅ 1a-1z 全切片 e2e acceptance 65 测试全绿([`reports/completed/2026-06-18-v1-e2e-acceptance.md`](./reports/completed/2026-06-18-v1-e2e-acceptance.md))
- ✅ e2e 后真实使用发现的 5 个生产 bug 全部修复([`reports/completed/2026-06-18-v1-followups.md`](./reports/completed/2026-06-18-v1-followups.md))
- ✅ admin SPA 消费 flagged 字段 + prod-mode/docker-prod e2e CI(`a660622`)

切片深度分布(v1 主干 + vendor-rrweb,2026-06-26):

- 🟢 verified-deep ×23(4 strict + 1 aligned + 18 touched)
- 🟡 verified-shallow ×9
- 🔴 implemented-unverified ×0
- ✅ vendor-rrweb: 🟢 touched（全部 5 切片完成并合并）

> **2026-06-19 测试信心审计结果**:`project-status.md` §5 此前自报 🟢 ×31,经 spec→test 对照实测,20 个切片应降级。审计方法详见 [`audits/2026-06-19-test-confidence-audit.md`](./audits/2026-06-19-test-confidence-audit.md)。降级原因:T0/T1 测试 gap 集中在认证/授权/GDPR/限流/可观测路径。修复 plan 见审计 §5。
>
> **前序**:1l/1o 曾因 1v 修复升回 🟢,本次审计根据 spec→test 实测再降级。1v 修了代码 bug 但**没补对应的回归测试**,所以审计后仍降级。这不是矛盾 — 代码正确不等于回归保护到位。

| 项 | 状态 |
|---|---|
| 架构 + 产品定位事实来源 | ✅ [`PLAN.md`](../PLAN.md) |
| Claude 工作指南 | ✅ [`CLAUDE.md`](../CLAUDE.md) |
| 文档规范 | ✅ [`docs/standards/`](./standards/) |
| LICENSE | ✅ AGPL-3.0 |
| README | ✅ |
| 切片 1a(仓库骨架) | 🟢 touched |
| 切片 1b(单向最小) | 🟡 |
| 切片 1c(rrweb 接入) | 🟡 |
| 切片 1d(录像归档) | 🔴 |
| 切片 1e(双向通道) | 🟡 |
| 切片 1f(表单 + 跳转) | 🟡 |
| 切片 1g(弹窗 + 聊天) | 🔴 |
| 切片 1h(认证 + 多运营 后端) | 🔴 |
| 切片 1h-ui(LoginView + 守卫) | 🟡 |
| 切片 1i(反爬虫) | 🟡 |
| 切片 1j(i18n + 部署 + CI) | 🟢 aligned |
| 切片 1k(安全阻断栈) | 🔴 |
| 切片 1l(GDPR 合规) | 🔴 |
| 切片 1m(可观测性) | 🟡 |
| 切片 1n(测试深度 + 文档) | 🟢 touched |
| 切片 1o(生产硬化) | 🟡 |
| 切片 1p(LLM friendly) | 🟢 touched |
| 切片 1q(死代码清理) | 🟢 touched |
| 切片 1r(i18n + logger 迁移) | 🟡 |
| 切片 1s(可观测性深化) | 🔴 |
| 切片 1t(测试覆盖补全) | 🟢 strict |
| 切片 1u(god files 拆分) | 🟢 touched |
| 切片 1v(审计后续修复) | 🟢 touched |
| 切片 1w(flagged session) | 🟡 |
| 切片 1x(登录暴力破解) | 🟡 |
| 切片 1y(visitor WS rate limit) | 🔴 |
| 切片 1z(生产就绪度补全) | 🟢 strict |
| 切片 1aa(TS 测试深化) | 🟢 strict |
| 切片 1ab(TrustedProxies 加固) | 🟢 strict |
| v1-e2e(全量 e2e acceptance) | 🟡 |
| v1-followups(5 个生产 bug fix) | 🟡 |

### 2.1 代码体量(2026-06-26 实测)

| 维度 | 数值 |
|---|---|
| Go 后端代码(server/,不含测试) | ~6700 LOC |
| Go 测试文件 | 59 个(`*_test.go`) |
| TypeScript 单测(admin + visitor-sdk) | 38 个(`*.test.ts`, +500 用例) |
| E2E 测试场景 | 22 个 spec(91+ test cases) |
| packages/replay-core | 37 源文件,~268KB; 8 测试文件, **146 测试** |
| Go 加权整体覆盖率(docker 环境) | recording **90.7%** / api **90.0%** / storage **91.5%** |
| config 包覆盖 | 98.0% 🟢 |
| privacy 包覆盖 | 95.0% 🟢 |
| proto 包覆盖 | **100.0%** 🟢(Go-1) |
| antiscrape 包覆盖 | **95.9%** 🟢(Go-1,86.7%→95.9%) |
| observability 包覆盖 | **91.7%** 🟢(Go-1,83.3%→91.7%) |
| logging 包覆盖 | **98.0%** 🟢(Go-2,79.6%→98.0%) |
| hub 包覆盖 | **94.1%** 🟢(Go-3,72.4%→94.1%,race -count=3 通过) |
| storage 包覆盖 | **86.5%** 🟡(Go-4,57.6%→86.5%;未达 90% 目标 3.5pp,剩余 scan 边缘分支 ROI 低) |
| recording 包覆盖 | **77.7%** 🟡(Go-5,48.0%→77.7%;未达 90% 目标 12.3pp,5 表 cascade + flushSession 错误路径 ROI 低) |
| api 包覆盖 | **47.9%** 🟡(Go-6 Commit 1,38.2%→47.9%;未达 90% 目标 42.1pp,WS handlers + HTTP 业务路径留 backlog) |
| cmd/server 包覆盖 | 4.9% 🔴(main 入口,e2e 兜底) |
| vitest coverage 配置 | ✅ 已配 v8 provider(TS-1);admin **85.67%**(TS-3) / visitor-sdk 36.05%(TS-2 partial,src/index.ts 留 backlog) |

> **覆盖口径说明**(2026-06-20 修正):Go `-cover` 必须在 docker-compose 启动后跑,否则 storage/api/recording/antiscrape 等依赖 PG/Redis/MinIO 的包会被 skip,显示虚假低覆盖率(如 storage 1.5% 是本地无 docker 的口径,实测 57.6%)。完整逐包覆盖率 + 关键未覆盖函数清单 + 历史虚标对账见 [`audits/2026-06-20-coverage-assessment.md`](./audits/2026-06-20-coverage-assessment.md)。

### 2.2 重命名重构记录(2026-06-20)

`marketing-monitor` → `pinconsole` 全量重命名 5 步,共 5 commit:

| 步 | commit | 内容 |
|---|---|---|
| 1 | `f461b59` | Go module `github.com/iannil/marketing-monitor` → `github.com/iannil/pinconsole` |
| 2 | `dbf631b` | pnpm scope `@marketing-monitor/*` → `@pinconsole/*`(admin / visitor-sdk / e2e) |
| 3 | `d3d5f03` | docker-compose + DB schema + Go embed 路径重建 |
| 4 | `ea271d7` | 当前活跃文档(project-status / MEMORY / daily / standards)+ CI / ops 全量改 pinconsole |
| 5 | `234fa06` | e2e DB 连接 + visitor-sdk package.json description 清理 + verify |

**历史快照保留**:`docs/reports/completed/` 22 份报告快照中的 `@marketing-monitor/*` 命令字面量已批量更新为 `@pinconsole/*`;e2e 测试中的"1r 切片后 SDK 不再输出 marketing-monitor 字面量"类历史断言**保持原样**(那是当时被移除的真实字符串,改了会让断言失效)。

## 3. 事实来源优先级（冲突时按此解析）

```
1. PLAN.md     — 架构、产品定位、技术栈、切片拆分、决策理由
2. CLAUDE.md   — Claude 工作指南（含文档/记忆/可观测性约定）
3. 本文件       — 当前状态与下一步
```

冲突场景示例:
- PLAN.md §3 锁定 Gin + coder/websocket,与历史草稿提到的 Go-Zero 冲突 → 用 Gin(PLAN.md 优先)
- 外部反馈"应支持 SaaS 多租户",CLAUDE.md/PLAN.md §3 明确不做 → 不做(本项目硬约束)

## 4. 架构决策清单（不可重新讨论）

详见 [`PLAN.md`](../PLAN.md) §3-§5。简表:

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

如需变更任一决策,先与用户确认 → 更新 PLAN.md → 更新此文件。

## 5. v1 切片状态（最终）

详见 [`PLAN.md`](../PLAN.md) §7。**深度判定标准**: [`standards/verification-depth.md`](./standards/verification-depth.md)。

图例:🟢 verified-deep(strict/aligned/touched) / 🟡 verified-shallow / 🔴 implemented-unverified

> **2026-06-19 测试信心审计**已对每个切片做 spec→test 对照,详见 [`audits/2026-06-19-test-confidence-audit.md`](./audits/2026-06-19-test-confidence-audit.md)。下表 badge 为审计后实测值。

| 子切片 | 内容 | 深度 | 报告 |
|---|---|---|---|
| 1a | 仓库骨架 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1a-implementation.md) |
| 1b | 单向最小 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1b-implementation.md) + [1ai](./reports/completed/2026-06-19-slice-1ai-storage-repo-tests-implementation.md) — session_repo PG 集成(CreateSession/Get/Touch/End/List 全覆盖) |
| 1c | rrweb 接入 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1c-implementation.md) |
| 1d | 录像归档 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1d-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) + [1ag](./reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md) — 4 T0 + 5 T1 全部关闭 + replay HTTP 行为级(parseSince/UUID/since 拒绝路径) |
| 1e | 双向通道 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1e-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) + [1ai-b](./reports/completed/2026-06-20-slice-1aib-storage-repos-b-implementation.md) — command_repo 全覆盖(Create/List/Delete + JSON round-trip) |
| 1f | 表单 + 跳转 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1f-implementation.md) |
| 1g | 弹窗 + 聊天 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1g-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) + [1ah](./reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-implementation.md) — 5/5 T1 关闭 + chat repo PG + 1ah chat_http listMessages UUID 路径 |
| 1h | 认证 + 多运营(后端) | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1h-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) + [1ag](./reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md) + [1ah](./reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-implementation.md) — **1ac-final 关闭全部 6/6 T0** + 1ag auth HTTP + 1ah claim HTTP(getClaim/release 行为级) |
| 1h-ui | admin LoginView + 守卫 | 🟢 touched | [spec](./reports/completed/2026-06-18-slice-1h-ui-spec.md) + [impl](./reports/completed/2026-06-18-slice-1h-ui-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) |
| 1i | 反爬虫 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1i-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) — 1ac 关闭 fail-open |
| 1j | i18n + 部署 + CI | 🟢 aligned | [impl](./reports/completed/2026-06-17-slice-1j-implementation.md) |
| 1k | 安全阻断栈 | 🟡 | [spec](./reports/completed/2026-06-18-slice-1k-spec.md) + [impl](./reports/completed/2026-06-18-slice-1k-implementation.md) — 1ac 关闭 8/9 T0(剩 1k-9 e2e 范围) |
| 1l | GDPR 合规 | 🟡 | [spec](./reports/completed/2026-06-18-slice-1l-spec.md) + [impl](./reports/completed/2026-06-18-slice-1l-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) — 1ac 关闭 6/6 T0 + admin role bug 修复 |
| 1m | 可观测性 | 🟢 touched | [spec](./reports/completed/2026-06-18-slice-1m-spec.md) + [impl](./reports/completed/2026-06-18-slice-1m-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) |
| 1n | 测试深度 + 文档虚标修复 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1n-implementation.md) |
| 1o | 生产硬化 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1o-implementation.md) — 1v R2 rubric 真实集成(代码层)+ [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) per-sub cancel 接线覆盖 |
| 1p | LLM friendly | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1p-implementation.md) |
| 1q | 死代码 + 重复清理 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1q-implementation.md) |
| 1r | i18n + logger 迁移 | 🟡 | [impl](./reports/completed/2026-06-18-slice-1r-implementation.md) — 1ac 未触及 |
| 1s | 可观测性深化 | 🟡 | [impl](./reports/completed/2026-06-18-slice-1s-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) — 1ad 关闭 13/13 T1 lifecycle 接线契约;T0 deep integration 仍开 |
| 1t | 测试覆盖补全 | 🟢 strict | [impl](./reports/completed/2026-06-18-slice-1t-implementation.md) |
| 1u | god files 拆分 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1u-implementation.md) |
| 1v | 审计后续修复 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1v-implementation.md) |
| 1w | flagged session 接入 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1w-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) — 1ad 关闭 4/4 T1 warn + is_flagged 接线 |
| 1x | 登录暴力破解防护 | 🟢 touched | [impl](./reports/completed/2026-06-18-slice-1x-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) — 1ac 关闭 Lua 原子 |
| 1y | visitor WS rate limit | 🟡 | [impl](./reports/completed/2026-06-18-slice-1y-visitor-ws-rate-limit.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) — 1ac 关闭 close+flag 接线契约 |
| 1z | 生产就绪度补全 | 🟢 strict | [impl](./reports/completed/2026-06-18-slice-1z-prod-readiness-gaps.md) |
| v1-e2e | 全量 e2e acceptance | 🟡 | [impl](./reports/completed/2026-06-18-v1-e2e-acceptance.md) |
| v1-followups | e2e 后 5 个生产 bug fix | 🟡 | [impl](./reports/completed/2026-06-18-v1-followups.md) |
| 1aa | TS 测试深化(admin 64 + SDK 48) | 🟢 strict | [impl](./reports/completed/2026-06-19-slice-1aa-ts-test-deepening.md) |
| 1ab | TrustedProxies 加固(P1-5) | 🟢 strict | [impl](./reports/completed/2026-06-19-slice-1ab-trusted-proxies.md) |
| 1ae | 测试健康度加固(9 项 P0+P1) | 🟢 touched | [audit](./audits/2026-06-19-test-health-audit.md) + [spec](./reports/completed/2026-06-19-slice-1ae-spec.md) + [impl](./reports/completed/2026-06-19-slice-1ae-implementation.md) — mutation score 71.4%→100%, e2e flaky 20%→0%, 整体 verdict 🔴→🟡 |
| 1af | 测试健康度深化(R3 续做 6 group) | 🟢 touched | [spec](./reports/completed/2026-06-19-slice-1af-spec.md) + [impl](./reports/completed/2026-06-19-slice-1af-implementation.md) — 23 新行为级测试,D1 PASS 率 ~55%→~75%, 整体 verdict 🟡→🟢 |
| 1ag | api handler 行为级测试(auth+replay) | 🟢 touched | [spec](./reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-spec.md) + [impl](./reports/completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md) — 12 新测试,api 覆盖 20%→25.5%,1d/1h 升 🟢 touched |
| 1ah | claim/chat handler 行为级测试 | 🟢 touched | [spec](./reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-spec.md) + [impl](./reports/completed/2026-06-19-slice-1ah-claim-chat-handler-tests-implementation.md) — 10 新测试,api 覆盖 25.5%→29.1%,getClaim 0%→83%、release 38%→76%、listMessages 0%→57% |
| 1aj | 修 parseSince 负数 + ws_ratelimit flaky | 🟢 touched | [spec](./reports/completed/2026-06-19-slice-1aj-followup-bugs-spec.md) + [impl](./reports/completed/2026-06-19-slice-1aj-followup-bugs-implementation.md) — 1 新测试 + 4 测试 err-skip 改造,api 覆盖 29.1%→29.3%,1ag follow-up 关闭 |
| 1ai | storage user+session repo PG 集成测试 | 🟢 touched | [spec](./reports/completed/2026-06-19-slice-1ai-storage-repo-tests-spec.md) + [impl](./reports/completed/2026-06-19-slice-1ai-storage-repo-tests-implementation.md) — 11 新测试,storage 覆盖 20.1%→39.2%,user_repo 100% + session_repo 核心 4 函数 100% |
| 1ai-b | storage visitor/command/event_blob repo 测试 | 🟢 touched | [spec](./reports/completed/2026-06-20-slice-1aib-storage-repos-b-spec.md) + [impl](./reports/completed/2026-06-20-slice-1aib-storage-repos-b-implementation.md) — 11 新测试,storage 覆盖 39.2%→57.8%,visitor 100%、command 84-100%、event_blob 81-100% |
| 1ai-c | AuthHandler 接口化 Phase 1 + login happy path | 🟢 touched | [spec](./reports/completed/2026-06-20-slice-1aic-auth-handler-interface-spec.md) + [impl](./reports/completed/2026-06-20-slice-1aic-auth-handler-interface-implementation.md) — 接口重构 + 3 happy path 测试 + 2 mock,auth.go login 37.8%→86.5%,api 覆盖 29.3%→31.2% |
| 1ai-d | me+logout happy path | 🟢 touched | [spec](./reports/completed/2026-06-20-slice-1aid-me-logout-happy-path-spec.md) + [impl](./reports/completed/2026-06-20-slice-1aid-me-logout-happy-path-implementation.md) — 4 新测试,auth.go me 40%→100%、logout 85.7%→100%,AuthHandler 全 4 handler happy+拒绝路径全覆盖 |
| 1ai-e | claim+chat listMessages 接口化 + happy path | 🟢 touched | [spec](./reports/completed/2026-06-20-slice-1aie-claim-chat-interface-spec.md) + [impl](./reports/completed/2026-06-20-slice-1aie-claim-chat-interface-implementation.md) — 3 新接口 + 4 happy path 测试,claim 25%→78.1%、listMessages 57%→76.2%,api 覆盖 31.8%→33.7% |
| 1ai-f | CommandHandler 接口化(精简)+ postCommand 拒绝路径 | 🟢 touched | [spec](./reports/completed/2026-06-20-slice-1aif-command-handler-interface-spec.md) + [impl](./reports/completed/2026-06-20-slice-1aif-command-handler-interface-implementation.md) — commandRepo 接口 + 3 测试,postCommand 0%→16%(拒绝路径) |
| 1ai-g | requireClaimOwnership 接口化 + ChatHandler 字段重构 + postMessage happy path | 🟢 touched | commit `2c186d0`/`0f5f347`/`a75c2f6` — 4 测试(Success/NotOwner/InvalidJSON/EmptyContent) + 2 mock,postMessage 35%→85%;实施记录见 [`daily/2026-06-19.md`](../memory/daily/2026-06-19.md) §"1ai-g requireClaimOwnership 接口化"(无独立 impl 报告) |
| 1ai-h | SessionHandler 接口化 + initSession happy path + replay 纯函数测试 | 🟢 touched | commit `7af0807`/`b748e43`/`08377d8` — 4 initSession + 6 replay 纯函数测试,initSession 0%→87%、decodePayloadAsEvent 0%→100%、eventPayloadToMap 0%→100%;实施记录见 [`daily/2026-06-19.md`](../memory/daily/2026-06-19.md) §"1ai-h SessionHandler 接口化"(无独立 impl 报告) |

**累计**:🟢 ×23(4 strict + 1 aligned + 18 touched) / 🟡 ×9 / 🔴 ×0

**1ac + 1ac-final + 1ad 完成统计**(2026-06-19):
- 关闭 28/28 T0 + 40/40 T1(68/68 critical + important 路径覆盖)
- 修复 2 个代码 bug:`deleteVisitor` 缺 admin role(1ac)+ `operatorWS` 完全无认证(1ac-final)
- 7 个原 🔴 切片全部升 🟡/🟢(1d→🟡, 1g/1h-backend/1k→🟢/🟡, 1l→🟡, 1s→🟡, 1y→🟢)
- 16 个切片 🟢 touched(从审计后 6 个升至)
- 剩余 T2/T3(40 项,~15 小时)留 backlog,不阻塞 v1 release

**1ag~1ai-h 完成统计**(2026-06-19~20):
- 累计 +75 测试 + 5 接口化重构(AuthHandler / ClaimHandler+ChatHandler.listMessages / CommandHandler / requireClaimOwnership+ChatHandler.postMessage / SessionHandler)
- api 包覆盖 20.0% → 实测 38.2%(docker 环境,详见 [`audits/2026-06-20-coverage-assessment.md`](./audits/2026-06-20-coverage-assessment.md))
- storage 包覆盖 20.1% → 实测 57.6%(docker 环境;repo 函数 70-100%,适配器 0%)
- 接口化模式 PoC 在 5 个 handler 验证

**vendor-rrweb 统计**(2026-06-25~26, feat 分支 21 commits):
- fork-0~4 全部完成并合并至 master（`82a7a355`）
- 删除 Svelte rrweb-player 依赖,净减 ~600 行 hack 代码
- ESM bundle: 423KB→396KB(-27KB) via fork-3a 精简
- 新增 2 parity e2e,360 单元测试全绿
- nodeID 跨端寻址全链路打通（snapshot 写 data-rr-node-id → elementFromPoint → Server → NodeMap.get → element.click()）

**累计估时**:solo 全职约 15-18 周(3.5-4.5 个月);业余约 10-13 个月。实际本次集中冲刺:2026-06-17~20 共 4 天(90+ commits) + 2026-06-25~26 共 2 天(21 commits: vendor-rrweb + 测试债 + ReplayPlayer 修复)。

## 6. 已识别风险

详见 [`PLAN.md`](../PLAN.md) §10 + [`audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)。

**已闭环**（含 vendor-rrweb 更新）:
- ✅ 全部 13 个 P0 安全/合规/部署阻断项(1k/1l 修 11 个真修,1v 补 2 个 workaround 已文档化)
- ✅ 1k-1u regression 8 个新发现(1v)
- ✅ 文档虚标(P0-10/11/12) → 1n 修复
- ✅ e2e 静默跳过 → 1n 改 strict assertion
- ✅ 浅测补深 → 1n + e2e acceptance
- ✅ 5 个 e2e 后生产 bug(v1-followups)
- ✅ **rrweb 在动态 SPA 下节点 ID 不稳定** → fork-4 nodeID 跨端寻址已打通（snapshot 写 `data-rr-node-id`,CoBrowseOverlay 真实 elementFromPoint 查询,end-to-end 验证通过）

**未修(不阻断 v1 release)**:
- 🟡 **P2/P3**:~20 条非阻断项(代码质量、文档完善、测试深化),详见审计文档,留作 post-v1 backlog
- **AGPL 可能劝退部分企业采用**:双 license 路径预留（已加 "commercial license available" 措辞）
- **fork-3b 上游测试转 Playwright**:5 组(snapshot/replayer/observer/shadow DOM/iframe/mask)暂留 backlog

## 7. 下一步动作

### v1 release 判定（§7.5）

**GO 判据**(全部满足):

- ✅ e2e 65 测试全绿
- ✅ 审计 13 个 P0 全闭环
- ✅ 文档对齐(本批次清理)
- ✅ docker-compose prod 凭证 fail-secure
- ✅ CI workflow 跑通(ci.yml + release.yml)
- ✅ release 二进制单 binary 部署(嵌入式 admin/sdk/landing)

**NO-GO 触发**(任一即阻断):
- 任何 P0 重现
- 任何深度 badge 虚标
- 文档与代码状态分歧

**结论**:✅ **v1 release ready**。

### post-v1 候选（按优先级）

1. **自定义域名**(PLAN.md §8 #3)— DNS 验证 + Let's Encrypt ACME + Host-header 路由
2. **页面编辑器**(PLAN.md §8 #2)— 拖拽 / 低代码 / JSON schema → Go 模板渲染
3. **Tauri 桌面端**(PLAN.md §8 #4)— Win + Mac,复用 admin SPA
4. **fork-3b 上游测试转 Playwright** — 5 组 snapshot/replayer/observer/shadow DOM/iframe/mask（~2-3d）
5. **反爬加固**(PLAN.md §8 #5)— CAPTCHA + honeypot + 动态类名/ID

## 8. LLM 协作提示

**进入新会话时**:

1. 先读本文件(项目状态快照)
2. 再按需读 [`CLAUDE.md`](../CLAUDE.md)(工作指南)、[`PLAN.md`](../PLAN.md)(架构详情)
3. 读当日的 `memory/daily/{date}.md`(如有)
4. 读相关的 `docs/reports/completed/`(注意每份报告顶部有**深度 badge + 叙述免责 disclaimer**)
5. 如需理解回放核心,读 `packages/replay-core/`（rrweb alpha.20 TS 源码 fork，非 rrweb-player Svelte）
6. 如需理解 e2e 后的真实使用反馈,读 [`reports/completed/2026-06-18-v1-followups.md`](./reports/completed/2026-06-18-v1-followups.md)

**遇到冲突时**:

- 范围扩张请求("加 X 功能"、"也支持 Y") → 检查是否在 v1 切片或后续切片路线(PLAN.md §8)。不在则停下来与用户确认是否调整 PLAN.md
- 架构决策冲突 → §4 列出的决策不能擅自动,先停下来
- "用户说 X 是 SaaS 多租户" → 不对,本项目明确不做多租户(PLAN.md §3 锁定单租户,"不考虑客户和销售"是用户硬约束)
- 提到不存在的命令/服务/脚本 → 检查是否真的存在

**写代码时**:

- 遵循 [`CLAUDE.md`](../CLAUDE.md) "LLM Friendly" 与 "可观测性开发" 章节
- 所有用户可见文案走 Vue I18n key(中英双语 day 1)
- 所有日志结构化 JSON(含 timestamp / trace_id / span_id / event_type / payload)
- 函数命名遵循 [`docs/standards/naming-conventions.md`](./standards/naming-conventions.md) §4
- 测试深度判定遵循 [`docs/standards/verification-depth.md`](./standards/verification-depth.md)
- `Partial<T>` 合并必先 `dropUndefined`(详见 [`memory/MEMORY.md`](../memory/MEMORY.md) 经验教训)
- claim 锁只用于写/control 端点,只读端点(list/get history)不要求 claim

**写文档时**:

- 模板见 [`docs/templates/`](./templates/)
- 结构规范见 [`docs/standards/doc-structure.md`](./standards/doc-structure.md)
- 修改即记录到 `docs/progress/`,完成 **spec + impl 一起**移到 `docs/reports/completed/`
- 完成报告必须带深度 badge + 叙述免责 disclaimer
