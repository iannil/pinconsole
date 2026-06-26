# 2026-06-22 实时输入不渲染 + 输入脱敏可配置

## 问题(用户报告)

1. client 端的输入操作没有在 admin 后台实时更新,需刷新后重新进入会话才能看见。
2. 输入内容显示为 `***`。

## 根因分析

### 症状 1:实时事件不渲染(真 bug)

`admin/src/components/ReplayPlayer.vue` 用 rrweb-player(autoPlay)创建 player,
后续实时事件通过 `player.addEvent()` 推送。

追踪 `rrweb-player@2.0.0-alpha.20` 源码状态机(`createPlayerService`):

- player 把缓冲事件快进播完后,最后一个事件 cast 时触发 `END` → 进入 `paused`
  状态并 `timer.clear()`(`rrweb-player.js` L11168-11182 / L10374-10377)。
- `paused` 状态下 `ADD_EVENT` 走 `addEvent` action(L10493):
  `isSync = event.timestamp < baselineTime`;实时事件 timestamp 晚于 baseline →
  `isSync=false`;此时 `timer.isActive()` 也为 false → **事件只 push 进内部数组,
  既不同步 cast 也不入 timer,因此不渲染**。
- 只有重建 player(刷新/重进会话,走 `rebuildPlayer` 全量 events)才会重新播出。

→ 表现为"输入操作不实时更新,需刷新重进才看见"。注意这其实影响所有增量事件,
输入只是用户最易察觉的。

### 症状 2:输入显示 ***(设计行为,非 bug)

`visitor-sdk/src/collectors/rrweb.ts` 默认 `maskAllInputs: true` +
`maskInputOptions.{text,textarea,email,...}: true`,即全部文本输入脱敏。
这是隐私安全基线(CLAUDE.md / PLAN.md §10:提交前按键监听属 GDPR 敏感处理)。

## 修复

### 症状 1 — 切入 rrweb live 模式(`admin/src/components/ReplayPlayer.vue`)

player 创建后监听 `finish` 事件;播放结束时调用 `getReplayer().startLive(baseline)`
切到 `live` 状态。`baseline` 设为远未来(`Date.now() + 365d`),使后续 `addEvent`
的 `event.timestamp < baselineTime` 恒成立 → 同步立即渲染,且规避访客/运营两端
时钟偏移导致的延迟。`liveMode` 标志防重复,`rebuildPlayer`/sessionId 切换时重置。

### 症状 2 — 输入脱敏可配置(`visitor-sdk`)

用户决策:**做成部署方可配置**,默认仍全部脱敏。

- `config.ts`:新增 `unmaskInputs?: boolean`(默认 false),支持
  `data-unmask-inputs` script 属性 / `window.MM_CONFIG`。
- `index.ts`:`unmaskInputs=true` 时给 RRWebCollector 传
  `{ maskAllInputs:false, maskInputOptions:{ password:true } }`——展示文本输入,
  **password 始终强制脱敏**(安全/合规底线)。开启需部署方自行通过 consent 流程
  取得同意并在隐私政策披露。

## 验证深度

- 🟡 症状 2:`tests/config.test.ts` + `tests/collectors_wiring.test.ts` 全绿(25),
  默认 `maskAllInputs=true` 不变;`tsc --noEmit` 通过。
- 🔴 症状 1:`vue-tsc --noEmit` 通过;**未做端到端运行时验证**(需 server +
  访客页 + admin 三方联调观察实时输入渲染)。逻辑已对照 rrweb-player 源码状态机定位
  到根因,但需实跑确认 `finish`→`startLive` 时序在真实事件流下无渲染缺口。

## 待办 / 注意

- `server/cmd/server/embedded/{admin,sdk}` 的产物由 CI 构建,本次仅改源码;
  需重新构建前端产物后实时行为才在单二进制中生效(未手动改 embedded 产物)。
- 建议后续补一条 admin e2e:订阅活跃 session,断言新 input 事件无需 rebuild 即渲染。
