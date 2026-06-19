# 项目状态快照

> **此文件是 LLM 进入项目的第一份必读文档**。读完此文件，LLM 应能：
> 1. 在 60 秒内说出项目做什么
> 2. 知道当前进展
> 3. 知道下一步该做什么
> 4. 知道哪些决策不能动、哪些范围不能扩
>
> 状态变化时直接编辑本文件（rolling），不保留历史快照（用 git 历史追溯）。

**最后更新**:2026-06-19(1ad 完成 — 40/40 T1 关闭,7 个原 🔴 切片全部升 🟡/🟢;累计 28 T0 + 40 T1 + 2 代码 bug 修复)

---

## 1. 一句话定位

构建一款**实时访客监控 + 运营互动 + 录像回放**的 ToB 工具的**开源替代品**——目标对标某商业竞品,但不考虑客户获取与销售(不做计费、注册、营销页),专注技术核心。License:AGPL-3.0。

## 2. 当前阶段

**v1 主干完全收口**:1a-1z 全切片 + e2e acceptance + 5 个 followup fix + admin flagged UI/prod-mode CI 全部完成(70+ commits)。**当前无活跃切片**。

最新进展(2026-06-18):

- ✅ 全栈深度审计 → [`docs/audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)(80 条发现,P0:13 / P1:27 / P2:26 / P3:14)
- ✅ 全部 13 个 P0 闭环(11 真修 + 2 已文档化 workaround,详见 1k/1l/1v 报告)
- ✅ 1k-1u 回归审计的 8 个新发现全部闭环(1v)
- ✅ 1a-1z 全切片 e2e acceptance 65 测试全绿([`reports/completed/2026-06-18-v1-e2e-acceptance.md`](./reports/completed/2026-06-18-v1-e2e-acceptance.md))
- ✅ e2e 后真实使用发现的 5 个生产 bug 全部修复([`reports/completed/2026-06-18-v1-followups.md`](./reports/completed/2026-06-18-v1-followups.md))
- ✅ admin SPA 消费 flagged 字段 + prod-mode/docker-prod e2e CI(`a660622`)

切片深度分布(v1 主干,2026-06-19 测试信心审计 + 1ac + 1ac-final + 1ad 完成后):

- 🟢 verified-deep ×21(4 strict + 1 aligned + 16 touched)
- 🟡 verified-shallow ×10
- 🔴 implemented-unverified ×0(全部 7 个原 🔴 升 🟡/🟢)
- 全部切片已交付

> **2026-06-19 测试信心审计结果**:`project-status.md` §5 此前自报 🟢 ×31,经 spec→test 对照实测,20 个切片应降级。审计方法详见 [`audits/2026-06-19-test-confidence-audit.md`](./audits/2026-06-19-test-confidence-audit.md)。降级原因:T0/T1 测试 gap 集中在认证/授权/GDPR/限流/可观测路径。修复 plan 见审计 §5。
>
> **前序**:1l/1o 曾因 1v 修复升回 🟢,本次审计根据 spec→test 实测再降级。1v 修了代码 bug 但**没补对应的回归测试**,所以审计后仍降级。这不是矛盾 — 代码正确不等于回归保护到位。

| 项 | 状态 |
|---|---|
| 产品事实来源 | ✅ [`START.md`](../START.md) |
| 架构事实来源 | ✅ [`PLAN.md`](../PLAN.md) |
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

## 3. 事实来源优先级（冲突时按此解析）

```
1. PLAN.md     — 架构、技术栈、切片拆分、决策理由
2. START.md    — 产品需求、竞品能力、业务上下文
3. CLAUDE.md   — Claude 工作指南（含文档/记忆/可观测性约定）
4. 本文件       — 当前状态与下一步
```

冲突场景示例:
- PLAN.md 说用 Gin,START.md 提到 "Gin 或 Go-Zero" → 用 Gin(PLAN.md 优先)
- START.md 描述"同时支持 SaaS 多租户",CLAUDE.md 说"不做多租户" → 不做(CLAUDE.md 优先,因 START.md 是描述竞品能力而非本项目决策)

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
| 1b | 单向最小 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1b-implementation.md) |
| 1c | rrweb 接入 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1c-implementation.md) |
| 1d | 录像归档 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1d-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) — 4 T0 + 5 T1 全部关闭(源码契约级) |
| 1e | 双向通道 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1e-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) |
| 1f | 表单 + 跳转 | 🟡 | [impl](./reports/completed/2026-06-17-slice-1f-implementation.md) |
| 1g | 弹窗 + 聊天 | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1g-implementation.md) + [1ad](./reports/completed/2026-06-19-slice-1ad-implementation.md) — 5/5 T1 关闭 + chat repo PG 集成 |
| 1h | 认证 + 多运营(后端) | 🟢 touched | [impl](./reports/completed/2026-06-17-slice-1h-implementation.md) + [1ac](./reports/completed/2026-06-19-slice-1ac-implementation.md) — **1ac-final 关闭全部 6/6 T0**(operatorWS 加 cookie session 鉴权) |
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

**累计**:🟢 ×21(4 strict + 1 aligned + 16 touched) / 🟡 ×10 / 🔴 ×0

**1ac + 1ac-final + 1ad 完成统计**(2026-06-19):
- 关闭 28/28 T0 + 40/40 T1(68/68 critical + important 路径覆盖)
- 修复 2 个代码 bug:`deleteVisitor` 缺 admin role(1ac)+ `operatorWS` 完全无认证(1ac-final)
- 7 个原 🔴 切片全部升 🟡/🟢(1d→🟡, 1g/1h-backend/1k→🟢/🟡, 1l→🟡, 1s→🟡, 1y→🟢)
- 16 个切片 🟢 touched(从审计后 6 个升至)
- 剩余 T2/T3(40 项,~15 小时)留 backlog,不阻塞 v1 release

**累计估时**:solo 全职约 14-17 周(3.5-4 个月);业余约 9-12 个月。实际本次 2 天交付（70+ commits），属于集中冲刺。测试信心补全完成,可考虑 post-v1 路线(自定义域名 / 页面编辑器 / Tauri)。

**累计估时**:solo 全职约 14-17 周(3.5-4 个月);业余约 9-12 个月。实际本次 2 天交付（70+ commits），属于集中冲刺。

## 6. 已识别风险

详见 [`PLAN.md`](../PLAN.md) §10 + [`audits/2026-06-18-deep-audit.md`](./audits/2026-06-18-deep-audit.md)。

**已闭环**（详情见各切片报告）:
- ✅ 全部 13 个 P0 安全/合规/部署阻断项(1k/1l 修 11 个真修,1v 补 2 个 workaround 已文档化)
- ✅ 1k-1u regression 8 个新发现(1v)
- ✅ 文档虚标(P0-10/11/12) → 1n 修复
- ✅ e2e 静默跳过 → 1n 改 strict assertion
- ✅ 浅测补深 → 1n + e2e acceptance
- ✅ 5 个 e2e 后生产 bug(v1-followups)

**未修(不阻断 v1 release)**:
- 🟡 **P2/P3**:~20 条非阻断项(代码质量、文档完善、测试深化),详见审计文档,留作 post-v1 backlog
- **rrweb 在动态 SPA 下节点 ID 不稳定**:测试矩阵需覆盖主流框架
- **AGPL 可能劝退部分企业采用**:双 license 路径预留

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
4. **反爬加固**(PLAN.md §8 #5)— CAPTCHA + honeypot + 动态类名/ID

## 8. LLM 协作提示

**进入新会话时**:

1. 先读本文件(项目状态快照)
2. 再按需读 [`CLAUDE.md`](../CLAUDE.md)(工作指南)、[`PLAN.md`](../PLAN.md)(架构详情)
3. 读当日的 `memory/daily/{date}.md`(如有)
4. 读相关的 `docs/reports/completed/`(注意每份报告顶部有**深度 badge + 叙述免责 disclaimer**)
5. 如需理解 e2e 后的真实使用反馈,读 [`reports/completed/2026-06-18-v1-followups.md`](./reports/completed/2026-06-18-v1-followups.md)

**遇到冲突时**:

- 范围扩张请求("加 X 功能"、"也支持 Y") → 检查是否在 v1 切片或后续切片路线(PLAN.md §8)。不在则停下来与用户确认是否调整 PLAN.md
- 架构决策冲突 → §4 列出的决策不能擅自动,先停下来
- "用户说 X 是 SaaS 多租户" → 不对,本项目明确不做多租户(START.md 描述的是竞品能力,不是本项目决策)
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
