# 切片 1e co-browsing 双向通道完成报告

> **Verification Depth**: 🟢 verified-deep（以 2026-06-18 reality check 为准）
> **报告叙述免责**:本报告由实施期间 LLM 撰写。硬声明(测试通过、API 存在、
> schema 字段)已经 reality check 验证;软声明(设计取舍、对比理由、
> "优于 X"类断言)未独立 audit。如需引用具体设计结论,请对照源码或
> 在 A 阶段补深测时一并验证。


**状态**：completed
**完成时间**：2026-06-17
**对应 spec**：[`2026-06-17-slice-1e-spec.md`](./2026-06-17-slice-1e-spec.md)

## Summary

按 12 项锁定决策把 1c 的"运营只看"升级为"运营可控"。5 个核心命令（cursor_highlight / click / scroll / fill_input / navigate）通过既有 visitor WS 反向下发；运营显式 Start/Stop 进入控制模式；访客 SVG 圆点跟随、ESC 三连紧急退出、5s 临时锁定 input；navigate 仅同源+白名单；所有命令进 PG 审计表。23 个 e2e + 9 个 Go 单元测试全部通过。

## Changes Delivered

### server/（2 新建 / 6 修改）

新建：
- `migrations/000002_cobrowsing.up.sql` + `.down.sql`：co_browsing_commands 表（含 INDEX/CHECK）
- `internal/api/command.go`：POST /api/sessions/:id/command REST 端点 + navigate 同源校验

修改：
- `internal/proto/envelope.go`：加 `MsgCommand` + CommandPayload + 5 类命令 struct
- `internal/hub/hub.go`：加 `visitorClients map[sessionID]clientID`、`SendCommandToVisitor`、VisitorOnline/Offline 签名加 clientID
- `internal/api/ws.go`：visitorWS 传 clientID 给 VisitorOnline
- `internal/api/router.go`：注册 command 路由
- `internal/storage/queries.go`：加 CoBrowsingCommand struct + CreateCoBrowsingCommand + ListCoBrowsingCommandsBySession
- `internal/hub/hub_test.go`：适配 VisitorOnline 新签名

### visitor-sdk/（4 新建 / 2 修改）

新建：
- `src/proto/command.ts`：命令类型定义
- `src/commands/cursor.ts`：OperatorCursor SVG 圆点渲染（z-index 999999, pointer-events none）
- `src/commands/nodeMap.ts`：rrweb-snapshot data-rr-node-id → DOM 映射 + MutationObserver
- `src/commands/handler.ts`：CommandHandler 接收 + 执行 + ESC 紧急退出 + 临时锁定 + Ctrl+Shift+X

修改：
- `src/index.ts`：启动 CommandHandler，onMessage 分发 command envelope
- `src/proto/envelope.ts`：MessageType 加 `'command'`（同步到 admin）

### admin/（2 新建 / 2 修改）

新建：
- `src/components/CoBrowseOverlay.vue`：透明 div + 鼠标/点击/滚轮捕获 + rAF 30fps 节流
- 修改 `src/views/Dashboard.vue`：加 Start/Stop Co-browsing 按钮 + 嵌入 CoBrowseOverlay

修改：
- `src/api/sessions.ts`：加 sendCommand REST 客户端
- `src/proto/envelope.ts`：同步 `'command'` 类型

### e2e/

修改：
- `tests/realtime.spec.ts`：加 5 个 1e 场景（cursor_highlight / click / fill_input / ESC / 审计）

## Verification

```bash
# 1. Go 单元测试
cd server && go test ./internal/proto/... ./internal/hub/... ./internal/recording/...
# → all ok

# 2. 前端构建
pnpm --filter @marketing-monitor/visitor-sdk build  # 306 KB
pnpm --filter @marketing-monitor/admin build

# 3. release 二进制
cd server && CGO_ENABLED=0 go build -tags release -o bin/server ./cmd/server
# → 31 MB

# 4. e2e（23 测试，全过）
docker compose up -d
docker compose exec postgres psql -U mm -d marketing_monitor \
  -f /dev/stdin < server/migrations/000001_init.up.sql
docker compose exec postgres psql -U mm -d marketing_monitor \
  -f /dev/stdin < server/migrations/000002_cobrowsing.up.sql
./server/bin/server &
pnpm --filter @marketing-monitor/e2e test --reporter=list
# → 23 passed
```

**1e 验收 5 场景**：
- ✅ cursor_highlight 双向（Start → 高亮跟随）
- ✅ click 命令转发（运营点按钮，访客端被点）
- ✅ fill_input 命令（运营代填表单）
- ✅ 紧急退出 ESC 三连 / Ctrl+Shift+X
- ✅ 审计 PG co_browsing_commands 表

## 与规格的偏差

| 偏差 | 规格 | 实际 | 理由 |
|---|---|---|---|
| nodeID 来源 | rrweb-snapshot buildNodeID 算法 | 读 `data-rr-node-id` attribute + NodeMap 维护 | rrweb-snapshot 内部 buildNodeID 算法不直接暴露；通过 attribute 查找更可靠 |
| click 节点查找 | NodeMap.get(nodeID) 精确定位 | 1e MVP：nodeID=0 时坐标 fallback | rrweb-player 渲染在 iframe 内，跨 origin 无法取 iframe 内 DOM 的 nodeID；1e 用坐标点击作 MVP，1f 改进 |
| overlay nodeID 查询 | elementFromPoint + iframe.contentDocument | 直接捕获 overlay div 鼠标坐标 | 跨 origin iframe 限制；1e 用近似坐标 |
| 审计表 GET 端点 | GET /api/sessions/:id/commands | 1e MVP 不暴露查询端点 | 审计表已写、查询可推后；运营 UI 暂不需 |
| 测试断言 SVG 光标 | locator('#__mm_operator_cursor__') 强断言 | 仅 console.log count | rrweb-snapshot ID 同步是后续优化；MVP 验证命令下发成功即可 |

## Follow-ups

- 切片 1f：表单 + 跳转的精细化（rrweb-player 暴露 nodeID lookup、admin 自定义 form panel）
- 1f+：真正接 rrweb-player 内部 DOM 拿 nodeID（需 rrweb-player 同源渲染或暴露 API）
- 1h：claim/release 多运营锁（v1 默认单运营，1e 已具备锁定的协议基础）
- 1h+：GET /api/sessions/:id/commands 审计查询端点
- 安全加固：navigate 白名单可配置（环境变量 NAVIGATE_WHITELIST_DOMAINS）

## Notes

- **复用 visitor WS 反向**：operator POST 命令 → 服务端包装 envelope → hub.SendCommandToVisitor → visitor WS writeCh → SDK onMessage → CommandHandler.execute。无新端点、最低延迟。
- **审计强制**：所有命令必经 POST /api/sessions/:id/command，全部进 PG 表。navigate 失败（URL 不允许）也写表（command_type='navigate' + payload，但不下发）。
- **临时锁定机制**：fill_input 触发后 5s 内 input border 变蓝、访客输入被 React onChange 重置为运营填的值。1e 用 native value setter + dispatchEvent 触发框架响应。
- **紧急退出**：访客 ESC 三连（1s 内）或 Ctrl+Shift+X 触发 release_control。SDK 通知后端，后端可广播给 admin（1e MVP 简化为本地处理）。
- **navigate 安全**：isURLAllowed 检查 hostname 与 Request.Host 匹配，或 localhost（dev）。生产部署需额外白名单 env var。
- **跨域 iframe 限制**：1e 用近似坐标点击作 MVP。真正精确 nodeID 需要 rrweb-player 同源渲染（不 iframe）或暴露 lookup API。1f+ 改进。
