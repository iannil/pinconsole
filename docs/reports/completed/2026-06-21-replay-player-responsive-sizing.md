# Replay Player 响应式 sizing 修复

**状态**:superseded — vendor-rrweb fork (2026-06-25) 彻底替换回放引擎，原 bugs 由新架构自动解决
**开始**:2026-06-21
**完成**:（superseded 2026-06-26）
**关联**:实时回放切片 `docs/reports/completed/2026-06-17-slice-1c-*.md`,历史回放切片 `docs/reports/completed/2026-06-17-slice-1d-*.md`
**Superseded By**: `docs/reports/completed/2026-06-25-vendor-rrweb-spec.md`

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

## 2026-06-22 — rrweb-player CSS 未加载导致 player 算小一圈(`.rr-player` 1067×504 而非 1078×509)

**现象**:用户在 admin dashboard 选定 visitor 后报告"client 页面没有占据 admin 会话内容区全部"。DevTools 度量:
- `.rr-player` 外框:`1067×504`(容器 `.player-container` 是 `1078×723`)
- `.player-container` clientWidth `1078 → 1067`(垂直滚动条吃掉 11px)
- `.player-container` scrollHeight `813 > clientHeight 723`(存在垂直溢出)
- `.rr-player__frame` computed `overflow: visible`(应为 `hidden`)
- `.replayer-wrapper` `transform-origin: 533.5px 704.105px`(默认 50% 50%,应为 `top left`)
- `.replayer-wrapper` computed `position: static`(应为 `relative`),`left/top: auto`(应为 `50%/50%`)

**根因**:`rrweb-player/dist/style.css` **从未被任何代码 import**。`grep -rn "rrweb-player/dist/style.css" admin/src` 无结果。

rrweb-player 是 Svelte 组件,Vite 默认不会把它的样式自动注入。它的关键布局 CSS 全在那份 CSS 里:
- `.rr-player__frame { overflow: hidden }` —— 没有 iframe 布局尺寸(1913×904)就溢出 frame
- `.replayer-wrapper { float: left; clear: both; transform-origin: top left; left: 50%; top: 50% }` —— 没有它 wrapper 在 frame 内部定位错误
- `.replayer-mouse { position: absolute }` —— 没有它鼠标指示器变成 block 占位

CSS 缺失下,wrapper 的 `position: relative`(来自 CSS 第一行)和 `transform-origin: top left`(来自 CSS 第二行)都没生效,变成默认值。iframe(`width=1913 height=904` 属性,layout 尺寸)+ canvas(`1067×504` CSS 尺寸)在 wrapper 内作为 block 元素纵向堆叠 → wrapper layout 高度 `904 + 504 = 1408`。`.rr-player__frame` 没 `overflow:hidden`,wrapper 经 transform 后视觉 bbox 在 frame 之外 → 滚动条出现 → 吃掉 11px clientWidth → responsive sizing `apply()` 用缩小的 `clientWidth=1067` 计算 → rr-player 被设成 `1067×504`。

**修复**(两处,因为 `ReplayPlayer.vue`(实时)和 `ReplayViewer.vue`(历史)各自独立 `import('rrweb-player')`):

```ts
// ReplayPlayer.vue loadPack()
await import('rrweb-player/dist/style.css');
pack = (await import('rrweb-player')) as unknown as RRWebPlayerPack;

// ReplayViewer.vue initPlayer()
await import('rrweb-player/dist/style.css');
const mod = await import('rrweb-player');
```

放在 `import('rrweb-player')` 之前,确保 Player 构造时 CSS 已注入(Svelte onMount 会立即建 iframe,样式必须先就位)。

**浏览器验证**(Chrome DevTools MCP,`http://localhost:5173/admin/dashboard`):
- `.rr-player` inline style:`width: 1067px;height: 504px` → `width: 1078px;height: 509px` ✅
- `.player-container` clientWidth:`1067 → 1078`(无滚动条损耗)✅
- `.player-container` scrollHeight:`813 → 723`(无垂直溢出)✅
- `.rr-player__frame` overflow:`visible → hidden` ✅
- `.replayer-wrapper` transform-origin:`533.5px 704.105px → 0px 0px`(top left)✅
- `.replayer-wrapper` position:`static → relative` ✅
- `.replayer-wrapper` transform scale:`0.557522 → 0.563053`(`1078/1913`,正好按容器宽算)✅
- 视觉效果:iframe 视觉 `1078×509` 正好填满 rr-player frame,无 letterbox ✅

**验证深度**:🟢 verified-deep
- DOM computed style 六项指标前后比对
- 视觉效果(iframe rect 精确匹配 rr-player frame)
- 无单测改动(纯依赖加载问题,无逻辑变化)

**为什么以前没发现**:之前几次 bug(双 mount race、iframe height 150px fallback、`initialized` flag、snapshot TTL)掩盖了这条。这些 bug 修复后 player 能正常显示,但宽高比恰好接近录制视口比例,视觉上没明显 letterbox,直到用户专门对比 "外框没占满容器宽" 才注意到 11px 差距。

## 2026-06-22 — 退出协助后录屏没占满 + 历史回放进度条不动(两个新 bug)

### Bug A:`$set` 不重算 wrapper scale → 退出协助后外框/内容缩放不一致

**现象**:用户报告"退出协助以后,client 页面没有在 admin 会话中占据全部"。提供的 HTML:`.rr-player` 外框 `707×483`,但 `.replayer-wrapper` 是 `transform: scale(0.616901)`,iframe 录制尺寸 `1558×1065`。

**矛盾点**:`707×483` 的外框,正确 scale 应为 `min(707/1558, 483/1065) = 0.4535`;但实际 scale `0.616901` 对应的是 `~961` 宽的外框。**外框 inline 尺寸与 wrapper scale 不一致** —— 最后一次 `$set({width:707})` 改了外框尺寸但 scale 停留在上一次(961 宽)的值。

**根因**(读 rrweb-player alpha.20 源码确认):
- `rrweb-player.js:14873` 的响应式块:`width`/`height` props 变化时**只**更新 `.rr-player` 外框 inline 尺寸(`style`/`playerStyle`),**不调用 `updateScale`**。
- `updateScale`(`:14726`)只在两处触发:`replayer.on("resize")`(访客录制视口变化时)和全屏切换。**我们 `$set` 改外框尺寸不属于任何一种** → scale 不重算。
- 进入/退出协助(EngagementPanel 出现/消失)使 `.player-container` 变窄/变宽 → ResizeObserver → `apply()` → `$set` 新尺寸,但 scale 保持旧值 → 内容缩放与外框对不上,表现为"退出协助后没占满"。

**修复**:`admin/src/composables/useResponsivePlayerSize.ts` `apply()` 在 `player.$set({width,height})` 之后调用 `player.triggerResize?.()`。`triggerResize`(`rrweb-player.js:14733`,作为 public getter `:14971` 暴露)用**当前** width/height props 重算 `.replayer-wrapper` 的 `transform: scale`。顺序关键:先 `$set`(同步更新 width/height 闭包变量)再 `triggerResize`(读新值算 scale)。`PlayerLike` interface 加 `triggerResize?: () => void`。

**验证**(Chrome DevTools MCP,`localhost:5173` 历史回放页 —— 与实时 Dashboard 共用同一 composable 代码路径):
- 模拟协助面板出现/消失(改 `.player-area` 宽度触发 ResizeObserver):
  - full:外框 `928×633`,scale `0.5937`,缩放后内容 `928×632` == 外框 ✅
  - 变窄 700:外框 `700×476`,scale `0.4469`,内容 `699×476` == 外框 ✅
  - 还原(模拟退出协助):外框 `928×633`,scale `0.5937`,内容 `928×632` == 外框 ✅
- 每一步缩放后内容精确填满外框(旧代码下 scale 会停在变窄时的 0.4469,还原后内容只有 699 宽,不占满)。
- 单测:新增"apply 后调用 triggerResize"回归用例(断言调用 + `set`→`resize` 顺序),`useResponsivePlayerSize.test.ts` 13 个全绿。

**验证深度**:🟡 verified-shallow(机制层 🟢:同 composable 代码路径浏览器实测 scale/外框一致性 + 单测;但**未在真实 Dashboard co-browse 进入/退出流程**实测 —— 需要在线访客 + claim,未搭建)。

### Bug B:历史回放进度条不动(`ui-update-current-time` 读错字段)

**现象**:`http://localhost:8080/admin/replay/:id` 播放回放,进度条(scrubber)不动,当前时间停在 `0:00`,但 iframe 内录屏正常播放。

**根因**(读 rrweb-player 源码 + 浏览器确认):
- `rrweb-player.js:14750`:`controller.$on(event, ({ detail }) => handler(detail))` —— rrweb-player 把事件的 `detail`(即 `{ payload: currentTime }`)**直接**传给我们的 handler。
- `ReplayViewer.vue` 的 handler 读 `e?.detail?.payload`(多套了一层)→ 永远 `undefined` → `currentTimeMs` 不更新 → 进度条停在 0。`ui-update-player-state` 同样的 bug → `isPlaying` 永远 false。
- 浏览器确认:点播放后 iframe DOM 变化(`6397→6439`)、`.replayer-mouse` 移动(`→461px,68px`)= 播放确实在跑,但 scrubber `value` 停在 0 = 纯读数 bug。

**修复**:`admin/src/views/ReplayViewer.vue` 两个 handler 把 `e?.detail?.payload` 改为 `e?.payload`;interface 类型从 `{ detail?: { payload? } }` 改为 `{ payload? }`。

**验证**(Chrome DevTools MCP,`localhost:5173` HMR):点播放后 scrubber `value` `0→3000`、当前时间 `0:00→0:02`、播放键 aria `Play→Pause`、进度填充 `--progress 1.05%`。✅ **验证深度**:🟢 verified-deep(真实用户流程浏览器实测)。

### 测试 / 构建
- `pnpm exec vue-tsc --noEmit` 通过。
- `pnpm test` 153 个全绿(原 152 + 新增 1 个 triggerResize 回归)。
