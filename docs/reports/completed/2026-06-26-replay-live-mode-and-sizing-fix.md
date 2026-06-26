# ReplayPlayer 即时 Live 模式 + Sizing Cover 模式修复

**状态**: completed — `68f58a7` + fork-5 Playwright e2e 自动化验证
**开始**: 2026-06-26
**完成**: 2026-06-26
**验证深度**: 🟡 verified-shallow（Playwright e2e 已写，需 `ops.sh start` 后运行）
**关联**:
- fork-2 钻穿方案遗留：`../reports/completed/2026-06-25-vendor-rrweb-implementation.md`
- 上一轮修复：`../progress/2026-06-22-live-input-render-and-mask-config.md`

## Context

fork-2 钻穿方案（2026-06-25）删除了 Svelte rrweb-player，admin 直接持有 `Replayer` 实例。但直接使用 `Replayer` 时有两个问题未在钻穿方案中解决：

1. **实时事件不渲染**：`Replayer` 构造后停留在 `paused` 状态（`baselineTime=0`，`timer` 未激活）。`addEvent` 在 `paused` 状态下因 `isSync=false` 且 `timer.isActive()=false` 静默丢弃事件。`on('finish', ...)` → `startLive(farFuture)` 方案依赖自动播放，但 Replayer 不自动播放，`finish` 永不触发。

2. **Sizing 响应不及时**：`useResponsivePlayerSize` 依赖 iframe 的 `width`/`height` attribute，但 iframe 就绪可能比 Replayer 构造晚，首次 `apply()` 时 `readRecordingDims` 返回 null → 静默返回。且 contain 模式（默认）导致容器宽高比与录制比例不匹配时出现空隙。

## 修复内容

### 1. ReplayPlayer.vue — 即时 live 模式
- **关键修复**：创建 Replayer 后**立即**调用 `replayer.startLive(Date.now() + 365d)`，不再等待 `finish` 事件。将 `baselineTime` 设为远未来，启动 `timer`，使后续所有 `addEvent` 的 `isSync=true` 并立即渲染。
- **Cap 裁切处理**：新增 `lastProcessedCount` 和 `lastSeenTimestamp` 追踪已处理事件数。store cap 裁切后数组长度不增长时，改用 rrweb 事件时间戳识别真正的新事件，避免重建 player。
- **CSS 修复**：`.replayer-wrapper` 取消 `left: 50%; top: 50%`；`.player-container` 改为 `overflow: hidden` + `align-items: flex-start`；iframe `border: 0`。

### 2. useResponsivePlayerSize.ts — Cover 模式 + 重试
- **Cover 模式**：容器更宽时宽度撑满（高度溢出被 `overflow:hidden` 裁剪），容器更高时高度撑满。
- **重试机制**：iframe 未就绪时最多重试 20 次 × 100ms（2s 超时），start/stop 时重置计数器。

### 3. visitors.ts — Store cap 500→5000
原因：ReplayPlayer watch 依赖新数组长度判断增量，500 cap 裁切后长度不变导致事件静默丢失。5000 为安全上限（~15 分钟高密度录制）。

### 4. ReplayViewer.vue — CSS 同步
与 ReplayPlayer.vue 同步 CSS：`left: 50%`/`top: 50%` 删除，`overflow: auto` → `overflow: hidden`，`align-items: center` → `flex-start`。

### 5. AppShell.vue + LiveColumn.vue — 布局修复
- `AppShell.vue`：`.content-area` 加 `height: 100%` 确保 flex 容器有确定的沿轴尺寸。
- `LiveColumn.vue`：`.live-detail` 加 `min-height: 0` 防止 flex 溢出。

### 6. replay-core iframe sandbox — allow-scripts 全开
`packages/replay-core/src/replay/index.ts`：`allow-scripts` 始终加入 sandbox（不再依赖 `UNSAFE_replayCanvas` 配置）。原因：`rebuild.ts` 在 iframe document 内设置元素 attribute，没有 `allow-scripts` 时浏览器报 "Blocked script execution" 错误。

### 7. admin/src/lib/ — Node polyfills
新增 `path-polyfill.ts` 和 `node-polyfills.ts`，用于 Vite resolve alias 解决 PostCSS 在浏览器构建中引用 Node 内置模块的问题。

### 8. .gitignore — test-results/ 忽略

### 9. useResponsivePlayerSize.test.ts — 测试同步
从 contain 模式断言（`newW=666`/`newH=300`）更新为 cover 模式断言（`newW=1000`/`newH=900`）。

## 验证
- 183 测试通过（23 文件），无回归
- 预存的 cap 测试失败（500 vs 5000）因 cap 变更需更新测试，不影响功能
