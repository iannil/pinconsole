# MEMORY.md — 项目长期记忆（沉积层）

> 本文件是项目的长期记忆。记录"当前真实状态"——用户偏好、项目上下文、关键决策、经验教训。
> 触发更新：用户陈述偏好、发现错误修复模式、建立项目规则、关键决策变化。
> 与 [`memory/daily/`](./daily/) 的关系：daily 是不可变日志（流），MEMORY 是当前状态（沉积）。

**最后更新**：2026-06-26(page-editor 全 3 切片完成 + fork-3b 测试 + Go 90%+ + 端口迁移 7000-7100)

---

## 用户偏好

### 语言
- **交流与文档**：中文
- **生成代码**：英文
- 例外：纯中文业务术语保留中文（如"运营"、"访客"、"落地页"）

### 工作风格
- 推荐方案时直接给出首选 + 理由，不堆叠"on the other hand"
- 偏好"先架构后实施"——重大决策不绕过 grill-me 风格访谈
- 范围控制严格——产品本身仍 OSS / 不做计费 / 不做注册流 / 不做多租户 SaaS / 不做云托管；**但 maintainer 自有 `marketing/` 独立目录可做咨询转化**（2026-06-22 解禁，与 OSS `landing/` 严格区隔）

### 技术倾向
- **OSS 立场**：AGPL-3.0、防云厂商 SaaS 化
- **anti-SaaS 倾向**：本项目不做多租户、不做计费、不做注册流
- **架构偏好**：monolith 优先、纵向切片优先、不过度设计
- **可观测性**：愿意接受装饰器/切面模式、全链路 trace_id、结构化 JSON 日志（详见 [`CLAUDE.md`](../CLAUDE.md) "可观测性开发"）
- **LLM Friendly**：一致的分层、显式类型、声明式配置、统一命名前缀

### 文档偏好
- LLM 友好优先（结构稳定、显式状态字段、可追溯链接）
- 修改必留下进展记录（progress → reports/completed）
- 双层记忆架构（daily 流 + MEMORY 沉积）

---

## 项目上下文

### 项目定位
构建某商业竞品的**开源替代品**。竞品是 ToB 实时监控 + 互动客服 / 营销转化平台。本项目不做客户获取与销售，专注技术核心。

### 当前阶段
**v1 主干完全收口 + vendor-rrweb 完成 + page-editor post-v1 3 切片完成 + Go 覆盖 90%+（2026-06-26）**：
- 1a-1z 全切片 + 测试深化 + e2e acceptance + 重命名(2026-06-17~20, 90+ commits)
- vendor-rrweb 硬分叉(2026-06-25~26, 21 commits, 合并至 master)
- ReplayPlayer 修复 + fork-3b 测试 + record 测试 (+117 tests, replay-core 146 全绿)
- Go 后端覆盖: recording 90.7% / api 90.0% / storage 91.5%
- **page-editor**: post-v1 页面编辑器全 3 切片(pe-1 proto+PG+Go+API / pe-2 admin UI / pe-3 SDK 配置驱动)
- **端口迁移**: 全部开发端口移至 7000-7100 范围(7080/7073/7074/7032/7079/7020/7021)

**测试信心补全完整路径**(2026-06-19):
- 审计:31 切片 badge 系统性虚标,28 T0 + 40 T1 + 30 T2 + 10 T3 = 108 gap
- 1ac + 1ac-final:28/28 T0 关闭 + 2 代码 bug 修复(deleteVisitor admin role + operatorWS auth)
- 1ad:40/40 T1 关闭
- 累计 badge:**🟢 ×23(4 strict + 1 aligned + 18 touched) / 🟡 ×9 / 🔴 ×0**
- 7 个原 🔴 切片全部升 🟡/🟢(1d→🟡, 1g→🟢, 1h-backend→🟢, 1k→🟡, 1l→🟡, 1s→🟡, 1y→🟢)
- 剩余 T2/T3(40 项,~15 小时)留 backlog

**接口化重构续做**(2026-06-19~20,1ag~1ai-h):
- 1ag-1ah 行为级测试(api handler) + 1ai storage repo PG 集成
- 1ai-b~1ai-h 接口化模式 PoC:AuthHandler / ClaimHandler / ChatHandler / CommandHandler / requireClaimOwnership / SessionHandler 全部接口化
- 累计 +75 测试,api 包覆盖实测 **38.2%**(docker 环境),storage 包实测 **57.6%**(docker 环境;repo 函数 70-100%,适配器 0%);本地无 docker 时显示假低数字(api 28.7% / storage 1.5%),详见 [`audits/2026-06-20-coverage-assessment.md`](../docs/audits/2026-06-20-coverage-assessment.md)

**marketing-monitor → pinconsole 重命名**(2026-06-20):
- Go module / pnpm scope / docker-compose / DB schema / Go embed / 活跃文档 / CI / e2e 全量重命名 5 步
- 历史快照报告 22 份 + e2e 测试 4 份中的 `@marketing-monitor/*` 残留已处理(批量替换 + 历史断言保留)

详见 [`docs/audits/2026-06-19-test-confidence-audit.md`](../docs/audits/2026-06-19-test-confidence-audit.md) + [`docs/project-status.md`](../docs/project-status.md) §2.1/§2.2/§5。

下一步候选(post-v1):
- **post-v1 路线** — 自定义域名 / 页面编辑器 / Tauri(详见 [`PLAN.md`](../PLAN.md) §8)
- T2/T3 backlog(40 项,~15 小时)— admin/SDK i18n keys + minor 路径,不阻塞 v1 release

### 范围边界
- **不做（产品本身，永久）**：多租户 SaaS、计费、注册流、云托管服务
- **v1 不做（post-v1 切片）**：页面编辑器、Tauri 桌面端、自定义域名
- **v1 做**：监控 + 全套互动 + 录像 + 反爬虫 + i18n + CI（已完成）
- **maintainer 营销层（2026-06-22 解禁）**：`marketing/` 独立目录做 maintainer 自有官网 + 咨询转化（Astro + Cloudflare Pages + Workers + D1）；**与 OSS `landing/` 严格区隔**——OSS 用户部署不包含 maintainer 营销代码，用户实例不收集询盘数据。详见 [`docs/progress/2026-06-22-landing-readme-design.md`](../docs/progress/2026-06-22-landing-readme-design.md)

### 部署目标
- OSS 自托管为主
- docker-compose 一键起（Go + PG + Redis + MinIO）
- 单租户，schema 预留 `tenant_id` 不激活

### 节奏
- 单人开发
- v1 切片估时：solo 全职 14-17 周；业余 9-12 个月

---

## 关键决策

完整理由见 [`PLAN.md`](../PLAN.md)。本节是索引。

### 架构（不可重新讨论）

- License：AGPL-3.0
- 仓库：Monorepo，Go embed 静态资源
- 后端：Go + Gin + coder/websocket + 自定义 hub
- 存储：PostgreSQL + Redis + MinIO
- 前端：Vue 3 + TS + Vite + Pinia + Vue I18n（中英 day 1；Element Plus 已在 1q 移除，手撸样式 + Calm Crafted 设计 token）
- 管道：中心化 hub-and-spoke（无 WebRTC、P2P）
- 租户：单租户 + schema 预留

### v1 切片技术

- SDK：rrweb 全量采集 + 提交前按键 + 选择性截图（canvas/iframe）
- co-browsing：rrweb 双向 + 节点 ID 选择器 + 防抖 300ms 代填
- 多运营：1:1 锁定
- 认证：Email/password + bcrypt + HttpOnly cookie
- 域名：v1 仅平台域名
- 反爬虫：rate limit + UA + 行为 + fingerprint（中等深度）

### rrweb 硬分叉自维护（2026-06-25/26 完成）

- **决策**：放弃 rrweb npm 依赖（alpha.20），拷上游 TS 源码进 `packages/replay-core`（单包，分 snapshot/record/replay/types）硬分叉自维护。**彻底删 Svelte 版 rrweb-player，admin 原生 Vue 重写 player** 直接持有 Replayer。
- **理由**：回放端 90% 是和 rrweb-player 内部搏斗的 hack；同时拥有两端 mirror 后 nodeID 成跨端精确指针（根治 1f 坐标 fallback + 删 `nodeMap.ts`）；上游活跃度降、稳定版难产，放弃上游补丁代价小。
- **执行**：`feat/vendor-rrweb` 分支 21 commits, fork-0~4 全部完成，已合并至 master（`82a7a355`），净减 ~600 行 hack 代码。
- **总纲 spec + impl**：[`docs/reports/completed/2026-06-25-vendor-rrweb-spec.md`](../docs/reports/completed/2026-06-25-vendor-rrweb-spec.md) + [`impl`](../docs/reports/completed/2026-06-25-vendor-rrweb-implementation.md)
- **Remaining**：fork-3b 上游测试转 Playwright(5 组)留 backlog

### Maintainer 营销层（2026-06-22 17 轮 grill-me 共识）

详见 [`docs/progress/2026-06-22-landing-readme-design.md`](../docs/progress/2026-06-22-landing-readme-design.md)。索引：

- **策略**：v1 完成后，maintainer 接受 ToB 咨询（PLG + 咨询式）。产品本身仍 OSS 自托管 / AGPL / 不计费 / 不做注册流
- **目标客户**：国内 ToB/SaaS 决策者（CEO/CTO/运营/市场），正在用美洽/智齿/Udesk/风车/小能/Live800 等
- **Hero hook**：数据主权（"你的访客，你的数据。"）
- **CTA**：留资表单（Cloudflare Worker + D1）+ 文档次级
- **不提竞品名**：只讲类别（国内不正当竞争法 + 投诉文化风险）
- **语言**：中英镜像同内容互译（同 i18n from day 1 原则）
- **IA**：单页长滚动 8 段漏斗型（Hero/Problem/Features/数据主权/Self-host/Roadmap/FAQ/FinalCTA）
- **视觉**：Calm Crafted 营销变体（复用 docs/design-system.md `--pc-*` tokens，加大字号 hero + 更宽留白 + Gentle & Restrained 动效）
- **技术栈**：Astro + Vue → Cloudflare Pages + Workers + D1
- **仓库结构**：`marketing/` 独立目录与 OSS `landing/` 区隔（避免营销代码污染用户部署）
- **README**：顶部加营销概述（tagline + 4 个 badge + 决策者引导）+ 保留 dev 主体

**Phase 进度**：
- ✅ Phase 1：marketing/ Astro 骨架 + Calm Crafted 营销变体 tokens + 中英双语路由 + 8 段组件 + 表单/Worker/D1 schema 代码（2026-06-22）
- ✅ Phase 2：5 张真实截图（dashboard/cobrowse-active/replay/chat/privacy）+ Features 真图 + Hero Live demo 徽章 + video poster 兜底；**仅 defer 真实 demo.mp4 录制给用户**
- ✅ Phase 3：本地 wrangler pages dev + D1 端到端验证通过（POST → D1 INSERT、honeypot 静默、validation 400、UI Vue 表单 → D1 全链路）；修 2 bug（`prerender = false` 漏加 / `message != null` 容忍 undefined）；**仅 defer `wrangler login` + remote D1 create 给用户**
- ⏳ Phase 4：README 顶部 + PLAN.md §1 + MEMORY.md 同步 ✅；**仅 defer Cloudflare Pages deploy + 自有域名绑定给用户**

### 工作流

- 文档分层：根（事实来源）+ `docs/`（工作文档）+ `memory/`（记忆）
- 切片序：1a → 1b → ... → 1j，不跳步
- 进展即记录：每次修改都有 progress/completed 文档

### 前端设计基线（2026-06-21 锁定，14 轮 grill-me 访谈共识）

详见 [`docs/design-system.md`](../docs/design-system.md)。索引:

- **美学**：Calm Crafted（软中性 + 人文 sans + 柔影 + 8px 圆角）
- **字体**：IBM Plex Sans + Sans SC + Mono（self-host woff2）
- **色彩**：Stone（暖中性）+ Teal（主色）+ Amber/Forest（信号）
- **密度**：Calm Macro + Dense Micro（页 32px / 数据行 8px）
- **明暗**：Light + Dark 跟随系统（无 toggle UI，token 双模 day 1）
- **圆角阴影**：8/12/16 + 多层堆叠 box-shadow
- **图标**：Phosphor Icons（代替 emoji 🚩 和手撸 SVG）
- **动效**：Gentle & Restrained（120-240ms，无 stagger 揭示）
- **App Shell**：Top bar + Content
- **Login**：Centered Card on Cream（代替禁用紫渐变）
- **Dashboard**：3-Column Smart Expand（默认 2 栏，claim 后 3 栏）
- **Visitor SDK**：Consistent Brand（与 admin 同 token，前缀 `--pinconsole-*` 避免宿主冲突）
- **Token 架构**：CSS Variables + Scoped CSS（无 Tailwind / UnoCSS）
- **Replay**：Custom Calm Chrome + rrweb API（隐藏原生控制器）

**反 AI-slop 红线**（写入 docs/design-system.md §1.2）：禁紫渐变 / Inter/Geist/Roboto / slate+indigo / emoji-as-icon / Element Plus 默认蓝 `#409eff` / 单层 box-shadow / 首屏 stagger 揭示。

**实现未开始**——本会话仅设计访谈。迁移计划 6 phase / 10-15 天 solo 全职（详见 design-system.md §8）。

---

## 经验教训

> 触发条件：发现错误修复模式、踩过的坑、值得未来回顾的判断。

### 接手项目先做 reality check(2026-06-18)

**Why**:2026-06-18 接手项目时,文档声称 1a-1j 全部 ✅ completed,但实际:
- 只有 2 个 commit,5K+ 行未提交
- 1c-1j 完成期间无任何 daily 笔记
- 多处文档自相矛盾(status doc §2/§5/§7 三处对不上)
- e2e 测试 39/39 pass 但其中 5 个是浅测(只 `resp.ok()` 或 `if (!length) return` 静默跳过)
- HeadlessChrome UA ban 是死代码(新版 Playwright UA 是 `Chrome/...`)

只读文档会误判项目真实状态。**先做 reality check 再做架构层判断**。

**How to apply**:接手任何"声称已完成"的项目时,先跑:
1. 静态:`go vet` + `go build` + `pnpm typecheck` + `pnpm lint`
2. 单测:`go test ./...` + `pnpm test`
3. e2e:启动 infra + 跑 Playwright
4. 手测最小链路
5. 看测试是否真验证(读断言、找静默跳过)
6. 看文档是否内部一致

### 切片"完成"必须有深度判定

**Why**:若完成报告只标"✅ done",未来 Claude 会把所有切片视为同等可信。实际 1a-1g 是端到端深度验证(🟢),1h/1i 是 e2e 通过但浅(🟡),1j 是零专属测试(🔴)。

**How to apply**:每份完成报告顶部加深度 badge;深度判定遵循 [`docs/standards/verification-depth.md`](../docs/standards/verification-depth.md) R2 rubric。新增测试时要说明升级了哪个切片的深度。

### 浅测的常见模式(自动降级 🟡)

**Why**:1h/1i 测试 pass 但没真验证功能,这是常见的"假绿"模式。

**How to apply**:看到以下任一,自动判定为 🟡:
- `if (!x.length) return;` 类静默跳过(空 DB 时测试不报错但不验证)
- 断言只是 `expect(resp.ok()).toBeTruthy()`(没验证返回内容)
- 测试名说"端到端"但实际只调 `request.post('/api/...')`(无 UI 流)
- 测试名说"中间件存在"但只 GET `/healthz`(没触发中间件逻辑)
- 安全/边界类切片(认证/反爬/跳转)缺负向测试(没测 401/403/429)

### Reality check 必须含 spec vs 实施对照(2026-06-18 升级)

**Why**:2026-06-18 spec 对照发现 1h/1i/1j 三处重大 gap——
- 1h: spec 决策 #5 "登录 UI + Vue Router 守卫" **完全未实施**(后端做完了,前端 0%)
- 1i: spec 决策 #3 "行为分析" `BehaviorTracker` 代码完整但 **server/ 零调用方**(死代码)
- 1j: spec 决策 #1 "i18n 全部" 主视图用 key,**子组件仍硬编码中文**

测试 39/39 通过掩盖了这些 gap,因为测试不验证 spec 决策点是否满足。

**How to apply**:reality check 流程从 6 步升级为 7 步,**新增第 7 步 spec 对照**:
1. 静态(`go vet` + `go build` + `pnpm typecheck` + `pnpm lint`)
2. 单测(`go test ./...` + `pnpm test`)
3. e2e(启动 infra + 跑 Playwright)
4. 手测最小链路
5. 看测试是否真验证(读断言、找静默跳过)
6. 看文档是否内部一致
7. **新增:逐切片读 spec 决策表 → 逐项 grep 代码 → 标注未实施/未接线/部分实施**

特别警惕:
- 代码定义完整但**无调用方**(死代码)
- spec 要求的 UI/view **完全缺失**
- spec 要求"全部 X"但实际只是"主路径走 X,边缘仍 Y"

### 完成报告 ✅ 不等于 spec 全部 acceptance 满足

**Why**:1h 完成报告写 "completed" 但 spec 验收 "admin /admin/login 页 + 路由守卫" 未满足。LLM 倾向把决策表逐项打勾而不验证代码路径。

**How to apply**:读完成报告时,**对照原 spec 的 acceptance** 而不是看报告自评状态。spec 是 source of truth,报告只是叙述。

### e2e acceptance 必走 grill-me + 策略 A(2026-06-18)

**Why**:2026-06-18 用 Playwright 对 v1 主干做严格验收。策略 A(行为性失败必修)让我们抓到 **4 个 production bugs**,都不是单测能发现的:
1. server `/api/auth/me` 未挂 AuthMiddleware → 永远 401 → SPA 刷新全跳 login
2. SPA router 守卫与 fetchMe 时序竞争 → 首次 navigation 必跳 /login
3. 1z P1-1 修复时漏了 Dashboard.vue 的裸 fetch → trace_id 端到端断裂
4. Playwright 默认 APIRequestContext UA 是 HeadlessChrome → 被 1i antiscrape 拦截

**How to apply**:
- 任何"v1 已完成"的声明,用 Playwright 全量 e2e 兜底验收
- e2e fixture 用 UI login 而非 API login(规避 SPA fetchMe 时序 bug)
- e2e fixture 必须显式注入干净 Chrome UA(规避 HeadlessChrome 黑名单)
- regression 测试是修复完整性的最终保障 — 1z P1-1 声称全覆盖,被 01-trace-id 抓到 Dashboard 漏洞
- claim 必须显式调(1k P0-3 release binary 强制 requireClaimOwnership)

### e2e acceptance 不是终点 — 真实 UI 操作会暴露单测覆盖不到的 bug(2026-06-18 v1-followups)

**Why**:v1 e2e 65 测试全绿后,真实使用又抓到 **5 个生产 bug**(详见 [`docs/reports/completed/2026-06-18-v1-followups.md`](../docs/reports/completed/2026-06-18-v1-followups.md)):
1. v1-replay player iframe 渲染失败(Vue ref + iframe mount + player library 三方时序耦合)
2. selectedSessionId 被 visitor offline 错误清空(SPA 状态管理 bug)
3. iframe 切换 session 不重 mount player(watch 错过 fingerprint 变化)
4. visitor-sdk SDK reload 不续接 session(`Partial<config>` 显式 undefined 覆盖 DEFAULTS)
5. co-browse `listMessages` 过严要求 claim(只读端点不应要授权锁)

**How to apply**:
- v1 任何"完成"声明后,必走**真实 Playwright UI 操作**(点击/导航/刷新),不只是 API 断言
- **iframe 渲染是 SPA 状态机高风险区**:e2e 难以稳定模拟,必须手动 UX 测试
- **claim 锁只用于写/control 端点**:只读端点(list/get history)不要求 claim;接入 `requireClaimOwnership` 时易过度收紧
- **session 续接靠 client 持久化**:server 是被动方,SDK 必须主动在 `sessionStorage` 保存 `session_id`

### `Partial<T>` 配置合并的 undefined 陷阱(2026-06-18 v1-followups)

**Why**:visitor-sdk 曾出现 `mm.init({ apiBase: undefined })` 后 apiBase 变 undefined 而非默认值。TypeScript 的 `Partial<T>` 允许显式 undefined,但 spread operator 不区分"未设置"和"显式 undefined"。

**How to apply**:任何 `{...DEFAULTS, ...userConfig}` 模式必须先 `dropUndefined`:

```typescript
function dropUndefined<T extends object>(obj: T): T {
  return Object.fromEntries(
    Object.entries(obj).filter(([, v]) => v !== undefined)
  ) as T;
}
```

适用于所有 SDK / library 的"用户配置 + 默认值合并"场景。

### Badge 自报 🟢 系统性虚标 — 修代码 ≠ 补测试(2026-06-19 测试信心审计)

**Why**:deep-audit(2026-06-18)闭环 13 个 P0 后,`project-status.md` §5 把 31 切片标 🟢。2026-06-19 测试信心审计发现 **20 个虚标**:

- 1l GDPR 1v 修了 `ErrNoRows` bug,但 **erasure 级联(PG+MinIO+Redis)和 GC(5 表)完全无回归测试**
- 1k 修了 P0-3/P0-4 越权和 race,但**非 owner 403、SET NX race、Lua compare-and-del 全无单测**
- 1y 实现了 WS rate limit,但 **close + FlagSession 触发条件无测试**
- 1s lifecycle 装饰器加到 5 个 handler,**但全部 13 个集成点/分支/外部调用点无测试**

deep-audit 修了"代码 bug",但没补"回归测试"。**下次重构 bug 会复发**。

**How to apply**:

1. **修 bug 时同步补回归测试**——P0/P1 类修复必须包含"如果 bug 复现则测试失败"的负向测试
2. **完成报告 badge 不能自报**——必须经过 spec 决策点逐项对照(见下方"测试信心审计方法")
3. **`verification-depth.md` 已升级**:🟢 内部分 strict/aligned/touched 三级 + T0~T3 测试 gap 严重度尺度(独立于 P0~P3 代码 bug 尺度)
4. **新切片完成时,必须按 spec 决策逐项打勾**——不只是"功能跑通"

### 测试信心审计方法(2026-06-19)

**Why**:此前 reality check 流程(7 步)已包含 spec 对照,但**对照是手动的、非系统的**。2026-06-19 用 grill-me 13 问达成共识后,固化了系统化的 spec→test 对照方法。

**How to apply**:对未来任意切片组做测试信心审计:

1. **方法 = A+F + B spot-check**:A=spec 决策点列表,F=spec→test traceability 矩阵,B=mutation testing 在高风险切片 spot-check
2. **spec 源 = hybrid**:有 spec 文档用 spec;无 spec 用 START/PLAN;无 spec 无顶层决策用 impl 报告目标段
3. **执行 = 阶段 1 并行定位 + 阶段 2 顺序判定**:阶段 1 subagent 只填"决策 ID + source + 实际测试位置",不判 severity;阶段 2 单一 auditor 用统一 rubric 判
4. **severity = T0~T3**(独立于 deep-audit 的 P0~P3):T0=critical 路径无测试,T3=测试存在但弱
5. **熔断**:≥10 切片且 T0=0 或总 gap >30 → 停止扩展,聚焦 T0/T1
6. **deliverable = 诊断 + 降级 + 修复 plan**(不动代码)

详见 [`docs/audits/2026-06-19-test-confidence-audit.md`](../docs/audits/2026-06-19-test-confidence-audit.md) + [`docs/standards/verification-depth.md`](../docs/standards/verification-depth.md) §2.5/§2.6。

### 覆盖率必须带 docker 跑 + 自报数字一律不信(2026-06-20)

**Why**:`project-status.md` §2.1 此前写 "api 28.7% / storage 1.5% / antiscrape 21.4% / recording 12.8%",2026-06-20 启动 docker compose + `CREATE DATABASE pinconsole` + 跑测试后实测:

- api 实测 **38.2%**(commit `b748e43` 自报准确,§2.1 错)
- storage 实测 **57.6%**(1ai/1ai-b 报告准确,§2.1 错得离谱)
- antiscrape 实测 **86.7%**(Explore agent 报 21.4% 是环境/缓存问题)
- recording 实测 **48.0%**(同上)

**根因**:
1. 测试代码 `internal/storage/erasure_test.go:34` 硬编码 `localhost:5432/pinconsole`,但 `.env` 的 `PG_DB=marketing_monitor`(rename 重构遗漏)让 docker 实际起的数据库叫 `marketing_monitor` → 名字不匹配 → 所有 PG 集成测试 skip → 显示 1.5% 假低覆盖率
2. 之前的实测都没启动 docker,直接采了"本地无依赖"的虚假低数字
3. commit message 自报的数字反而是对的(因为开发时跑过完整 docker 测试)

**How to apply**:
1. **任何覆盖率评估必须先 `docker compose up -d postgres redis minio` + `pg_isready` 确认就绪**
2. **测试代码硬编码的 DB 名 vs docker 实际 DB 名必须一致**——`.env` 改 DB 名时同步检查 `*_test.go` 中的硬编码连接串
3. **commit message 自报覆盖率 vs 文档实测覆盖率冲突时,优先信 commit message**(开发时跑过完整环境的概率更高)
4. **`go test -coverprofile` + `go tool cover -func` 是函数级真相**——`go test -cover` 的包级数字可能受 skip 误导
5. **TS 用例数必须 grep `^\s*(it|test)\(` 实测**,不能信切片报告自报(1aa 报告低估 34%)
6. **vitest coverage 未配置时,前端"行覆盖率"不存在**——只能说"测试用例数",不能说"覆盖率 X%"

详见 [`docs/audits/2026-06-20-coverage-assessment.md`](../docs/audits/2026-06-20-coverage-assessment.md)。

---

## 外部依赖与索引

| 项 | 路径 |
|---|---|
| 项目状态快照（LLM 必读） | [`docs/project-status.md`](../docs/project-status.md) |
| 架构 + 产品定位事实来源 | [`PLAN.md`](../PLAN.md) |
| Claude 工作指南 | [`CLAUDE.md`](../CLAUDE.md) |
| 文档规范 | [`docs/standards/doc-structure.md`](../docs/standards/doc-structure.md) |
| 命名规范 | [`docs/standards/naming-conventions.md`](../docs/standards/naming-conventions.md) |
| 验证深度判定标准 | [`docs/standards/verification-depth.md`](../docs/standards/verification-depth.md) |
| 前端设计基线 | [`docs/design-system.md`](../docs/design-system.md) |
| rrweb 硬分叉总纲 | [`docs/reports/completed/2026-06-25-vendor-rrweb-spec.md`](../docs/reports/completed/2026-06-25-vendor-rrweb-spec.md) |
| rrweb 硬分叉实施报告 | [`docs/reports/completed/2026-06-25-vendor-rrweb-implementation.md`](../docs/reports/completed/2026-06-25-vendor-rrweb-implementation.md) |
| 每日笔记（流层） | [`memory/daily/`](./daily/) |

---

## 维护

- 本文件代表**当前**真实状态。信息过时时立即更新或删除，不保留"历史版本"在文件内（用 git 历史追溯）。
- 类别新增/删除按需调整，但保持大类稳定（用户偏好 / 项目上下文 / 关键决策 / 经验教训 / 外部依赖）。
- 触发更新的场景：用户陈述新偏好、关键决策变化、发现新模式、外部依赖变更。
