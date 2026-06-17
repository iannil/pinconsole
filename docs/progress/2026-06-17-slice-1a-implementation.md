# 切片 1a 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1a-spec.md`](./2026-06-17-slice-1a-spec.md)
**关联**：[`PLAN.md`](../../PLAN.md) §7

## Context

按已敲定的切片 1a 规格（13 项决策 + 文件级布局）落地仓库骨架。本文件记录实施过程的关键变更与偏差。

## Changes

按规格 §仓库布局 的文件清单逐项落地。详细变更日志：

- [ ] 顶层 workspace 文件（Makefile / package.json / pnpm-workspace.yaml / .env.example / .gitignore / .editorconfig）
- [ ] server/ Go 骨架（go.mod / cmd/ / internal/ / embed.go / Dockerfile / .air.toml / .golangci.yml）
- [ ] admin/ Vue3 骨架（package.json / vite.config.ts / src/）
- [ ] visitor-sdk/ 骨架（package.json / vite.config.ts library mode / playground/）
- [ ] landing/demo/index.html 静态页
- [ ] e2e/ Playwright smoke
- [ ] docker-compose.yml（含 dev / prod profiles）
- [ ] linting 配置全套
- [ ] GitHub Actions（ci.yml + release.yml）
- [ ] 依赖安装与构建验证

## Status

进行中。已确认工具链：Go 1.26.2 / Node 24.14 / pnpm 10.32 / Docker 29.4 / compose v5.1.2。

缺失的本地工具（实施时文档化或安装）：air / golangci-lint / golang-migrate / overmind。

## 与规格的偏差

记录实施过程中对规格的调整：

- **`concurrently`（npm）替代 `overmind`**：避免 tmux 依赖，跨平台更友好。Makefile 中 `make dev` 改用 `pnpm dev`（concurrently 管理 3 进程）。
- 其余决策按规格执行。

## Next

按任务清单推进。完成后写完成报告 [`docs/reports/completed/2026-06-17-slice-1a-implementation.md`](../reports/completed/2026-06-17-slice-1a-implementation.md)。

## Blockers

无。

## Notes

实施期间遵循：
- 不修改 PLAN.md / START.md（事实来源）
- 不修改 CLAUDE.md（用户域）
- 任何架构偏差必须在此文件"与规格的偏差"小节记录
- 所有文件使用 LF 换行、UTF-8 无 BOM
- Go 代码英文 + 中文注释（业务术语保留中文）
- TS/Vue 代码英文 + 中文注释
