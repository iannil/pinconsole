# 切片 1g 弹窗 + 聊天完成报告

**状态**：completed
**完成时间**：2026-06-17
**对应 progress**：[`docs/progress/2026-06-17-slice-1g-implementation.md`](../../progress/2026-06-17-slice-1g-implementation.md)
**规格来源**：[`docs/progress/2026-06-17-slice-1g-spec.md`](../../progress/2026-06-17-slice-1g-spec.md)

## Summary

按 9 项锁定决策落地弹窗推送（show_popup command，结构化 JSON + SDK 模板）+ 双向即时聊天（PG chat_messages 表 + REST + WS 命令下行 + 访客端浮动气泡 widget + admin ChatPanel 嵌入）。31 个 e2e 全部通过。

## Changes Delivered

### server/（2 新建 / 4 修改）
- `migrations/000003_chat.up.sql` + `.down.sql`：chat_messages 表
- `internal/api/chat.go`：GET/POST /api/sessions/:id/messages
- `internal/proto/envelope.go`：加 CommandPopup + CommandChatMessage
- `internal/api/command.go`：buildCommandPayload 支持 show_popup
- `internal/api/router.go`：注册 chat 路由
- `internal/storage/queries.go`：ChatMessage struct + CreateChatMessage + ListChatMessagesBySession

### visitor-sdk/（3 新建 / 3 修改）
- `src/ui/popup.ts`：弹窗渲染（textContent 防 XSS）
- `src/ui/chatWidget.ts`：右下角浮动气泡 + 展开 + 消息列表 + 输入框
- `src/proto/command.ts`：加 show_popup + chat_message 类型
- `src/commands/handler.ts`：处理 show_popup + chat_message + onChatMessage 回调
- `src/index.ts`：启动 ChatWidget + onChatMessage → receiveMessage + onSend → POST messages

### admin/（1 新建 / 2 修改）
- `src/components/ChatPanel.vue`：聊天面板（2s 轮询拉取 + 发送）
- `src/components/VisitorPanel.vue`：嵌入 ChatPanel + 弹窗推送表单
- `src/api/sessions.ts`：加 listMessages + sendMessage + CommandType 扩展

### e2e/
- 4 个 1g 场景

## Verification

```
31 passed (1.8m)
```

**1g 验收 4 场景**：
- ✅ 弹窗推送 + 访客端渲染（#__mm_popup__ 可见）
- ✅ 双向聊天端到端（admin POST → visitor WS 接收）
- ✅ 消息历史持久化（GET 返回全部消息）
- ✅ 离线消息不丢（访客关闭后 POST 仍写 PG + GET 查到）

## v1 切片进度

| 切片 | 状态 |
|---|---|
| 1a 仓库骨架 | ✅ |
| 1b 单向最小 | ✅ |
| 1c rrweb 接入 | ✅ |
| 1d 录像归档 | ✅ |
| 1e 双向通道 | ✅ |
| 1f 表单 + 跳转 | ✅ |
| 1g 弹窗 + 聊天 | ✅ |
| 1h 认证 + 多运营 | ⏳ |
| 1i 反爬虫 | ⏳ |
| 1j i18n + 部署 + CI | ⏳ |

**v1 切片已完成 70%**（7/10）。剩余 3 个切片（认证、反爬虫、部署+CI）估时约 3-4 周 solo 全职。
