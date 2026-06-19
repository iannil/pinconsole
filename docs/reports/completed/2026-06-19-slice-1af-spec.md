# 切片 1af test-health-deepening — Spec

**切片编号**：1af
**类型**：测试质量加固（1ae 续做）
**创建时间**：2026-06-19
**状态**：approved
**关联**：[test-health-audit](../audits/2026-06-19-test-health-audit.md)、[1ae impl](../reports/completed/2026-06-19-slice-1ae-implementation.md)

## Context

为什么做这个切片？

- **触发原因**：1ae 关闭 9 项 P0+P1 后，D1 PASS 率从 38.2% 升到 ~55%。剩余 ~30 项源码契约 PARTIAL 测试仍是整体 verdict 卡在 🟡 的主因。审计 §8 backlog 明确列了这些为"留 post-v1"，但用户选择现在做。
- **业务/技术价值**：把 D1 PASS 率推到 ≥80%，整体 verdict 升 🟡→🟢。post-v1 任何 refactor 都有更强信心底座。
- **不做的代价**：post-v1 (自定义域名/页面编辑器/Tauri) refactor 时，源码契约测试不能捕获语义错误，可能 silently 破坏功能。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 全 30 项 / B: 高风险 15 项 / C: 全 30 项分批 + check-in | **C** | 15h 工时大，分批可中间调整。每批 ~2-3h，3-4 批完成 |
| 2 | 升级策略 | A: 全部行为级（删源码契约）/ B: 行为级 + 保留源码契约 / C: 选择性 | **B** | 源码契约仍有重构回归价值，保留作为 additional check；行为级作为主测 |
| 3 | TS 测试模式 | A: 复用 ws-trace-inherit mock 模式 / B: 用 vitest inline mock | **A** | R6 推广（audit §7.6 推荐），统一模式降低维护成本 |
| 4 | Vue 组件测试 | A: mount + i18n spy / B: shallow mount + stub store / C: 完整集成 | **A** | R5 已验证此模式（LoginView），可复用 |
| 5 | Mutation 验证范围 | A: 全部新测试 / B: 抽样高风险 5 项 / C: 不验证（1ae 已验过模式） | **B** | 全量验证太耗时，模式相同可信；抽样 5 项确认无 regression |

## Acceptance（可验证的成功标准）

### 必须满足（按 group）

- [ ] **G1 1s lifecycle ×10**：每个原 grep 测试都有对应行为级测试（buffer logger 验证 Lifecycle 日志产出）
- [ ] **G2 1w flagged ×4**：每个原 grep 测试都有对应行为级测试（真 Redis seed + helper 调用）
- [ ] **G3 1c collectors ×5**：每个原 TS 源码契约测试都有真实例化 + 字段断言
- [ ] **G4 1b transport ×4**：每个原 TS 源码契约测试都有 mock WS + 行为断言
- [ ] **G5 v1f dashboard ×3**：修复 brittle `\n}` 函数体切割，改用 AST 或真 mount + spy
- [ ] **G6 1m+1f+1o ×5**：每个原 Go 源码契约测试都有行为级补充

### 验证维度

- [ ] `go test ./...` 全绿（预计 +15-20 新测试）
- [ ] `pnpm test:js` 全绿（预计 +10-15 新测试）
- [ ] D1 PASS 率：~55% → **≥80%**
- [ ] 整体 verdict：🟡 → 🟢

### 不在本切片范围

- T2/T3 测试 gap（40 项，见 1ad 报告）— 留 backlog
- Mutation CI 集成 — 留 post-v1
- CI 90 天历史分析（需 gh auth） — 留 post-v1

## 工时预算

| 项 | 工时 |
|---|---|
| G1 1s lifecycle ×10 | 3h |
| G2 1w flagged ×4 | 1.5h |
| G3 1c collectors ×5 (TS) | 2h |
| G4 1b transport ×4 (TS) | 2h |
| G5 v1f dashboard ×3 (TS Vue) | 1.5h |
| G6 1m+1f+1o ×5 | 2h |
| 验收 + 抽样 mutation + 报告 | 1h |
| **合计** | **~13h** |

## 完成后预期

- D1 PASS 率：~55% → ≥80%
- D2 弱断言：~15 → ~10（升级附带清理）
- D3 Mutation Score：100%（维持）
- D4 Flakiness：0%（维持）
- 整体 verdict：🟡 → 🟢

## Verification Depth Badge 目标

🟢 touched（全部 group 完成 + 抽样 mutation 验证通过）

升 🟢 strict 仍需 post-v1 加 mutation CI（R7）。
