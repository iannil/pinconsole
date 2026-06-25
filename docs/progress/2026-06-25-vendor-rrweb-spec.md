# 切片 vendor-rrweb — Spec（总纲）

**切片编号**：vendor-rrweb（fork-0 ~ fork-4 工作流总纲）
**类型**：重构 / 新功能（横切录制端 + 回放端）
**创建时间**：2026-06-25
**状态**：approved
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
| **fork-0** | clone alpha.20 源码 → 拷入 `packages/replay-core`（snapshot/record/replay/types）→ 能编译 + parity diff 夹具就位。**纯新增文件,不动 SDK/admin**。核实 `data-rr-node-id` 假设 | parity diff 夹具绿 | 是（并存对照） |
| **fork-1** | SDK record 切到 replay-core | 1c + 1l 绿 | admin 侧仍依赖 |
| **fork-2** | admin 回放切 replay-core + 原生 Vue 重写 player + **删 rrweb/rrweb-snapshot/rrweb-player 三依赖 + Svelte** | 1c + 1d 绿 | **否（依赖全删）** |
| **fork-3** | 精简：砍 canvas/console/packer/plugin 管线 + 移植上游保留功能 jest 测试 | 移植 jest 全绿 + 1c/1d/1l 不回归 | 否 |
| **fork-4** | 新能力：nodeID 跨端寻址 + 删 nodeMap + 删坐标 fallback + ②元素操作原语 | 1e/1f + 新增正负向 e2e 绿 | 否 |

**顺序理由**：fork-0 纯新增零风险；fork-1/2 等价替换、每步有 e2e 兜底可回滚；**删三依赖卡在 fork-2 末**（录制+回放都切完才删，留对照到最后）；精简与新能力放在 replay-core 独占之后，雕刻空间最大。

## Acceptance（总纲级，可验证）

- [ ] `packages/replay-core` 编译通过，`main` 指向 `src/index.ts`，`type: module`，含 `NOTICE`（fork 自 rrweb-io/rrweb @ alpha.20 + commit hash）+ 保留 MIT 文件头
- [ ] fork-2 完成后 `rg "rrweb" visitor-sdk/package.json admin/package.json` 无命中（三依赖删净）；admin 无 Svelte 运行时
- [ ] 全程既有 e2e（1c/1d/1e/1f/1l）保持绿，无回归
- [ ] fork-3 后移植的上游 jest（snapshot/replayer/observer/shadow DOM/iframe/mask）全绿
- [ ] fork-4 后 `nodeMap.ts` 与坐标 fallback 删除；新增正向（按 nodeID 精确点选/代填命中）+ 负向（陈旧/不存在 nodeID → 优雅 no-op 不崩）e2e 绿

## 深度目标

按 [verification-depth.md](../standards/verification-depth.md) R2 rubric：

- 🟢 verified-deep：fork-1/2/4 均要求既有或新增 Playwright e2e 覆盖真浏览器交互；fork-4 作为 cobrowse 边界类切片**必须含负向测试**（陈旧 nodeID → no-op）
- 🟡 verified-shallow：fork-3 精简以移植 jest 单测保真为主，e2e 仅验"不回归"，不新增正向 e2e
- 🔴 不接受：任何切片不得以 `if (!x.length) return;` 静默 pass

## 验证四防线

1. **既有 e2e 当验收门**：1c/1d/1e/1f/1l 真浏览器跑录制→回放→点选→脱敏。
2. **parity diff 夹具锁阶段一等价**：同一事件流喂旧 Replayer vs 新 replay-core Replayer，diff iframe 最终 outerHTML，删旧依赖前做、删完即弃。
3. **移植上游 jest 锁精简保真**：覆盖简单访客页打不到的 shadow DOM/iframe/CSS/adopted stylesheet edge case。
4. **新 nodeID 配正负向 e2e**。

## 范围边界

**本工作流做**：
- 硬分叉 rrweb（snapshot/record/replay）TS 源码进 `packages/replay-core`
- admin 原生 Vue 重写 player，删除 rrweb-player/Svelte
- 精简：砍 canvas/console/packer/plugin
- 新能力：nodeID 跨端寻址 + 元素操作原语

**不做（避免范围爬升）**：
- 升级到比 alpha.20 更新的版本 → fork 之后独立动作
- 运行时动态改 mask / 自定义事件通道统一 / 录制质量分级 → fork 之后独立切片
- packer 内联压缩 → 压缩下沉存储层 gzip，与本工作流解耦
- 砍 Shadow DOM / 同源 iframe → 访客站点不可预测，保留保命

## 估时（粗）

- fork-0：1-2 天（拷源码 + 编译打通 + parity 夹具）
- fork-1：1 天
- fork-2：2-3 天（Vue 重写 player 是重头）
- fork-3：2-3 天（精简 + 移植 jest）
- fork-4：2-3 天（nodeID 链路 + 正负向 e2e）

## 关联

- 后续切片文档：`docs/progress/2026-06-25-slice-fork-0-*-spec.md` 起逐片创建
- 完成后连 implementation 一起移 `docs/reports/completed/`，带深度 badge
- fork-2 改 SDK/admin 既有文件前，按 [change-safety.md](../standards/change-safety.md) 备份原文件至 `/backup`

## Notes（grill-me 访谈记录要点）

- **被否决方案**：薄抽取(B)——会在 record/replay mirror 边界反复踩 id 不一致；包装层(C)——现状已证明是债务黑洞。
- **fork-0 第一待核实项**：`nodeMap.ts` 假设 DOM 上有 `data-rr-node-id`，但 rrweb mirror 默认把 nodeID 存 JS Map 不写 DOM。高度怀疑现状点选一直走坐标 fallback、`data-rr-node-id` 路径从未命中。拷进 snapshot 源码后**第一件事核实**，这正是 fork-4 要根治的根因。
- **架构主张核心**：fork 的正收益 = 同时拥有两端 mirror，让 nodeID 成为跨端精确指针（rrweb 回放能工作的前提就是两端共享同一 id 空间）；负债清理 = 持有 Replayer 实例后 sandbox/sizing/live/mount 四类 hack 全部消解。
