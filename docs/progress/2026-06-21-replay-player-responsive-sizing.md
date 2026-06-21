# Replay Player 响应式 sizing 修复

**状态**:in_progress(待用户在浏览器验证后移到 `reports/completed/`)
**开始**:2026-06-21
**完成**:(待浏览器验证)
**关联**:实时回放切片 `docs/reports/completed/2026-06-17-slice-1c-*.md`,历史回放切片 `docs/reports/completed/2026-06-17-slice-1d-*.md`

## Context

用户报告 admin 实时 dashboard 的 replay 显示尺寸和实际录制视口尺寸不一致。提供的 HTML 显示:`.rr-player` 外框是 `1024×576`(16:9),iframe(真实录制视口)是 `1579×904`(≈1.746:1),按高度缩放后 iframe 宽度只有 1006.5px,左右各约 9px 空隙,且外框比例与录制视口比例不一致。

## 根因分析(两次定位)

### 第一次(错诊)

认为是 `ReplayPlayer.vue` 没给 rrweb-player v2 alpha.18 传 `width`/`height` props,触发其硬编码默认值 `1024×576`(`node_modules/.../rrweb-player.js:14615-14616`)。写了个从首个 rrweb Meta event (type=4) 提取 `data.width`/`data.height` 的 composable。

**结果**:修复未生效。用户再次提供的 HTML 显示 `.rr-player` 还是 `1024×576`。

### 第二次(真因)

`server/internal/api/ws.go:511-518` 配合 `extractFullSnapshotEnvelope` (`ws.go:557-580`) —— **服务端 subscribe 时只缓存并发送 full snapshot (rrweb type=2),不缓存 meta event (type=4)**。meta event 只在访客首次连接时由 SDK 上报一次,服务端没保留。

admin 实时订阅时收到的事件流 = [full snapshot, ...incrementals],**没有 meta event**。所以 `extractRecordingDimensions` 找不到 type=4,返回 null,`apply()` 提前 return,player 保留默认 1024×576。

历史回放页 `ReplayViewer.vue` 通过 REST API 获取完整事件流(含 meta),没这个问题 — 但仍然有默认 1024×576 不响应窗口尺寸的问题。

## 最终修复方案

**核心思路**:改用 rrweb-player 创建的 iframe 元素的 `width`/`height` attribute 作为录制视口尺寸来源,不依赖 meta event 是否被传到 admin。

`rrweb-player` 内部用 meta event 的 width/height 设置 iframe attribute(`node_modules/.../rrweb-player.js:11014-11015`)。即使 admin 没收到 meta event,iframe 仍带有正确的录制视口尺寸 — 这是最可靠的信号源,因为它是 rrweb-player 内部实际使用的同一份数据。

## Changes

- [x] `admin/src/composables/useResponsivePlayerSize.ts`:从 container 内的 iframe 元素读 `width`/`height` attribute 得到录制视口。`apply()` 内联读 iframe,无需外部传事件。两个 observer:
  - `ResizeObserver`:监听容器尺寸变化
  - `MutationObserver`:监听 iframe 插入 + iframe width/height attribute 变化(visitor viewport 改变),用 `shouldReapply` 过滤掉普通 DOM mutation(避免 replay 过程中 iframe 内部 DOM 变化频繁触发)
- [x] `admin/src/components/ReplayPlayer.vue`(实时回放)接入 composable:player 创建后 `startResponsiveSizing()`,player 销毁/sessionId 切换/组件卸载时 `stopResponsiveSizing()`。`RRWebPlayerInstance` 类型补 `$set`。
- [x] `admin/src/views/ReplayViewer.vue`(历史回放)接入 composable:同上。`RRWebPlayerInstance` 接口补 `$set`。
- [x] 两处 `.player-container` CSS 改为 `display: flex; align-items: center; justify-content: center;` 让 `.rr-player` 外框居中。
- [x] `admin/tests/useResponsivePlayerSize.test.ts` 12 个单测:start 立即 apply、容器更宽/更窄、iframe 后插入触发、iframe width/height 变化触发、容器 resize 触发、player null 守卫、0x0 跳过、iframe 缺尺寸跳过、多 iframe 取首个有效、stop 释放、re-start 清旧 observer、非 iframe mutation 过滤。

## Status

代码完成。`pnpm exec vue-tsc --noEmit` 通过,`pnpm test` 全套 152 通过(140 原有 + 12 新增),`pnpm build` production 构建通过。

**验证深度**:🟡 verified-shallow
- composable 单测覆盖核心逻辑(数学、observer 生命周期、过滤、回退)
- 类型 + 构建 + 全套测试通过
- 但**未做浏览器端 e2e 验证**

剩余工作:
- 用户在浏览器实测:启动 `pnpm dev` → 打开 admin dashboard → 选在线 visitor 触发 replay → 观察外框尺寸是否跟随真实录制视口比例(而非 1024×576)。调整窗口尺寸时外框应跟随。
- 历史回放路径:`/admin/replay/:session_id` 选历史会话,同上观察。

## Next

用户浏览器验证通过后:
1. 把本文件移到 `docs/reports/completed/`
2. 状态字段改为 `completed`,补完成日期
3. 验证深度按实际结果调整

## Blockers

无。

## Notes

### 关键源码位置(便于后续追溯)

- `node_modules/.pnpm/rrweb-player@2.0.0-alpha.18/.../dist/rrweb-player.js:14615-14616` — 默认 `width=1024`、`height=576` props
- `:11014-11015` — rrweb-player 用 meta event 的 `data.width`/`data.height` 设 iframe attribute
- `:14771-14772` + `:14783-14797` — `$set({width, height})` 触发 reactive update 重设 `.rr-player` inline style
- `server/internal/api/ws.go:557-580` — `extractFullSnapshotEnvelope` 只识别 type=2(full snapshot),不识别 type=4(meta)
- `server/internal/api/ws.go:511-518` — subscribe 时下发缓存的 full snapshot envelope

### 设计决策

- **为什么用 iframe width/height 而非 meta event**:meta event 在实时订阅链路上不被服务端缓存/下发,只有历史 API 才能拿到完整事件流。iframe attribute 是 rrweb-player 自己提取并使用的同一份数据,所有路径(实时/历史)都可靠。
- **为什么用 MutationObserver 而非只 ResizeObserver**:iframe 由 rrweb-player 在 constructor 后异步插入 DOM(虽然在大多数情况下同步,但 visitor viewport 中途变化时 iframe attribute 会更新),需要 DOM observer 兜底。
- **为什么 shouldReapply 过滤**:rrweb Replayer 在回放过程中持续修改 iframe 内部 DOM(contentDocument 重建),如果每次 mutation 都触发 apply 会高频调 `$set` 引起 Svelte re-render。过滤只关心 iframe 本身的插入/属性变化。

### 已知边界

- **mirror iframe 干扰**:rrweb 可能创建多个 iframe(mirror + renderer)。composable 取第一个有合法 width/height 的。若 mirror iframe 也有合法尺寸但与 renderer 不一致,会取错。当前测试覆盖这个 case 但实际项目中是否发生未验证。
- **录制中途 viewport 变化**:visitor 改 viewport 后,rrweb 会发新的 FullSnapshot 或 Resize event,服务端缓存刷新,iframe attribute 更新,MutationObserver 触发 re-apply。理论上自洽,实际未做 e2e 验证。

### 未来可能的优化方向(post-v1)

- 服务端 `extractFullSnapshotEnvelope` 改为同时缓存 meta event(`ws.go:557`),subscribe 时一起下发。这样 admin 收到的初始事件流是 [meta, full snapshot],更符合 rrweb 协议期望,可能解决一些其他依赖 meta event 的边界 case。
- 但 client-side 修复已经解决问题,服务端改动风险更大,暂不动。
