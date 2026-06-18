# IMPLEMENTATION_PLAN — 当前正在做什么

> CLAUDE.md 第 39 行要求:通过 IMPLEMENTATION_PLAN.md 让模型理解当前正在做什么、边界、下一步。
> 本文件 rolling 更新,每次开始新切片时改写;完成后归档到 `docs/reports/completed/`。

**当前状态**:v1 主干已交付,正在做生产硬化 + LLM friendly 收尾
**最后更新**:2026-06-18

## 当前焦点

无活跃切片(刚完成 1p-llm-friendly)。下一步候选见 docs/project-status.md §7。

## v1 已交付切片(深度 + 顺序)

| 切片 | 内容 | 深度 |
|---|---|---|
| 1a-1j | 原始 v1 切片 | 🟡/🟢/🔴 混合,见 project-status §5 |
| 1k | 安全阻断栈 | 🟢 |
| 1l | GDPR 合规 | 🟢 |
| 1h-ui | admin LoginView + 守卫 | 🟢 |
| 1m | 可观测性 | 🟢 |
| 1n | 测试深度 + 文档虚标 | 🟢 |
| 1o | 生产硬化 | 🟢 |
| 1p | LLM friendly(proto 共享 + IMPLEMENTATION_PLAN + change-safety) | 🟢 |

## 下一步候选(按优先级)

1. **死代码 + 重复清理**(P1):e2e/helpers 整目录、Element Plus 注册零使用、queries.sql vs queries.go、6+ deprecated 函数
2. **代码质量**(P1/P2):5 Go 包零单测 + god files 拆分
3. **i18n + logger 迁移**(P1):admin/utils/time.ts + SDK handler/chatWidget + 27 处 console.*
4. **测试场景补全**(P2):1b SDK 重连 + 1e/1g cursor/ESC + 1c canvas/WebGL(需 testcontainers)

## 决策原则

- 任何架构层工作先读 PLAN.md(架构事实来源)
- 任何产品层冲突以 START.md 为准
- 切片流程:grill-me → spec → impl → 验收 → spec+impl 移到 docs/reports/completed/
- 范围控制严格:不做多租户、不做计费、不做注册流(CLAUDE.md 硬约束)
- 安全/反爬虫/GDPR 是一等公民(START.md 明确)
- 测试深度判定遵循 docs/standards/verification-depth.md(R2 rubric)

## 历史切片(已归档)

详见 [`docs/reports/completed/`](./docs/reports/completed/) — 每切片 spec + implementation 两文件 + v1-slice-plan 总览。
