# 切片 1ae test-health-fix — Spec

**切片编号**：1ae
**类型**：测试质量加固
**创建时间**：2026-06-19
**状态**：approved
**关联**：[test-health-audit](../audits/2026-06-19-test-health-audit.md)、[test-confidence-audit](../audits/2026-06-19-test-confidence-audit.md)、[1ac impl](../reports/completed/2026-06-19-slice-1ac-implementation.md)、[1ad impl](../reports/completed/2026-06-19-slice-1ad-implementation.md)

## Context

为什么做这个切片？

- **触发原因**：2026-06-19 test-health-audit 整体 verdict 🔴。D1 PASS 率仅 38.2%,D3 mutation score 71.4%,D4 e2e flakiness 25%。审计识别 2 个 P0(安全/合规) + 3 个 P1(质量)项。
- **业务/技术价值**:让 v1 测试套件从"看起来测了"升级到"真的能捕获重构回归",为 post-v1(自定义域名/页面编辑器/Tauri)refactor 提供信心底座。
- **不做的代价**:post-v1 任何 refactor 都可能 silently 破坏 auth/GDPR/安全路径而测试不告警。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | R2 erasure 隔离策略 | A: 测试 schema 无 CASCADE / B: mock PG pool 断言 SQL / C: 两者都做 | **B** | A 需新建测试 schema + 维护迁移,成本高;B 用 pgxmock 断言每条 Exec 调用,直接验证函数逻辑;C 过度工程 |
| 2 | R3 升级范围 | A: 全部源码契约测试 / B: 仅 5 个高风险切片 / C: 仅 P0 闭包 | **B** | A 工时 ~30h 超 slice budget;C 漏 P1 重要路径;B 是审计建议的精确范围,~5-8h |
| 3 | R4 e2e flaky 修复 | A: 加测试 setup truncate / B: 增加 wait / C: 用 unique tenant 隔离 | **A** | 先调试根因,如果是数据累积 → truncate;如果是 timing → wait;如果是 sharing → unique tenant。在调试后定 |
| 4 | Mutation CI 集成 | A: 本切片加 stryker / mutesting / B: 留 post-v1 backlog | **B** | 一次性 setup 成本高,且本项目 test 套件稳定后再集成更合理。本次只做手动 mutation 验证 |
| 5 | Badge 升级目标 | A: 全部升 🟢 strict / B: 升到 🟡 verified-shallow 以上 / C: 关键路径升 🟢 touched | **C** | strict 要求"所有 spec 决策点 + mutation CI",超本切片范围;touched 要求"切片目标段有测试 + 断言强度通过 mutation",可达成 |
| 6 | 是否修弱断言 D2 | A: 修高危 7 处 / B: 修中危 9 处 / C: 留 backlog | **C** | D2 verdict 已 🟢(20 处刚好阈值内),修复 ROI 低;高危 7 处大多与 R3 重叠(如 flusher_wiring 同源) |

## Acceptance（可验证的成功标准）

切片"完成"的可验证判据。每条都用客观断言。

### 必须满足(P0 + P1 修复)

- [ ] **R1**:加 `TestOperatorWS_NoCookie_Returns401_Behavioral` 到 `ws_auth_test.go`,真调 handler + 断言 `status=401` + 断言无 upgrade。验证:M4 mutation(把 auth 调用包进 `if false`)重新应用后**测试必失败**。
- [ ] **R2**:加 `TestDeleteVisitorByFingerprint_ExecutesExplicitDeletes` 到 `erasure_test.go`,用 pgxmock 断言每条 DELETE 被调用。验证:M3 mutation(跳过 chat_messages delete)重新应用后**测试必失败**。
- [ ] **R3a**:`TestLogin_UsesBcryptCompareHashAndPassword` 加真 handler 调用,wrong password → 401 + cookie cleared。
- [ ] **R3b**:`TestAuthCookie_HttpOnly_AlwaysTrue` + `SecureFlag_Threading` 改用 httptest recorder + 真 `Set-Cookie` header 断言。
- [ ] **R3c**:`TestMigration_FailFastOnMigrationError` 改用 failing migration 真跑 + 断言错误传播。
- [ ] **R3d**:`flusher_wiring_test.go` 3 个测试改用 mock MinIO/PG/Redis + 调用断言。
- [ ] **R3e**:`observability_wiring_test.go` ≥3 个 lifecycle 接线点改用真 ctx + logger + 断言返回值消费。
- [ ] **R4**:e2e `1d-replay.spec.ts:71` 跑 10 次,**0 次失败**(0% flakiness)。
- [ ] **R5**:`session-expired.test.ts:71-89` 改为 mount 真 LoginView + i18n spy,断言 `t('login.error_session_expired')` 被调。

### 验证维度

- [ ] `go test ./...` 全绿(预计 +5-10 新测试)
- [ ] `pnpm test:js` 全绿(预计 +2-3 新测试)
- [ ] **重跑 D3 mutation M3+M4**:两个之前 survivor 现在 **KILLED**
- [ ] **重跑 D3 mutation M1+M2+M5+M6+M7**:仍 KILLED(无 regression)

### 不在本切片范围

- T2/T3 测试 gap(40 项,见 1ad 报告)— 留 backlog
- D2 中危/低危弱断言修复 — 留 backlog
- Mutation CI 集成(stryker/mutesting)— 留 post-v1
- CI 90 天历史分析(需 gh auth) — 留 post-v1
- Badge 升级到 🟢 strict(需要 mutation CI) — 留 post-v1

## 工时预算

| 项 | 工时 |
|---|---|
| R5 T0-1h-ui-3 修复 | 30min |
| R1 operatorWS 行为级测试 | 1h |
| R2 erasure mock 测试 | 2h |
| R3a-e 5 个测试升级 | 5-6h |
| R4 e2e flaky 调试 + 修复 | 2-3h |
| 验收 + mutation 重跑 + 报告 | 1h |
| **合计** | **~11-13h** |

## 完成后预期

- D1 PASS 率:38.2% → ≥65%(R3 5 项 + R1 + R2 = 7 项升 PASS,可能附带 5+ 项连带升级)
- D3 Mutation Score:71.4% → ≥90%(M3+M4 修复后,7/7 killed)
- D4 Flakiness:20% → 0%(R4 修复后)
- 整体 verdict:🔴 → 🟡(若 D2/D3/D4 全过 + D1 ≥80% 则 🟢)
- 1ac/1ad 关键闭包 badge:🟢 touched → 🟢 strict(对 R1-R3 涉及的切片)

## Verification Depth Badge 目标

🟢 touched(本切片目标:R1-R5 完成 + mutation 验证通过)

升级 🟢 strict 需 post-v1 加 mutation CI。
