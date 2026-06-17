# 切片 1g 规格说明（弹窗 + 聊天）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7
**前置**：[切片 1f 完成](./2026-06-17-slice-1f-implementation.md)

## Context

弹窗推送（运营→访客单向，结构化 JSON + SDK 模板）+ 双向即时聊天（PG 持久化 + 复用 WS envelope + 离线不丢）。

## 9 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | 弹窗 vs 聊天 | 分离 |
| 2 | 聊天通道 | 复用 envelope |
| 3 | 聊天存储 | PG 表 chat_messages |
| 4 | 弹窗格式 | 结构化 JSON + SDK 模板 |
| 5 | 访客端聊天 UI | 右下角浮动气泡 |
| 6 | Admin 聊天面板 | VisitorPanel 下方嵌入 |
| 7 | 消息排序 | PG 自增 id |
| 8 | 离线消息 | 写入 PG + 重连拉取 |
| 9 | 1g 验收 | 4 项 |

## 范围（acceptance criteria）

- [ ] PG 表 chat_messages
- [ ] show_popup command（结构化 JSON + SDK 模板渲染）
- [ ] 双向聊天（访客→admin event/chat_message，admin→访客 command/chat_message）
- [ ] GET /api/sessions/:id/messages + POST /api/sessions/:id/messages
- [ ] 访客端右下角浮动气泡聊天 widget
- [ ] admin ChatPanel 嵌入 VisitorPanel
- [ ] 端到端 4 场景：弹窗推送 / 双向聊天 / 消息持久化 / 离线不丢

## 估时

Solo 全职 5-7 天 / 业余 2-3 周。
