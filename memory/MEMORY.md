# MEMORY.md — 项目长期记忆（沉积层）

> 本文件是项目的长期记忆。记录"当前真实状态"——用户偏好、项目上下文、关键决策、经验教训。
> 触发更新：用户陈述偏好、发现错误修复模式、建立项目规则、关键决策变化。
> 与 [`memory/daily/`](./daily/) 的关系：daily 是不可变日志（流），MEMORY 是当前状态（沉积）。

**最后更新**：2026-06-18（v1 e2e acceptance 完成,65 测试全绿 + 4 production bugs 修复）

---

## 用户偏好

### 语言
- **交流与文档**：中文
- **生成代码**：英文
- 例外：纯中文业务术语保留中文（如"运营"、"访客"、"落地页"）

### 工作风格
- 推荐方案时直接给出首选 + 理由，不堆叠"on the other hand"
- 偏好"先架构后实施"——重大决策不绕过 grill-me 风格访谈
- 范围控制严格——"不考虑客户和销售"是硬约束，不要替用户加 SaaS/计费/多租户

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
**v1 全切片 e2e acceptance 完成(2026-06-18)**。65 passed / 0 failed / 4 skipped(gated prod-mode + docker-prod 测试)。深度分布:**全部 🟢 verified-deep**(1a-1z 全切片)。详见 [`docs/reports/completed/2026-06-18-v1-e2e-acceptance.md`](../docs/reports/completed/2026-06-18-v1-e2e-acceptance.md)。

下一步候选:
- **1y visitor-ws-rate-limit**(in_progress,实施未完成)— docs/progress/ 中
- **admin SPA 显示 flagged 标记**(1w P1-29 后端已就绪,UI 未消费)
- **prod-mode e2e CI job**(1k/1l gated tests 需要)

### 范围边界
- **不做**：多租户 SaaS、计费、注册流、营销页
- **v1 不做**：页面编辑器、Tauri 桌面端、自定义域名（这些是 v1 之后的切片）
- **v1 做**：监控 + 全套互动 + 录像 + 反爬虫 + i18n + CI

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
- 前端：Vue 3 + TS + Vite + Pinia + Element Plus + Vue I18n（中英 day 1）
- 管道：中心化 hub-and-spoke（无 WebRTC、P2P）
- 租户：单租户 + schema 预留

### v1 切片技术

- SDK：rrweb 全量采集 + 提交前按键 + 选择性截图（canvas/iframe）
- co-browsing：rrweb 双向 + 节点 ID 选择器 + 防抖 300ms 代填
- 多运营：1:1 锁定
- 认证：Email/password + bcrypt + HttpOnly cookie
- 域名：v1 仅平台域名
- 反爬虫：rate limit + UA + 行为 + fingerprint（中等深度）

### 工作流

- 文档分层：根（事实来源）+ `docs/`（工作文档）+ `memory/`（记忆）
- 切片序：1a → 1b → ... → 1j，不跳步
- 进展即记录：每次修改都有 progress/completed 文档

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

---

## 外部依赖与索引

| 项 | 路径 |
|---|---|
| 项目状态快照（LLM 必读） | [`docs/project-status.md`](../docs/project-status.md) |
| 架构事实来源 | [`PLAN.md`](../PLAN.md) |
| 产品事实来源 | [`START.md`](../START.md) |
| Claude 工作指南 | [`CLAUDE.md`](../CLAUDE.md) |
| 文档规范 | [`docs/standards/doc-structure.md`](../docs/standards/doc-structure.md) |
| 命名规范 | [`docs/standards/naming-conventions.md`](../docs/standards/naming-conventions.md) |
| 验证深度判定标准 | [`docs/standards/verification-depth.md`](../docs/standards/verification-depth.md) |
| 每日笔记（流层） | [`memory/daily/`](./daily/) |

---

## 维护

- 本文件代表**当前**真实状态。信息过时时立即更新或删除，不保留"历史版本"在文件内（用 git 历史追溯）。
- 类别新增/删除按需调整，但保持大类稳定（用户偏好 / 项目上下文 / 关键决策 / 经验教训 / 外部依赖）。
- 触发更新的场景：用户陈述新偏好、关键决策变化、发现新模式、外部依赖变更。
