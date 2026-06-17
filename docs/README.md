# docs/ 文档索引

> 本目录是项目的文档中心。所有非"事实来源"的工作文档（进度、报告、审计、规范、模板）都在此。
> 事实来源文档（[`CLAUDE.md`](../CLAUDE.md) / [`PLAN.md`](../PLAN.md) / [`START.md`](../START.md)）保留在仓库根。

## 新读者从这里开始

1. [`project-status.md`](./project-status.md) — **必读**。项目当前状态、架构决策清单、下一步动作、LLM 协作提示
2. [`../CLAUDE.md`](../CLAUDE.md) — Claude 工作指南（含文档/记忆/可观测性/LLM Friendly 约定）
3. [`../PLAN.md`](../PLAN.md) — v1 切片架构事实来源
4. [`../START.md`](../START.md) — 产品需求与竞品分析

## 目录结构

| 路径 | 用途 | 当前内容 |
|---|---|---|
| [`project-status.md`](./project-status.md) | rolling 项目状态（LLM 友好） | 2026-06-17 初始版 |
| [`progress/`](./progress/) | 进行中或已开始的修改单元（一改一文） | [`2026-06-17-v1-slice-plan.md`](./progress/2026-06-17-v1-slice-plan.md)、[`2026-06-17-slice-1a-spec.md`](./progress/2026-06-17-slice-1a-spec.md)、[`2026-06-17-slice-1a-implementation.md`](./progress/2026-06-17-slice-1a-implementation.md)、[`2026-06-17-slice-1b-spec.md`](./progress/2026-06-17-slice-1b-spec.md) |
| [`reports/completed/`](./reports/completed/) | 已完成的修改报告 + 验收证据 | [`2026-06-17-v1-slice-plan.md`](./reports/completed/2026-06-17-v1-slice-plan.md)、[`2026-06-17-slice-1a-implementation.md`](./reports/completed/2026-06-17-slice-1a-implementation.md) |
| [`audits/`](./audits/) | 审计发现（冗余/过期/错误梳理） | [`2026-06-17-initial-cleanup.md`](./audits/2026-06-17-initial-cleanup.md) |
| [`standards/`](./standards/) | 规范（命名、结构、流程） | [`doc-structure.md`](./standards/doc-structure.md)、[`naming-conventions.md`](./standards/naming-conventions.md) |
| [`templates/`](./templates/) | 各类文档模板 | [`progress.md`](./templates/progress.md)、[`report.md`](./templates/report.md) |

## 文档生命周期

```
开始修改 → 写 docs/progress/{date}-{name}.md
   ↓ 完成且通过验收
写 docs/reports/completed/{date}-{name}.md （与 progress 同名）
   ↓
更新 docs/project-status.md
```

详见 [`standards/doc-structure.md`](./standards/doc-structure.md) §4。

## 修改文档时的清单

- [ ] 是否使用了对应模板？见 [`templates/`](./templates/)
- [ ] 文件名是否符合规范？见 [`standards/naming-conventions.md`](./standards/naming-conventions.md)
- [ ] 是否在 `progress/` 记录进展？
- [ ] 完成后是否移到 `reports/completed/`？
- [ ] 是否更新 `project-status.md` 的相关状态字段？
- [ ] 是否在 `memory/daily/{date}.md` 追加今日工作记录？
