# 切片 1h 规格说明（认证 + 多运营）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7
**前置**：[切片 1g 完成](./2026-06-17-slice-1g-implementation.md)

## 7 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | Session 存储 | Redis（TTL 24h） |
| 2 | 用户初始化 | 启动时 env var ADMIN_EMAIL + ADMIN_PASSWORD |
| 3 | Claim 锁存储 | Redis key claim:session:{sid}，TTL 5 分钟 |
| 4 | 认证范围 | 运营端点全保护，访客端点公开 |
| 5 | 登录 UI | /admin/login + Vue Router 守卫 |
| 6 | Claim UX | Claim/Release 按钮 + 状态显示 |
| 7 | 1h 验收 | 4 项 |

## 范围

- [ ] PG users 表 + bcrypt
- [ ] POST /api/auth/login + /api/auth/logout + GET /api/auth/me
- [ ] AuthMiddleware（cookie → Redis session → user_id）
- [ ] 运营端点全保护（admin SPA + REST + WS operator）
- [ ] 启动时初始化默认 admin（env var）
- [ ] admin /admin/login 页 + 路由守卫
- [ ] Claim/Release 按钮 + Redis 锁
- [ ] 4 验收场景

## 估时

Solo 全职 5-7 天。
