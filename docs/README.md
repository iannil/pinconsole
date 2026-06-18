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
| [`project-status.md`](./project-status.md) | rolling 项目状态（LLM 友好） | v1 主干完全收口版本（2026-06-18） |
| [`progress/`](./progress/) | 当前正在进行的修改单元（一改一文） | 空（无在进行的修改） |
| [`reports/completed/`](./reports/completed/) | 已完成的切片 spec + implementation 报告 | v1 全切片 1a-1z + v1-e2e + v1-followups 共 30+ 报告 + [`v1-slice-plan.md`](./reports/completed/2026-06-17-v1-slice-plan.md) |
| [`audits/`](./audits/) | 审计发现（冗余/过期/错误梳理） | [`initial-cleanup`](./audits/2026-06-17-initial-cleanup.md) / [`deep-audit (80 findings)`](./audits/2026-06-18-deep-audit.md) / [`1k-1u-regression`](./audits/2026-06-18-1k-1u-regression.md) |
| [`standards/`](./standards/) | 规范（命名、结构、流程、验证深度、变更安全） | [`doc-structure.md`](./standards/doc-structure.md) / [`naming-conventions.md`](./standards/naming-conventions.md) / [`verification-depth.md`](./standards/verification-depth.md) / [`change-safety.md`](./standards/change-safety.md) |
| [`templates/`](./templates/) | 各类文档模板 | [`progress.md`](./templates/progress.md) / [`report.md`](./templates/report.md) / [`audit.md`](./templates/audit.md) / [`spec.md`](./templates/spec.md) |

## 切片报告索引（docs/reports/completed/）

v1 切片按顺序排列，每片含 spec（事前决策）+ implementation（事后总结）:

| 切片 | spec | implementation | 深度 |
|---|---|---|---|
| 1a 骨架 | [spec](./reports/completed/2026-06-17-slice-1a-spec.md) | [impl](./reports/completed/2026-06-17-slice-1a-implementation.md) | 🟢 |
| 1b 单向最小 | [spec](./reports/completed/2026-06-17-slice-1b-spec.md) | [impl](./reports/completed/2026-06-17-slice-1b-implementation.md) | 🟢 |
| 1c rrweb | [spec](./reports/completed/2026-06-17-slice-1c-spec.md) | [impl](./reports/completed/2026-06-17-slice-1c-implementation.md) | 🟢 |
| 1d 录像归档 | [spec](./reports/completed/2026-06-17-slice-1d-spec.md) | [impl](./reports/completed/2026-06-17-slice-1d-implementation.md) | 🟢 |
| 1e 双向通道 | [spec](./reports/completed/2026-06-17-slice-1e-spec.md) | [impl](./reports/completed/2026-06-17-slice-1e-implementation.md) | 🟢 |
| 1f 表单 + 跳转 | [spec](./reports/completed/2026-06-17-slice-1f-spec.md) | [impl](./reports/completed/2026-06-17-slice-1f-implementation.md) | 🟢 |
| 1g 弹窗 + 聊天 | [spec](./reports/completed/2026-06-17-slice-1g-spec.md) | [impl](./reports/completed/2026-06-17-slice-1g-implementation.md) | 🟢 |
| 1h 认证 + 多运营(后端) | [spec](./reports/completed/2026-06-17-slice-1h-spec.md) | [impl](./reports/completed/2026-06-17-slice-1h-implementation.md) | 🔴 |
| 1h-ui LoginView + 守卫 | [spec](./reports/completed/2026-06-18-slice-1h-ui-spec.md) | [impl](./reports/completed/2026-06-18-slice-1h-ui-implementation.md) | 🟢 |
| 1i 反爬虫 | [spec](./reports/completed/2026-06-17-slice-1i-spec.md) | [impl](./reports/completed/2026-06-17-slice-1i-implementation.md) | 🟢 |
| 1j i18n + 部署 + CI | [spec](./reports/completed/2026-06-17-slice-1j-spec.md) | [impl](./reports/completed/2026-06-17-slice-1j-implementation.md) | 🟢 |
| 1k 安全阻断栈 | [spec](./reports/completed/2026-06-18-slice-1k-spec.md) | [impl](./reports/completed/2026-06-18-slice-1k-implementation.md) | 🟢 |
| 1l GDPR 合规 | [spec](./reports/completed/2026-06-18-slice-1l-spec.md) | [impl](./reports/completed/2026-06-18-slice-1l-implementation.md) | 🟢 |
| 1m 可观测性 | [spec](./reports/completed/2026-06-18-slice-1m-spec.md) | [impl](./reports/completed/2026-06-18-slice-1m-implementation.md) | 🟢 |
| 1n 测试深度 + 文档虚标 | — | [impl](./reports/completed/2026-06-18-slice-1n-implementation.md) | 🟢 |
| 1o 生产硬化 | — | [impl](./reports/completed/2026-06-18-slice-1o-implementation.md) | 🟢 |
| 1p LLM friendly | — | [impl](./reports/completed/2026-06-18-slice-1p-implementation.md) | 🟢 |
| 1q 死代码 + 重复清理 | — | [impl](./reports/completed/2026-06-18-slice-1q-implementation.md) | 🟢 |
| 1r i18n + logger 迁移 | — | [impl](./reports/completed/2026-06-18-slice-1r-implementation.md) | 🟢 |
| 1s 可观测性深化 | — | [impl](./reports/completed/2026-06-18-slice-1s-implementation.md) | 🟢 |
| 1t 测试覆盖补全 | — | [impl](./reports/completed/2026-06-18-slice-1t-implementation.md) | 🟢 |
| 1u god files 拆分 | — | [impl](./reports/completed/2026-06-18-slice-1u-implementation.md) | 🟢 |
| 1v 审计后续修复 | — | [impl](./reports/completed/2026-06-18-slice-1v-implementation.md) | 🟢 |
| 1w flagged session 接入 | — | [impl](./reports/completed/2026-06-18-slice-1w-implementation.md) | 🟢 |
| 1x 登录暴力破解防护 | — | [impl](./reports/completed/2026-06-18-slice-1x-implementation.md) | 🟢 |
| 1y visitor WS rate limit | — | [impl](./reports/completed/2026-06-18-slice-1y-visitor-ws-rate-limit.md) | 🟢 |
| 1z 生产就绪度补全 | — | [impl](./reports/completed/2026-06-18-slice-1z-prod-readiness-gaps.md) | 🟢 |
| v1-e2e 全量 acceptance | — | [impl](./reports/completed/2026-06-18-v1-e2e-acceptance.md) | 🟢 |
| v1-followups e2e 后 5 fix | — | [impl](./reports/completed/2026-06-18-v1-followups.md) | 🟢 |

> **深度判定标准**:见 [`standards/verification-depth.md`](./standards/verification-depth.md)。深度 badge 含义:🟢 verified-deep / 🟡 verified-shallow / 🔴 implemented-unverified。
> **累计**：🟢 ×29 / 🔴 ×1（1h-backend spec partial）。

## 文档生命周期

```
开始修改 → 写 docs/progress/{date}-{name}.md
   ↓ 完成且通过验收
spec + implementation 一起移到 docs/reports/completed/{date}-{name}.md
   ↓
更新 docs/project-status.md
   ↓
memory/daily/{date}.md 追加今日工作记录
```

详见 [`standards/doc-structure.md`](./standards/doc-structure.md) §4。

## 修改文档时的清单

- [ ] 是否使用了对应模板？见 [`templates/`](./templates/)
- [ ] 文件名是否符合规范？见 [`standards/naming-conventions.md`](./standards/naming-conventions.md)
- [ ] 是否在 `progress/` 记录进展？
- [ ] 完成后 **spec 与 implementation 一起**移到 `reports/completed/`？
- [ ] 是否更新 `project-status.md` 的相关状态字段？
- [ ] 是否在 `memory/daily/{date}.md` 追加今日工作记录？
- [ ] 切片是否标定验证深度（🟢/🟡/🔴）？见 [`standards/verification-depth.md`](./standards/verification-depth.md)
