# 切片 1j 规格说明（i18n + 部署 + CI）

**状态**：in_progress
**开始**：2026-06-17
**前置**：[切片 1i 完成](./2026-06-17-slice-1i-implementation.md)
**v1 最终切片**

## 3 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | i18n | 全部 98 处硬编码 → i18n key |
| 2 | CI | ci.yml 加 migration + e2e smoke + lint |
| 3 | 验收 | i18n 全量 / CI 绿色 / Prod 启动 / README |

## 估时

Solo 全职 2-3 天。
