# 切片 1f 规格说明（表单 + 跳转：精细化 co-browsing）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7、[`docs/project-status.md`](../project-status.md) §5
**前置**：[切片 1e 完成](./2026-06-17-slice-1e-implementation.md)

## Context

把 1e 的"坐标 fallback co-browsing"升级为"nodeID 精确 + 跨页面续接 + 浮动输入框代填 + Toast 提示 + navigate 白名单"。

## 范围（acceptance criteria）

- [ ] admin overlay 点击 rrweb-player iframe 时通过 postMessage 取 nodeID
- [ ] 浮动输入框：运营点 input 弹出，onBlur 后 fill_input
- [ ] 访客端 Toast：fill_input 时右上角浮动提示
- [ ] presence.navigated 消息：含 old_session_id + new_session_id + visitor_id
- [ ] admin 自动重订阅新 session（presence.navigated 收到后）
- [ ] navigate 白名单：env var `NAVIGATE_ALLOWED_DOMAINS` + SDK data-attribute
- [ ] 访客跳转 = 新 session（同 visitor）
- [ ] 端到端 4 场景：浮动输入框代填 / nodeID+click 跨 iframe / navigate 自动重订阅 / 白名单拒绝
- [ ] Go 单元测试 + Playwright e2e 全部通过

## 9 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | 跨页面 session | 访客跳转 = 新 session（同 visitor） |
| 2 | admin nodeID 获取 | postMessage 跨 iframe |
| 3 | 表单 UI | 继续 overlay 模式 |
| 4 | 键盘转发 | fill_input onBlur（防抖 300ms） |
| 5 | 访客端代填提示 | Toast + border |
| 6 | navigate 后状态 | admin 自动重订阅新 session |
| 7 | admin 代填入口 | 浮动输入框 |
| 8 | navigate 白名单 | env var + SDK data-attribute |
| 9 | 1f 验收 | 4 项 |

## 涉及的代码改动

### server/

修改：
- `internal/proto/envelope.go`：PresencePayload 加 Navigated 字段（old_session_id, new_session_id）
- `internal/api/command.go`：navigate 白名单从 env 读
- `internal/config/config.go`：加 `NavigateAllowedDomains []string`
- `internal/api/ws.go`：visitor hello 时记录 visitor_id（已有）；broadcast navigated

### visitor-sdk/

新建：
- `src/commands/toast.ts`：右上角浮动 toast

修改：
- `src/index.ts`：页面 unload 检测 navigate，broadcast navigated
- `src/commands/handler.ts`：doFill 时显示 toast
- `src/config.ts`：读取 data-allowed-domains

### admin/

新建：
- `src/components/FloatingInput.vue`：浮动输入框

修改：
- `src/components/CoBrowseOverlay.vue`：postMessage 取 nodeID
- `src/components/ReplayPlayer.vue`：iframe sandbox allow-same-origin allow-scripts；监听 message
- `src/composables/useWs.ts`：presence.navigated 自动重订阅
- `src/stores/visitors.ts`：支持 navigated presence

### e2e/

修改：
- `tests/realtime.spec.ts`：加 4 个 1f 场景

## 估时

- **Solo 全职**：约 5-7 天
- **Solo 业余**：约 2-3 周

## Next

实施完切片 1f 后：

1. 写 `docs/reports/completed/{date}-slice-1f-implementation.md`
2. 更新 `docs/project-status.md` §5 切片状态表（1f → completed）
3. 启动切片 1g（弹窗 + 聊天）

## Blockers

无。1e 双向通道已建立，1f 在此基础上加精确化。

## Notes

- 1f 不实现表单字段类型识别（自动填用户名/邮箱等智能推断），运营手动选 input
- 1f 不实现表单提交（form_submit），与 fill_input 重复
- 1f 不实现访客 SPA 内路由变化的自动续接（仅全页面跳转）
- navigate 白名单默认含当前域名 + 平台白名单 + SDK data-attribute + env var
