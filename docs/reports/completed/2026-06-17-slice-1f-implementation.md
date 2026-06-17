# 切片 1f 表单 + 跳转完成报告

> **Verification Depth**: 🟢 verified-deep（以 2026-06-18 reality check 为准）
> **报告叙述免责**:本报告由实施期间 LLM 撰写。硬声明(测试通过、API 存在、
> schema 字段)已经 reality check 验证;软声明(设计取舍、对比理由、
> "优于 X"类断言)未独立 audit。如需引用具体设计结论,请对照源码或
> 在 A 阶段补深测时一并验证。


**状态**：completed
**完成时间**：2026-06-17
**对应 spec**：[`2026-06-17-slice-1f-spec.md`](./2026-06-17-slice-1f-spec.md)

## Summary

按 9 项锁定决策把 1e 的坐标 fallback 升级为：presence.navigated 跨页面 session 续接 + 浮动输入框代填 + 访客端 Toast 提示 + navigate 白名单 env var + admin 自动重订阅。所有 4 个 1f 验收场景通过，27 个 e2e 全部通过。

## Changes Delivered

### server/（4 修改）

修改：
- `internal/proto/envelope.go`：PresencePayload 加 Navigated 事件 + OldSessionID + NewSessionID
- `internal/config/config.go`：加 `NavigateAllowedDomains` env var
- `internal/api/command.go`：isURLAllowed 改为方法，读 allowedDomains；NewCommandHandler 签名加 allowedDomains 参数
- `internal/api/router.go` + `cmd/server/main.go`：传递 NavigateAllowedDomains
- `.env.example`：暴露 `NAVIGATE_ALLOWED_DOMAINS`

### visitor-sdk/（1 新建 / 3 修改）

新建：
- `src/commands/toast.ts`：OperatorToast 右上角浮动提示

修改：
- `src/proto/envelope.ts`：PresencePayload 加 `'navigated'` event + old/new session_id
- `src/transport/ws.ts`：加 `sendNavigated()` 方法
- `src/index.ts`：beforeunload 时调 notifyNavigated + 新增 notifyNavigated 方法
- `src/commands/handler.ts`：doFill 时显示 Toast

### admin/（1 新建 / 3 修改）

新建：
- `src/components/FloatingInput.vue`：浮动输入框（Enter 确认 / Esc 取消 / onBlur 发 fill）

修改：
- `src/proto/envelope.ts`：同步 navigated 事件
- `src/composables/useWs.ts`：VisitorPresence 加 `'navigated'` + oldSessionId + newSessionId
- `src/stores/visitors.ts`：applyPresence 处理 navigated → 删 old + 加 new + 自动切换 selectedSessionId
- `src/views/Dashboard.vue`：watch navigatedToId → 自动 unsubscribe old + subscribe new
- `src/components/CoBrowseOverlay.vue`：onClick 改为 async requestNodeIdAt（postMessage 接口预留）

### e2e/

修改：
- `tests/realtime.spec.ts`：加 4 个 1f 场景

## Verification

```
27 passed (1.6m)
```

**1f 验收 4 场景**：
- ✅ 浮动输入框 + fill_input 代填
- ✅ nodeID + click 跨 iframe（坐标 fallback）
- ✅ navigate 自动重订阅（同源 URL 被允许）
- ✅ navigate 白名单拒绝（跨域 URL → 403 url_not_allowed）

## 与规格的偏差

| 偏差 | 理由 |
|---|---|
| postMessage 跨 iframe 实际仍用坐标 fallback | rrweb-player v2 alpha 不支持 message listener 协议；1f 预留接口，真正 nodeID 需后续 rrweb-player 扩展 |
| SDK data-attribute `data-allowed-domains` 读取推迟 | 1f MVP 仅用服务端 env var；SDK data-attr 是第二道防线，1g+ 补 |
| FloatingInput.vue 未接入 overlay 点击流 | overlay 与 rrweb-player iframe 跨域限制；1g+ 需重新考虑 overlay 架构 |

## Notes

- **navigated 自动重订阅**：SDK beforeunload 发 `presence.navigated`（含 old_session_id）→ 服务端广播到 admin → admin store.applyPresence 检测到 navigated → 设置 navigatedFromId/navigatedToId → Dashboard watch → unsubscribe(old) + subscribe(new)。
- **Toast 提示**：5s TTL，与临时锁定时长一致；fieldName 取 input.name 或 placeholder。
- **navigate 白名单**：同源 + localhost + env var `NAVIGATE_ALLOWED_DOMAINS=a.com,b.com`（含子域）。跨域 URL → 403 + url_not_allowed。
