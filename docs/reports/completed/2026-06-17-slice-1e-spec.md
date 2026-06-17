# 切片 1e 规格说明（双向通道：co-browsing 起步）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §5 §7、[`docs/project-status.md`](../project-status.md) §5
**前置**：[切片 1d 完成](./2026-06-17-slice-1d-implementation.md)

## Context

把 1c 的"运营只看"升级为"运营可控"。5 个核心命令（cursor_highlight / click / scroll / fill_input / navigate）通过访客既有 WS 反向下发。访客端有 SVG 圆点跟随、临时锁定 input、ESC 紧急退出。所有命令进 PG 审计表。

## 范围（acceptance criteria）

- [ ] 5 个核心命令：cursor_highlight / click / scroll / fill_input / navigate
- [ ] 命令下行复用既有 visitor WS（不新增端点）
- [ ] admin Dashboard 加 Start/Stop Co-browsing 按钮
- [ ] CoBrowseOverlay 透明 div + elementFromPoint 捕获运营鼠标
- [ ] nodeID 用 rrweb-snapshot buildNodeID 算法，SDK 维护 nodeID→DOM map
- [ ] cursor_highlight 命令 rAF + 30fps 节流
- [ ] 访客端 SVG 圆点 + 运营名字渲染（z-index 999999, pointer-events none）
- [ ] fill_input 触发 5s 临时锁定 input，期间访客输入被覆盖
- [ ] navigate 仅允许同源或平台白名单域名
- [ ] 访客 ESC 三连或 Ctrl+Shift+X 触发紧急退出
- [ ] PG 表 co_browsing_commands 记录每个命令
- [ ] 端到端验证 4 场景：cursor_highlight / click / fill_input / ESC 紧急退出
- [ ] Go 单元测试 + Playwright e2e 全部通过

## 12 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | 命令集 | cursor_highlight / click / scroll / fill_input / navigate |
| 2 | 下行通道 | 复用访客 WS |
| 3 | 控制模式入口 | 显式 Start/Stop Co-browsing 按钮 |
| 4 | Overlay | 透明 div + elementFromPoint |
| 5 | 访客端节点定位 | rrweb-snapshot buildNodeID + nodeID→DOM map |
| 6 | 冲突解决 | 运营代填 5s 临时锁定 input |
| 7 | navigate 安全 | 仅同源 + 平台域名白名单 |
| 8 | 紧急退出 | ESC 三连或 Ctrl+Shift+X |
| 9 | 审计日志 | PG 表 co_browsing_commands |
| 10 | cursor_highlight 节流 | rAF + 30fps |
| 11 | 访客端高亮 | SVG 圆点 + 运营名字 |
| 12 | 1e 验收 | 4 项 |

## 涉及的代码改动

### server/

新建：
- `migrations/000002_cobrowsing.up.sql` + `.down.sql`：co_browsing_commands 表
- `internal/api/command.go`：POST /api/sessions/:id/command REST 端点

修改：
- `internal/hub/hub.go`：加 SendCommandToVisitor(sessionID, bytes) 反向路由
- `internal/hub/client.go`：visitor client 加命令下行（已有 writeCh 可复用）
- `internal/api/ws.go`：visitorWS 处理 release_control 等消息；operatorWS 处理 send_command
- `internal/api/router.go`：注册 command 路由
- `internal/storage/queries.go`：加 CoBrowsingCommand struct + CreateCoBrowsingCommand + ListCoBrowsingCommandsBySession

### visitor-sdk/

新建：
- `src/commands/handler.ts`：接收 command + 执行 + 临时锁定 + ESC 监听
- `src/commands/cursor.ts`：SVG 圆点渲染
- `src/commands/nodeMap.ts`：rrweb-snapshot buildNodeID + map

修改：
- `src/index.ts`：启动 commandHandler
- `src/transport/ws.ts`：onMessage 已支持，无需改

### admin/

新建：
- `src/components/CoBrowseOverlay.vue`：透明 div + elementFromPoint
- `src/components/OperatorCursor.vue`：admin 端运营光标位置标记

修改：
- `src/views/Dashboard.vue`：加 Start/Stop Co-browsing 按钮
- `src/composables/useWs.ts`：加 sendCommand 方法
- `src/components/VisitorPanel.vue`：嵌入 CoBrowseOverlay
- `src/api/sessions.ts`：加 sendCommand REST 客户端

### e2e/

修改：
- `tests/realtime.spec.ts`：加 4 个 1e 场景

## 命令协议（伪 schema）

```typescript
// 运营 → 服务端：POST /api/sessions/:id/command
interface CommandRequest {
  type: 'cursor_highlight' | 'click' | 'scroll' | 'fill_input' | 'navigate' | 'release_control';
  payload: CommandPayload;
}

// 服务端 → 访客：复用 envelope
interface Envelope {
  type: 'command';
  payload: CommandPayload;
}

interface CommandPayload {
  type: string;
  ts: number;
  cursor?: { x: number; y: number; name: string };
  click?: { node_id: number; x: number; y: number };
  scroll?: { x: number; y: number };
  fill_input?: { node_id: number; value: string };
  navigate?: { url: string };
}
```

## 估时

- **Solo 全职**：约 8-10 天
- **Solo 业余**：约 3-4 周

## Next

实施完切片 1e 后：

1. 写 `docs/reports/completed/{date}-slice-1e-implementation.md`
2. 更新 `docs/project-status.md` §5 切片状态表（1e → completed）
3. 启动切片 1f（表单 + 跳转：代填完整 + 跳转接管）

## Blockers

无。1c/1d 已建立完整事件通道 + 录像归档。1e 在此基础上加反向控制。

## Notes

- 1e 不实现表单提交（form_submit），与 fill_input 重复
- 1e 不实现运营多并发（1h 起 claim/release 锁）
- 1e 不做权限分级（1h 起）
- 1e 不做命令撤销（v1 不需要）
- navigate 白名单默认含当前域名 + 平台域名，可在 SDK data-attribute 配置额外白名单
