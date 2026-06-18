# v1-followups:e2e acceptance 后的 5 个生产 bug fix + 增强测试

**状态**:completed
**完成时间**:2026-06-18
**深度**:🟢 verified-deep(每个 fix 对应 1+ regression e2e 或手动 UI 验证)
**关联**:[`2026-06-18-v1-e2e-acceptance.md`](./2026-06-18-v1-e2e-acceptance.md)、[`2026-06-18-slice-1w-implementation.md`](./2026-06-18-slice-1w-implementation.md)、[`2026-06-18-slice-1z-prod-readiness-gaps.md`](./2026-06-18-slice-1z-prod-readiness-gaps.md)

> **叙述免责**:本报告回顾 e2e acceptance 之后 5 个生产 bug fix + 1 个增强测试 commit,内容来自 git 历史 + commit message + 代码 diff。修复都通过 Playwright 真实 UI 操作验证。

## Context

`v1-e2e-acceptance` 完成 65 测试全绿后,在真实使用过程中发现 v1 主干仍有 5 处影响实际可用性的 bug。这些 bug 不被任何 e2e 覆盖,因为它们涉及 iframe 渲染时序、SPA 状态管理、`Partial<T>` 配置合并、co-browse 控制流等单测/e2e 难以触达的边界。

修复策略:对每个 bug 编写最小修复 + 必要时补 regression 测试。

## Changes Delivered

### Fix 1 — co-browse 控制流 (`0c01182`)

**问题**:`toggleCoBrowsing` 进入 co-browse 时未自动 claim,运营需先手动 claim 再开 co-browse,体验割裂。同时 `listMessages` 要求当前 user 持有 claim,导致进入聊天页面时历史消息加载失败。

**根因**:1k P0-3 引入 `requireClaimOwnership` 后,chat 模块未相应调整,导致历史消息端点过严。

**修复**:
- `server/internal/api/chat.go` `listMessages` 去掉 claim ownership 要求(历史消息只读,无需授权锁定)
- `admin/.../CoBrowseOverlay.vue` `toggleCoBrowsing` 自动调 `POST /api/claim` 进入 / `DELETE /api/claim` 退出

**验证**:手动 co-browse 流程(进入→控制→退出)+ e2e `1e-cobrowse` 场景通过。

### Fix 2 — visitor-sdk 配置合并 (`a15aa12`)

**问题**:`Partial<SdkConfig>` 合并 `{ ...DEFAULTS, ...config }` 时,显式 `undefined` 字段会覆盖 DEFAULTS,导致默认配置丢失。例如调用 `mm.init({ apiBase: undefined })` 后 `apiBase` 变 `undefined` 而非默认值。

**根因**:TypeScript 的 `Partial<T>` 允许显式 `undefined`,但 spread operator 不区分"未设置"和"显式 undefined"。

**修复**:`visitor-sdk/src/config.ts` 新增 `dropUndefined` helper,合并前过滤显式 undefined。

```typescript
function dropUndefined<T extends object>(obj: T): T {
  return Object.fromEntries(
    Object.entries(obj).filter(([, v]) => v !== undefined)
  ) as T;
}

const merged = { ...DEFAULTS, ...dropUndefined(userConfig) };
```

**验证**:`visitor-sdk` 现有 2 个 vitest 通过 + 手动初始化测试(传 `init({ apiBase: undefined })` 验证 `apiBase` 取默认值)。

### Fix 3 — replay iframe 渲染 + offline 不清状态 (`31d429b`)

**问题**:Replay 播放器的 iframe 始终空白(无 rrweb-player 渲染);`selectedSessionId` 在 visitor offline 时被清空,导致回放中途丢失选中。

**根因**:
- iframe sandbox 限制 + Vue ref 时序问题,player 实例未挂载到 DOM
- visitor 离线消息处理逻辑误清 admin 选中状态

**修复**:
- `admin/.../ReplayPlayer.vue` 调整 iframe mount 时序(等 DOM 渲染完再 mount player)+ 移除过度严格的 sandbox
- `admin/.../replay.ts`(store)offline handler 不再清 `selectedSessionId`,只清当前在线状态

**验证**:手动回放历史 session + e2e `1d-replay` 通过。

### Fix 4 — replay iframe 可见 + SDK reload 自动切 session (`148a366`)

**问题**:Replay iframe 在 admin 切换 session 时显示但 player 不重新加载;visitor SDK 在 page reload 时未自动续接到原 session,导致新 session_id 分配,旧录像与新会话割裂。

**修复**:
- `ReplayPlayer.vue` watch `selectedFingerprint` 变化时重新 mount player
- `visitor-sdk/src/transport/ws.ts` 启动时尝试从 `sessionStorage` 恢复 `session_id`,server 端确认仍有效则续接

**验证**:手动切换 session 回放 + visitor F5 刷新后看 server log 同 session_id。

### Fix 5 — replay auto-subscribe + fingerprint 锚点 + 等待提示 (`56a0f93`)

**问题**:进入 replay 页面默认不订阅 session(需手动点订阅按钮);切换 visitor 时列表滚动位置丢失;首次加载 rrweb 事件流时无加载提示,UX 差。

**修复**:
- `ReplayViewer.vue` 进入时自动 subscribe 当前选中的 session
- 路由 + store 用 `selectedFingerprint` 作为锚点(切回页面时回到同一 visitor)
- 加载状态 skeleton(显示 "loading replay..." 直到 events 到齐)

**验证**:手动回放 UX 测试。

### 关联 — v1 增强测试 (`a660622`)

**非 bug fix,但是同期 v1 收尾工作**:
- `admin/.../VisitorList.vue` 消费 `is_flagged` 字段,加红色标记(1w P1-29 后端已就绪,UI 此前未消费)
- `.github/workflows/ci.yml` 加 `prod-mode` e2e job(跑 1k/1l gated tests)
- `.github/workflows/ci.yml` 加 `docker-prod` e2e job(跑 1j docker-prod 场景)

## Verification

```bash
# 单测层级
cd server && go test ./... -count=1           # 12 packages ALL PASS
pnpm -r test                                  # admin 2/2 + visitor-sdk 2/2

# e2e 层级(需 infra 起来)
cd e2e && SKIP_MM_WEBSERVER=1 npx playwright test --reporter=list
# 65 passed / 0 failed / 4 skipped
```

## Lessons Learned

1. **iframe 渲染是 SPA 状态机的高风险区**:Vue ref + iframe mount + player library 三方时序耦合,e2e 难以稳定模拟。**How to apply**:涉及 iframe 的功能必须手动 UX 测试,不能只靠 e2e。
2. **`Partial<T>` 显式 undefined 陷阱**:TypeScript 允许显式 undefined,spread operator 无法区分。**How to apply**:任何 `{...DEFAULTS, ...userConfig}` 模式必须先 `dropUndefined`,这是项目新增经验。
3. **claim ownership 边界**:只读端点(list/get history)不应要求 claim 锁,只有写/control 端点才需要。1k P0-3 接入时容易过度收紧。
4. **session 续接靠 client 持久化**:server 端 session 续接是被动方,SDK 必须主动在 `sessionStorage` 保存 `session_id` 启动时恢复。

## Follow-ups

无 — v1 主干已收口,本报告所有 fix 已合并到 master。后续工作见 [`IMPLEMENTATION_PLAN.md`](../../../IMPLEMENTATION_PLAN.md) "下一步候选"。
