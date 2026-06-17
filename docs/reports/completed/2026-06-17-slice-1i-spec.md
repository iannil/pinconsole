# 切片 1i 规格说明（反爬虫）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**规格来源**：本文件
**前置**：[切片 1h 完成](./2026-06-17-slice-1h-implementation.md)

## 6 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | Rate limit | Redis 固定窗口 |
| 2 | RL 范围 | 仅公开端点 |
| 3 | 行为分析 | 服务端启发式标记 |
| 4 | Fingerprint | SDK 启动采集 → hello → visitors.meta |
| 5 | UA 黑名单 | 内置 + env var |
| 6 | 验收 | 4 项 |

## 估时

Solo 全职 3-5 天。
