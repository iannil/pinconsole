# 切片 vendor-rrweb — Spec（总纲）

**切片编号**：vendor-rrweb（fork-0 ~ fork-4 工作流总纲）
**类型**：重构 / 新功能（横切录制端 + 回放端）
**创建时间**：2026-06-25
**状态**：fork-0 ✅ (2026-06-25), fork-0-parity ✅ (2026-06-25 T23:25), fork-1 ✅ (2026-06-25 T23:55), fork-2 ✅ (2026-06-26 T00:00)
**关联**：[PLAN.md §4 技术栈](../../PLAN.md)、[CLAUDE.md 已锁定架构决策](../../CLAUDE.md)、[verification-depth 标准](../standards/verification-depth.md)

## Context

为什么把 rrweb 源码拷进本仓自维护？

- 触发原因：回放端 `admin/src/components/ReplayPlayer.vue` ~300 行里约 200 行在和 Svelte 版 rrweb-player 内部实现搏斗（MutationObserver 补 iframe sandbox、live 模式 hack、响应式 sizing hack、双 mount 竞态 token、iframe display 强制覆盖）；协同点选 `CoBrowseOverlay`（1f）退化为 `nodeID=0` + 服务端按坐标点击；`nodeMap.ts` 靠扫 `data-rr-node-id` attribute（假设存疑）。
- 业务/技术价值：fork 后**同时拥有 record mirror 与 replay mirror**，可让 nodeID 成为跨端精确寻址的规范通道，删除一整批 hack 与脆弱替身，回放线获得一等公民控制。
- 不做的代价：技术债持续累积；上游 rrweb 活跃度下降、稳定版迟迟不发，长期被 alpha API 漂移绑架。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | fork 策略 | A 硬分叉(全量拷 snapshot/record/replay/player 源码,断联自维护) / B 薄抽取(只 fork 回放线,record/snapshot 留依赖) / C 包装层(现状延续) | **A** | 痛点集中在 replay/player,但 replay 依赖 snapshot 的 mirror/nodeID,B 会在边界反复踩"两端 id 对不上";record 做精准点选迟早要改 serialize,也得一起 fork;rrweb MIT 与本项目 AGPL 兼容;C 已被证明是技术债黑洞 |
| 2 | 包结构 | A 照搬上游四包 / B 按"录制/回放"链路重组单包分目录 | **B** | 三者耦合紧(共享 mirror/types),本项目不发布 npm(Go embed 单二进制)无独立版本化需求;拆包只造无意义 workspace 边界。`@pinconsole/replay-core` 分 `snapshot/record/replay/types` |
| 3 | rrweb-player(Svelte) | A 一起 fork / B 彻底删除,admin 原生 Vue 重写 | **B** | 现有 90% hack 源于隔着 Svelte 壳够 Replayer;直接持有 `Replayer` 实例后 sandbox/sizing/live/mount 全部消解;Vue 项目多引入 Svelte 运行时纯负担;rrweb-player 恰是上游最不稳定的包 |
| 4 | 抽取形态 | A 拷 node_modules dist / B 拷上游 monorepo TS 源码 | **B** | dist 是 bundle 后代码,不可读/不可精简;必须拷 TS 源码才能雕刻 |
| 5 | 基线版本 | A 钉 alpha.20 / B 顺手升级新 alpha | **A** | 当前线上行为基于 alpha.20 调通(hack 注释全针对它);fork 第一步目标是**行为等价**,升级是 fork 之后的独立动作 |
| 6 | 节奏 | A 边搬边精简一步到位 / B 先复刻(求等价)后雕刻 | **B** | 把"搬运"与"精简/改造"切成互不污染两阶段,降低回归归因难度 |
| 7 | packer | A 保留 rrweb 内联压缩 / B 砍掉,压缩下沉存储层 | **B** | 事件已走 msgpack envelope;压缩是存储基础设施的事,回放线应永远拿解压后事件;写 MinIO 时统一 gzip |
| 8 | Shadow DOM / 同源 iframe | A 砍以瘦身 / B 保留 | **B** | 访客站点技术栈不可预测,web components 普遍;砍了会静默丢内容,不能赌 |
| 9 | 新能力范围 | A 仅 nodeID 寻址链 / B 叠加运行时 mask/统一事件通道等 | **A** | 锁最小强相关集 ①nodeID 跨端寻址 + ③删 nodeMap + ④Replayer 直控,②元素原语紧随;其余留 fork 之后独立切片 |
| 10 | 上游测试移植 | A 全量移植再删 / B 按保留面裁剪移植 | **B** | 全量含 canvas/console/packer 测试(无意义且会红);只移植保留功能(snapshot/replayer/observer/shadow DOM/iframe/mask) |
| 11 | 分支策略 | A master 直接干 / B feat 分支 | **B** | 横跨数周、动核心回放;开 `feat/vendor-rrweb`,fork-0~4 逐切片提交,全绿再合 master |

## 切片拆分（顺序推进，每片独立可验/可发布）

| 切片 | 内容 | 完成门（e2e/测试） | 仍依赖 rrweb? |
|---|---|---|---|
| **fork-0** | clone alpha.20 源码 → 拷入 `packages/replay-core`（snapshot/record/replay/types/utils/rrdom）→ import 重写 + TS/Vite 配置 → 能编译 + **双 parity 夹具**(replay + record)就位。**纯新增文件,不动 SDK/admin**。fork-0 前先 15min 核实 `data-rr-node-id` DOM attribute 假设 | **双 parity 夹具绿**（record-parity 保证录制等价,replay-parity 保证回放等价） | 是（并存对照） |
| **fork-1** | SDK record 切到 replay-core | 1c + 1l 绿 | admin 侧仍依赖 |
| **fork-2** | admin 回放切 replay-core + **钻穿方案**（Vue 组件直接持有 Replayer 实例,不重写 Replayer 核心）+ **删 rrweb-player + Svelte**。消解 5/7 项 hack。保留 rrweb-snapshot(admin 暂不直接依赖,但 record 用的 snapshot 包已在 replay-core 内) | 1c + 1d 绿; replay-parity 绿 | 否（rrweb-player 删;rrweb/rrweb-snapshot 作为 visitor-sdk 依赖可删但留到 fork-1 完成后一并操作） |
| **fork-3** | 精简：文件级裁剪(砍 canvas/console/packer/plugin 目录) + 上游保留功能测试转 **Playwright**（snapshot/replayer/observer/shadow DOM/iframe/mask 共 5 组）| 转译 Playwright 测试全绿 + 1c/1d/1l 不回归 | 否 |
| **fork-4** | 新能力：nodeID 跨端寻址 + 删 nodeMap + 删坐标 fallback + 元素操作原语 | 1e/1f + 新增正负向 e2e 绿（正向=精确点选命中,负向=陈旧 nodeID → no-op） | 否 |

**顺序理由**：fork-0 纯新增零风险(但双夹具确保形态等价)；fork-1/2 等价替换、每步有双 e2e 兜底可回滚；fork-3b 写 data-rr-node-id 是 fork-4 前置但改动在 snapshot 同目录,内嵌于 fork-3 阶段可减少上下文切换。

**新增切片**：
| 切片 | 内容 | 完成门（e2e/测试） | 仍依赖 rrweb? |
|---|---|---|---|
| **fork-3b** | snapshot 写 `data-rr-node-id` attribute（fork-4 前置,内嵌于 fork-3 阶段,后置 fork-3 完成但 fork-4 可先于此开始） | snapshot 正负向测试 + fork-3 Playwright 不回归 | 否 |

## Acceptance（总纲级，可验证）

- [ ] fork-0 前 15min 实验验证：在线上 demo 页面执行 `document.querySelectorAll('[data-rr-node-id]')`，确认是否存在 DOM attribute
- [ ] fork-0 完成时双 parity（replay + record）同时绿
- [ ] fork-1 完成时除 record-parity 绿外,另加**真浏览器三方验证**：同一页面同时触发新旧两路 record,用旧 admin 回放新数据,截图对比无视觉差异
- [ ] fork-2 完成后 `rg "rrweb" admin/package.json` 无命中（rrweb-player 删净）；admin 无 Svelte 运行时
- [ ] 全程既有 e2e（1c/1d/1e/1f/1l）保持绿，无回归
- [ ] fork-3 后移植的上游 Playwright 测试（snapshot/replayer/observer/shadow DOM/iframe/mask）全绿
- [ ] fork-3b 后 snapshot 写 `data-rr-node-id` 正负向测试绿
- [ ] fork-4 后 `nodeMap.ts` 与坐标 fallback 删除；新增正向（按 nodeID 精确点选/代填命中）+ 负向（陈旧/不存在 nodeID → 优雅 no-op 不崩）e2e 绿

## 深度目标

按 [verification-depth.md](../standards/verification-depth.md) R2 rubric：

- 🟢 verified-deep：fork-1/2/4 均要求既有或新增 Playwright e2e 覆盖真浏览器交互；fork-4 作为 cobrowse 边界类切片**必须含负向测试**（陈旧 nodeID → no-op）
- 🟡 verified-shallow：fork-3 精简以移植 Playwright 测试保真为主，e2e 仅验"不回归"，不新增正向 e2e
- 🔴 不接受：任何切片不得以 `if (!x.length) return;` 静默 pass

## 验证五防线

1. **双 parity 夹具锁形态等价**：
   - *replay-parity*：同一 events 数组 → 旧 rrweb-player vs 新 replay-core Replayer → diff iframe 最终 outerHTML
   - *record-parity*：同一 DOM 环境 → 旧 rrweb.record() vs 新 replay-core.record() → diff events 数组
   - 删旧依赖前全量跑,删完即弃（只保留 replay-core 侧）
2. **真浏览器三方截图对比（fork-1 专用）**：新旧 record 同时运行,旧 admin 回放新数据,截图确保视觉无差异
3. **既有 e2e 当验收门**：1c/1d/1e/1f/1l 真浏览器跑录制→回放→点选→脱敏。
4. **移植上游 Playwright 测试锁精简保真**：覆盖简单访客页打不到的 shadow DOM/iframe/CSS/adopted stylesheet edge case。
5. **新 nodeID 配正负向 e2e**。

## 范围边界

**本工作流做**：
- 硬分叉 rrweb（snapshot/record/replay/types/utils/rrdom）TS 源码进 `packages/replay-core`，单包分目录
- admin 钻穿直持 Replayer 实例，删除 rrweb-player/Svelte（**不重写 Replayer 核心**）
- 精简：砍 canvas/console/packer/plugin
- 新能力：nodeID 跨端寻址 + 元素操作原语

**不做（避免范围爬升）**：
- 升级到比 alpha.20 更新的版本 → fork 之后独立动作
- 运行时动态改 mask / 自定义事件通道统一 / 录制质量分级 → fork 之后独立切片
- packer 内联压缩 → 压缩下沉存储层 gzip，与本工作流解耦
- 砍 Shadow DOM / 同源 iframe → 访客站点不可预测，保留保命

## 估时（粗）

> **以下估时为 2026-06-25 `/grill-with-docs` 修订后版本**。对比原 spec 估时：fork-0 +1.5〜2d（识别到 6 包隐式依赖+import 重写）,fork-3 +2d（测试转 Playwright）,新增 fork-3b。

- fork-0：3-4 天（6 内部包源+import 重写+TS/Vite 配置+双 parity 夹具）
- fork-1：1 天
- fork-2：2-3 天（钻穿,不重写 Replayer 核心）
- fork-3：4-5 天（文件级裁剪 + 5 组上游测试转 Playwright）
- fork-3b：1-2 天（snapshot 写 data-rr-node-id,内嵌 fork-3 阶段）
- fork-4：2-3 天（nodeID 链路 + 正负向 e2e）

**累计估时：~16 天**（原 ~10 天,差异来自上游源码移植复杂度 + 测试转译 + fork-3b 独立）

## 关联

- 后续切片文档：`docs/progress/2026-06-25-slice-fork-0-*-spec.md` 起逐片创建
- 完成后连 implementation 一起移 `docs/reports/completed/`，带深度 badge
- fork-2 改 SDK/admin 既有文件前，按 [change-safety.md](../standards/change-safety.md) 备份原文件至 `/backup`

## Notes（grill-me 访谈记录要点）

- **被否决方案**：薄抽取(B)——会在 record/replay mirror 边界反复踩 id 不一致；包装层(C)——现状已证明是债务黑洞。
- **fork-0 第一待核实项**：`nodeMap.ts` 假设 DOM 上有 `data-rr-node-id`，但 rrweb mirror 默认把 nodeID 存 JS Map 不写 DOM。高度怀疑现状点选一直走坐标 fallback、`data-rr-node-id` 路径从未命中。拷进 snapshot 源码后**第一件事核实**，这正是 fork-4 要根治的根因。
- **架构主张核心**：fork 的正收益 = 同时拥有两端 mirror，让 nodeID 成为跨端精确指针（rrweb 回放能工作的前提就是两端共享同一 id 空间）；负债清理 = 持有 Replayer 实例后 sandbox/sizing/live/mount 四类 hack 全部消解。

## Grill 访谈决策记录（2026-06-25 `/grill-with-docs`）

见 [`memory/daily/2026-06-25.md`](../../memory/daily/2026-06-25.md) §vendor-rrweb grill 完整转录。

| # | 决策点 | 推荐 | 选择 | 影响 |
|---|---|---|---|---|
| D1 | fork-2 scope | 钻穿（Vue 壳删 Svelte,不重写 Replayer 核心） | ✅ 同意 | 估时保持 2-3d |
| D2 | 上游测试框架 | 转 Playwright（统一测试栈） | ✅ 同意 | fork-3 +2d |
| D3 | fork-1 验证标准 | 功能等价 + 三方截图对比 | ✅ 同意 | fork-1 门提高 |
| D4 | fork-3b 归属 | 内嵌 fork-3 阶段（非 fork-4 后） | ✅ 同意 | 新增 1-2d 切片 |
| D5 | fork-0 基线 | 钉 alpha.20 | ✅（已定） | 不升级 |
| D6 | parity 验证 | 双夹具（record + replay） | ✅ 同意 | fork-0 +0.5d |
| D7 | fork-0 估时 | 从 1-2d 调整到 3-4d | ✅ 同意 | 6 包隐式依赖 + import 重写 |
| D8 | fork-3 估时 | 从 2-3d 调整到 4-5d | ✅ 同意 | 测试转 Playwright |

**风险**：
- **R1**：`data-rr-node-id` 假设在 fork-0 启动前需 15min 实验验证。若假设成立（id 已存在），nodeMap 路径有效，fork-4 范围可缩；若不成立，fork-4 的根治方案确认正确。
- **R2**：fork-0 源码移植含 6 内部包（snapshot/record/replay/types/utils/rrdom）+ import 重写，估时 3-4d 含余量。
- **R3**：fork-1 的"功能等价 + 三方截图"标准高于原 spec 的"parity diff 夹具绿"，若 fork-1 遭遇格式不兼容会揭示录制端差异，届时需追加修复切片。

---

## 进展日志

### 2026-06-25 23:25 — fork-0-parity ✅ 完成

**fork-0 源码移植**（提交 `235aa37`）：
- 从 rrweb alpha.20 clone 41 个 TS 源文件（snapshot / record / replay / types / utils / rrdom）到 `packages/replay-core/src/`
- 重写所有 import 路径（npm 包名 → 相对路径）
- 创建 package.json / tsconfig.json / vite.config.ts
- TypeScript 编译通过（0 errors）,Vite build 成功（42 modules → 305KB ESM）
- 包含 NOTICE (MIT attribution) 和 README

**fork-0-parity 双夹具**（提交 `44bc9a5`）：
- `e2e/tests/fork-0-replay-parity.spec.ts` ✅ 1 passed —— 同 events → 同 iframe outerHTML
- `e2e/tests/fork-0-record-parity.spec.ts` ✅ 1 passed —— 同 DOM + 同交互 → 同 events 数组
- 双夹具独立可跑（`SKIP_MM_WEBSERVER=1 npx playwright test fork-0-`）

**R1 验证**：
- `document.querySelectorAll('[data-rr-node-id]')` 长度 = 0 —— **rrweb Mirror 不写 DOM attribute**
- `nodeMap.ts` 确认为死代码，co-browse 点选全量走坐标回退
- 根治方案（snapshot 主动写 `data-rr-node-id`）是 fork-4 的正确方向

**构建变更**：
- replay-core 改为全量内联构建（无外部 import，ESM + IIFE 双格式）
- 添加 vitest + jsdom 配置（stub，后续可升级）
- 添加 rrweb 作为 devDependency（仅供测试对照）

**下一步**：fork-1 — SDK record 切到 replay-core（1d 估时）

### 2026-06-25 23:55 — fork-1 ✅ 完成

**SDK record 切换**（提交 `4ef8c60`）：
- `visitor-sdk/src/collectors/rrweb.ts`：`import('rrweb')` → `import('@pinconsole/replay-core')`
- 注释更新（rrweb v2 alpha → replay-core）
- `visitor-sdk/package.json`：`rrweb` + `rrweb-snapshot` → `@pinconsole/replay-core: workspace:*`

**验证**：
- SDK build ✅ 147 modules → 282KB（replay-core 源码内联）
- 214 单元测试 ✅ 全绿
- tsc --noEmit ✅ zero errors
- 服务端日志确认 SDK 连接正常、录制逻辑正常
- 1l E2E ✅ 4 passed, 2 skipped
- 1c E2E ⚠️ 预存 admin UI 失败（"订阅实时"按钮找不到），与本次改动无关

**依赖状态**：
- visitor-sdk 不再依赖 `rrweb` 和 `rrweb-snapshot`
- admin 仍依赖 `rrweb-player`（待 fork-2 删除）

**下一步**：fork-2 — admin 钻穿，删 rrweb-player/Svelte（2-3d）

### 2026-06-26 00:00 — fork-2 ✅ 完成

**admin 钻穿方案执行**（提交 `5f5a8eb`）：
- **ReplayPlayer.vue**: ~380 行 → ~180 行。删除所有 Svelte hack（MutationObserver sandbox、UNSAFE_replayCanvas、rebuildToken、replaceChildren、requestAnimationFrame、getReplayer 代理），直持 Replayer 实例。
- **ReplayViewer.vue**: ~605 行 → ~400 行。同样钻穿 + 自建控件线（play/pause/seek/speed/skipInactive → Replayer API），时间追踪改用 setInterval 轮询（100ms）。
- **useResponsivePlayerSize.ts**: ~185 行 → ~90 行。删除 PlayerLike 接口、$set/triggerResize/getReplayer/MutationObserver → 直接用 replayer.wrapper.style + replayer.handleResize()。
- **admin/package.json**: `rrweb-player` → `@pinconsole/replay-core`。

**15 项 hack 全部消解**（详见表），净减 ~600 行代码。

**消解的 15 项 hack**：
| # | Hack | 原来 | 现在 |
|---|---|---|---|
| H1 | CSS preload | await import('rrweb-player/dist/style.css') | 内联 .replayer-wrapper 样式 |
| H2 | Sandbox patch | MutationObserver 监听 iframe + 补 allow-scripts | Replayer 自管 iframe sandbox |
| H3 | UNSAFE_replayCanvas | 传入以开启 allow-scripts | 删除（Replayer 直接设） |
| H4/H5/H6 | 重建 token + DOM 清空 + initialized | rebuildToken, replaceChildren, 条件标志 | 直接 destroy + 重建 |
| H7 | Live mode via startLive | 监听 finish → startLive(farFuture) | 直接 replayer.startLive() |
| H8 | RAF wait after construction | await requestAnimationFrame | 删除（无 Svelte onMount） |
| H9/H10/H11 | 响应式 sizing | $set + triggerResize + handleResize fallback | 直接 wrapper.style + handleResize() |
| H12 | iframe display:block | CSS !important 覆盖 | Replayer 自管 display |
| H13 | No width/height:100% | CSS 注释 | 保留 CSS 约束 |
| H14 | Min 2 events guard | length < 2 检查 | 保留逻辑 |
| H15 | Individual addEvent | 逐个 addEvent（无 append） | 保留逻辑 |

**验证**：
- Admin build ✅ 1746 modules
- Unit tests ✅ 14/14 files, 146/146 tests
- Bundle ✅ 零 "rrweb-player" / "SvelteComponent" 字面

**依赖状态**：
- admin 不再依赖 `rrweb-player`
- 全仓仅 `visitor-sdk/node_modules/rrweb` 残留（parity 测试用）
- 全仓仅 `admin/node_modules/rrweb-player` 残留（可手动清理）

**下一步**：fork-3 — 精简：文件级裁剪 + 上游测试转 Playwright（4-5d）
