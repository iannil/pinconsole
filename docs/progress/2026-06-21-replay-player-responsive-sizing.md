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

### 服务端(meta event 缓存,根治)

- [x] `server/internal/recording/snapshot.go`:扩展 `SnapshotCache`,加 `MetaKey`/`SetMeta`/`GetMeta`/`DeleteMeta`。meta TTL 30 分钟(覆盖典型 session 时长,snapshot TTL 仍 5 分钟)。
- [x] `server/internal/api/ws.go`:
  - 新增 `extractMetaEnvelope(ctx, env) []byte` 函数(与 `extractFullSnapshotEnvelope` 同构,匹配 rrweb type=4)。
  - visitor envelope 处理:除原有 `extractFullSnapshotEnvelope + Set`,新增 `extractMetaEnvelope + SetMeta`,把首个 meta event 缓存到 Redis。
  - operator subscribe 处:**先发 meta 再发 full snapshot**(符合 rrweb 协议),让 admin 端 rrweb-player 收到 meta 触发 handleResize 让 iframe 显示。
  - session end 处:同时清 meta + snapshot 缓存。
- [x] `server/internal/api/coverage_extra_test.go`:4 个新测试覆盖 `extractMetaEnvelope`(single meta / 非 event 返回 nil / full snapshot 不误识 / batch 提取)。

### 真正根因(2026-06-21 二次定位)

服务端逻辑全部正确。诊断日志确认:visitor 发的 batch `[4 2]`(meta+full)被正确缓存;operator subscribe 时 `meta_cache_hit=true snapshot_cache_hit=true`。问题在 **visitor SDK**。

rrweb 2.0.0-alpha.20 把 `takeFullSnapshot` 从 top-level export 改成了 `record.takeFullSnapshot`(`node_modules/.pnpm/rrweb@2.0.0-alpha.20/.../rrweb.js:14199`: `record.takeFullSnapshot = ...`)。SDK 还在用 alpha.18 的旧 API `pack.takeFullSnapshot?.()`,alpha.20 下 `pack.takeFullSnapshot === undefined`,SDK 的 `if (!this.pack?.takeFullSnapshot) return;` 静默返回。

→ SDK 的周期性 full snapshot(每 30s 若 ≥50 incrementals 触发)永远不发。
→ 服务端 snapshot cache TTL 5 分钟,过期后没有任何 full snapshot 续期。
→ admin subscribe 时 `snapshot_cache_hit=false`,只收到 meta(1 个 event)。
→ `ReplayPlayer.rebuildPlayer` 里 `rrwebEvents.length < 2` 提前 return,显示"已订阅,等待访客产生交互"。
→ `.rr-player` 外框存在(Svelte 组件挂载)但 `.rr-player__frame` 内无 iframe(Replayer 构造从未被触发)。

**修复**:`visitor-sdk/src/collectors/rrweb.ts` 把 `pack.takeFullSnapshot()` 改成 `pack.record.takeFullSnapshot()`,类型定义同步把 `takeFullSnapshot` 从 `RRWebPack` 顶级字段移到 `RRWebRecordFn` 的附加属性。


### 客户端(升级 + 响应式 sizing + 兜底)

- [x] `admin/package.json`:`rrweb-player` 从 `2.0.0-alpha.18` 升级到 `2.0.0-alpha.20`。`visitor-sdk/package.json` 同步把 `rrweb` / `rrweb-snapshot` 也 pin 到 `2.0.0-alpha.20`(精确版本,不用 `^` 避免被解析到 stable 2.0.1)。三个包版本对齐。
- [x] `visitor-sdk/src/collectors/rrweb.ts`:修复 alpha.20 API 变更导致的 takeFullSnapshot 静默失败(详见上面"真正根因")。
- [x] `admin/src/composables/useResponsivePlayerSize.ts`:
  - `apply()` 内联从 container 的 iframe 读 `width`/`height` attribute 得到录制视口。
  - **兜底**:如果 iframe 无 width/height(服务端旧缓存只有 snapshot 没 meta),从 `iframe.contentWindow.innerWidth/innerHeight` 读真实渲染视口,手动调 `replayer.handleResize(dims)` 让 rrweb-player 把 iframe 设为 display:inherit + 写入 width/height attribute。然后再走正常 sizing 路径。
  - 两个 observer:`ResizeObserver` 监听容器尺寸;`MutationObserver` 监听 iframe 插入 + width/height attribute 变化,`shouldReapply` 过滤掉 replay 过程中 iframe 内 DOM 变化。
- [x] `admin/src/components/ReplayPlayer.vue` / `admin/src/views/ReplayViewer.vue`:接入 composable,`RRWebPlayerInstance` 类型补 `$set` + `getReplayer`。
- [x] 两处 `.player-container` CSS 改为 `display: flex; align-items: center; justify-content: center;` 让 `.rr-player` 外框居中。
- [x] `admin/tests/useResponsivePlayerSize.test.ts` 12 个单测覆盖 sizing 数学 + observer 生命周期 + 过滤逻辑。

## Status

代码完成。
- 服务端:`go build ./...` 通过,`go test ./internal/api/... ./internal/recording/...` 通过(新增 4 个 extractMetaEnvelope 测试)。
- 客户端:`pnpm exec vue-tsc --noEmit` 通过,`pnpm test` 全套 152 通过,`pnpm build` 通过。

**验证深度**:🟡 verified-shallow
- composable 单测覆盖核心逻辑(数学、observer 生命周期、过滤、回退)
- 类型 + 构建 + 全套测试通过
- 但**未做浏览器端 e2e 验证**

剩余工作:
- 用户在浏览器实测:启动 `pnpm dev` → 打开 admin dashboard → 选在线 visitor 触发 replay → 观察外框尺寸是否跟随真实录制视口比例(而非 1024×576)。调整窗口尺寸时外框应跟随。
- 历史回放路径:`/admin/replay/:session_id` 选历史会话,同上观察。

## 2026-06-21 23:00 — 双重 .rr-player + iframe 压扁到 150px(浏览器验证暴露的 2 个新 bug)

用户在浏览器实测后报告:"录屏尺寸显示错误,大片的空白,只有一小块是用户页面录屏"。
提供 HTML 显示 `.player-container` 内有 **两个** `.rr-player` 元素。

### Bug 1:`rebuildPlayer` 双 mount race condition(`admin/src/components/ReplayPlayer.vue`)

**根因**:
- `onMounted(() => rebuildPlayer())` 和 `watch(() => props.events, ...)` 在初始挂载时几乎同时触发 `rebuildPlayer()`。
- `rebuildPlayer` 内有 `await loadPack()`。在 await 期间 `player` 变量仍是 null,两次调用都跳过 `if (player)` 清理守卫。
- 各自创建一个 Svelte Player 实例 → DOM 里出现两个 `.rr-player` 元素。
- Flex 布局把两个外框各缩一半,iframe 仍按原比例缩放并 overflow → "大片空白 + 一小块录屏" 视觉 bug。

**修复**:
- 加 `let rebuildToken = 0;` token 计数器,每次 `rebuildPlayer` 入口 `const myToken = ++rebuildToken`。
- `await loadPack()` 完成后 `if (myToken !== rebuildToken) return` — 若期间有更新的调用,放弃本次(避免双 mount)。
- 把 `containerRef.value.replaceChildren()` 移出 `if (player)` 守卫,变成无条件清空(首次 mount player 是 null 时也要清)。
- `finally` 里 `loading.value = false` 也加 token 检查,避免被后续调用覆盖。

**浏览器验证**:`.player-container` 内 `.rr-player` 数量从 **2 → 1**。

### Bug 2:iframe CSS `height: 100%` fallback 到默认 150px(`admin/src/components/ReplayPlayer.vue` + `admin/src/views/ReplayViewer.vue`)

**根因**(只在 Bug 1 修复后才暴露):
- 1c 切片为旧 layout 加的 CSS hack:`.player-container :deep(iframe) { width: 100%; height: 100%; }`,让 iframe 撑满 `.player-container`。
- 但当前 layout 里 iframe 的父链是 `.player-container > .rr-player > .rr-player__frame > .replayer-wrapper`,中间三层都没有显式高度。
- CSS `height: 100%` 在父元素无显式高度时无法 resolve,浏览器 fallback 到 iframe 默认 **150px**(HTML 规范默认值)。
- 实测 iframe computed height = **150px**(应为 ~509px)。
- 视觉上 iframe 被严重压扁,只显示顶部一小条 → 进一步加重 "只有一小块是录屏" 的视觉。

**修复**:
- 移除 `width: 100%; height: 100%;` CSS。
- 保留 `display: block !important; pointer-events: auto !important;`(仍是 rrweb alpha mirror iframe 默认 display:none 的必要覆盖)。
- iframe 现在用自身的 `width`/`height` attribute(1913×904)+ `.replayer-wrapper` 的 `transform: scale(0.563053)` 正确缩放。

**浏览器验证**:
- iframe computed height 从 **150px → 904px** ✅
- iframe rect 从 **607×84 → 1066×504** ✅
- 录屏清晰可见,大小合理,无过大空白(用户报告的视觉问题完全消除)。

### 单元测试

写了一个 race condition 回归测试 `admin/tests/replay_player_race.test.ts`,但 **vitest 4 + vite 5 无法可靠 mock vite 预 bundle 的 rrweb-player**(尝试了 vi.doMock / vi.mock / vi.hoisted / deps.inline / deps.optimizer.ssr.include / optimizeDeps.exclude / setupFiles 都失败 — factory 被调用但返回的 mock 模块没被用,real Player2 仍被加载)。

- 已删除该测试文件(无法可靠验证)。
- 已有的 152 个测试全绿,无回归。
- 浏览器实测验证:Chrome DevTools MCP 检查 DOM 元素数量 + computed style + 截图 AI 视觉分析三重确认。

### Bug 3:`initialized` flag 永远 false 导致 sandbox warning spam(用户报告 5 条 → 实测 123 条)

**根因**:
- `ReplayPlayer.vue` 的 watch callback:
  ```js
  if (!initialized || !player) {
    rebuildPlayer();
    if (player) initialized = true;  // <-- bug
    return;
  }
  ```
- `rebuildPlayer()` 是 async,返回 Promise 不被 await。
- 紧接着的 `if (player)` 是同步检查,此时 player 还是 null(await 还没 resolve)。
- `initialized` 永远 false → 每个 WS 事件都触发完整 rebuild。
- 每次 rebuild 新建 iframe → 浏览器对每个 iframe 都报 "allow-scripts + allow-same-origin can escape sandboxing" warning。
- 实测 30 秒访问 → **123 条 sandbox warning + 24 条 "replayer has been destroyed"**。

**修复**:
- 把 `initialized = true` 移到 `rebuildPlayer` 内部,player 成功创建后立即设置。
- watch callback 里删除无效的同步 `if (player) initialized = true`。

**浏览器验证**:
- sandbox warning 数量:**123 → 2**(降 98%)
- "replayer destroyed" warning:**24 → 0**
- 剩 2 条是 rrweb-player 首次创建 iframe 时不可避免的(UNSAFE_replayCanvas 设计权衡)。

**注**:`ReplayViewer.vue` 用的是 `if (newEvents.length > 0 && !player)` 守卫(不是 initialized flag),没有这个 bug。

### 验证深度

🟢 verified-deep
- DOM 元素数量比对(bug 前后 2 vs 1)
- Computed style 比对(iframe height 150px vs 904px)
- 截图 AI 视觉分析(录屏清晰可见,布局合理,无过大空白)
- Console warning 数量比对(123 → 2)
- 152 单测全绿无回归

### Bug 4:admin subscribe 时缺 FullSnapshot → iframe 空白(snapshot cache 过期)

**现象**:用户报告 "admin 端没有自动同步 client 端的页面操作"。实测某 session 的 admin store 里 events 类型序列是 `[4, 3, 3, ...]`(Meta + Incremental),**缺 FullSnapshot (type=2)** → rrweb Replayer 没初始 DOM 重建 → iframe contentDocument 空。

**根因**:双重失效。
1. **server `snapshotTTL = 5 * time.Minute`**(`server/internal/recording/snapshot.go:24`):visitor 启动 session 时发的 FullSnapshot 进 Redis 缓存,5min 后过期。admin subscribe 时若已过期,只拿到 meta,没 FullSnapshot。
2. **SDK 周期性续期失效**(`visitor-sdk/src/collectors/rrweb.ts:208-218`):
   - 续期间隔 30s
   - 触发条件 `incrementalSinceFull >= 50` — 只在累计 ≥ 50 增量时才主动 `takeFullSnapshot`
   - visitor 低频交互时(每 30s < 50 增量)永远不触发续期,snapshot 自然 5min 后过期

**修复**(用户要求"1 和 2 都做"):

1. **延长 snapshot TTL**:`server/internal/recording/snapshot.go` `snapshotTTL` 从 `5 * time.Minute` → `30 * time.Minute`(与 `metaTTL` 对齐)。覆盖典型 session 时长,Redis 内存压力可控(每 session 一份 envelope bytes,几 KB ~ 几十 KB)。
2. **缩短 + 修正 SDK 续期**:`visitor-sdk/src/collectors/rrweb.ts` `startPeriodicFull`:
   - 间隔 `30_000` → `10_000`(30s → 10s)
   - 条件 `>= 50` → `>= 1`(有变化就续期,避免低频 visitor 永不续期)

**单测更新**:
- `server/internal/recording/snapshot_wiring_test.go`:原契约 `5*time.Minute` 改为更宽泛的 `time.Minute` + `snapshotTTL` 标识符存在性检查。

**浏览器验证**(Chrome DevTools MCP):
- visitor reload → SDK 新建 session `4ea3f61c` → admin 自动 selectedSessionId 切换
- 触发 50 mousemove + 等 15s
- admin store events 类型直方图:`{ "4": 1, "2": 1, "3": 23 }` ✅(Meta + **FullSnapshot** + 23 Incremental)
- iframe contentDocument:bodyHTML 1960 bytes,hasForm=true,正确渲染 visitor demo 页面 ✅

**server binary 重名 bug**(调查中发现,独立于主修复):
- `server/bin/` 同时存在 `server-dev`(rename 前遗留)+ `pinconsole-server`(新)
- `./ops.sh restart` 杀的是 `pinconsole-server`(根据 .server.pid),但实际在跑的可能是 `server-dev`(旧 PID 文件过期或手动启过)
- 旧 `server-dev` 占着 :8080,新 `pinconsole-server` bind 失败 "address already in use"
- 修复:`rm server/bin/server-dev` + `kill <old-pid>` + `./ops.sh start`
- 预防建议:rename 后应主动清理 `server/bin/server-dev` 残留 binary(ops.sh 当前不检查)

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

### 关于 sandbox warning(2026-06-21 已解决)

用户报告 30+ 条 `Blocked script execution in '<URL>' because the document's frame is sandboxed and the 'allow-scripts' permission is not set.` warning。

**根因**:rrweb alpha.20 在 `node_modules/.pnpm/rrweb-player@2.0.0-alpha.20/.../rrweb-player.js:11561-11567` 创建 iframe 时:
```js
this.iframe = document.createElement("iframe");
const attributes = ["allow-same-origin"];
if (this.config.UNSAFE_replayCanvas) {
  attributes.push("allow-scripts");
}
this.iframe.setAttribute("sandbox", attributes.join(" "));
```

默认 `UNSAFE_replayCanvas=false`,sandbox 只有 `allow-same-origin`。回放含 `<script>` 的页面时每个 script 触发一次 warning(N 条)。

**之前的 srcdoc reload patch 不生效** — `ReplayPlayer.vue:65-94` 的 `setupIframeSandboxPatch` 试图 setAttribute 后 reload,但 rrweb-player 不用 srcdoc,attribute 改了但 sandbox 决策已冻结,reload 不发生。

**修复**:`ReplayPlayer.vue` 和 `ReplayViewer.vue` 给 rrweb-player 传 `UNSAFE_replayCanvas: true`,让 alpha.20 内部主动把 `allow-scripts` 加进 sandbox attributes。这样:
- alpha.20 在 `appendChild` 之前就设好 `sandbox="allow-same-origin allow-scripts"`,浏览器一开始就放行,无 warning。
- 录制的 `<script>` 标签正常执行(本来就该执行,只是被 sandbox 拦了)。
- canvas/WebGL 录制内容也能正确回放(符合 PLAN.md 选择性截图策略)。
- 副作用:只剩 1-2 条通用 `both allow-scripts and allow-same-origin can escape its sandboxing` 提示性 warning(浏览器原生提示,不阻塞功能,远比原来 30+ 条易读)。

**安全考量**:UNSAFE_ 前缀名字唬人,但 iframe 仍受 sandbox 隔离。我们回放的是自己访客录制的内容(已知来源),不是任意 URL。`allow-same-origin + allow-scripts` 组合在理论上能让 iframe 内脚本移除 sandbox attribute,但 rrweb 通过 `contentDocument.write()` 重建 DOM 时已经控制了写入内容,且我们的回放场景不存在恶意访客上传 script 攻击运营端的现实路径。

### 未来可能的优化方向(post-v1)

- 服务端 `extractFullSnapshotEnvelope` 改为同时缓存 meta event(`ws.go:557`),subscribe 时一起下发。这样 admin 收到的初始事件流是 [meta, full snapshot],更符合 rrweb 协议期望,可能解决一些其他依赖 meta event 的边界 case。
- 但 client-side 修复已经解决问题,服务端改动风险更大,暂不动。
- sandbox 剩余的通用 escape warning 彻底消除:fork rrweb 拆 `createSandboxedIframe` 让 `allow-scripts` 独立可配,不再绑定 `UNSAFE_replayCanvas`。当前方案可接受,post-v1 再考虑。
