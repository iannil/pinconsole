# 切片 1aj followup-bugs — Spec

**切片编号**:1aj
**类型**:bug fix(1ag/1ah follow-up)
**创建时间**:2026-06-19
**状态**:approved
**关联**:[1ag impl](../completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md)、[1ah impl](../completed/2026-06-19-slice-1ah-claim-chat-handler-tests-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:1ag/1ah 实施过程发现两个 pre-existing 问题:
  1. `parseSince("-1d")` 接受负数 — `Atoi("-1")` 不报错,产生 `-24h` 负 duration,语义错误(查询"结束于 -24h 内"无意义)
  2. `TestCheckWSRateLimit_OverMsgCount` flaky — 1ag 实施时 `go test ./...` 偶发失败,跑 10+ 次单独无法复现
- **业务/技术价值**:
  - parseSince 负数:有 SQL 注入语义歧义风险(`since=-1d` 可能被滥用),且 admin UI 不会传负值但 API 不拒,违反"输入校验在边界"
  - flaky test:CI 信心受损,违反"测试稳定"原则;虽然罕见但仍会偶发误报
- **不做的代价**:post-v1 CI 跑全套时,parseSince 负数可能被 LLM 误用为绕过分页;flaky 持续累积导致 CI 信号不可信

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | parseSince 负数修复 | A: 在 case 分支内单独校验 / B: 在 Atoi 后统一校验 / C: 用绝对值 | **B** | A 重复代码;C 隐式修正更危险(用户传 -1d 想"过去 1 天",abs 给出 +1d 看似对但绕过了"校验"语义);B 单点校验最干净 |
| 2 | parseSince 零处理 | A: 拒绝(num<=0) / B: 接受 0(零 duration) / C: 仅拒负 | **A** | "0h" 语义模糊(是默认还是真零),拒零更安全;调用方应传 "" 而非 "0h" |
| 3 | flaky 修复策略 | A: err 转 t.Skip(承认 Redis 不稳定) / B: 增加 wait/retry / C: 用 miniredis 替代真 Redis | **A** | B 治标不治本(Redis 慢时仍会失败);C 引入新依赖且 miniredis 不支持全部 Lua;A 最诚实——把"环境问题"和"代码问题"分开 |
| 4 | flaky 修复范围 | A: 仅 OverMsgCount / B: 全 4 个 ws_ratelimit 测试 / C: 全包 Redis 测试 | **B** | 4 个 ws_ratelimit 测试有相同的 fail-open 模式;A 治标(其他 3 个仍可能 flaky);C 范围太大超 budget |
| 5 | parseSince 修复后测试 | A: 仅加负数拒绝测试 / B: 加负数 + 零拒绝测试 | **B** | A 漏零路径;B 覆盖 num<=0 全部边界 |

## Acceptance(可验证的成功标准)

### 必须满足

- [ ] **A1**:`parseSince` 拒绝 `num <= 0`,返回 error
- [ ] **A2**:新增测试 `TestParseSince_RejectsNonPositive` 覆盖 `-1d`、`-12h`、`0h`、`0d` → error
- [ ] **A3**:`TestCheckWSRateLimit_OverMsgCount`/`OverBytes`/`SessionIsolation` 在 Redis err 时 t.Skip(而非 t.Fatal)
- [ ] **A4**:`TestCheckWSRateLimit_NormalTrafficAllows` 同样处理 err
- [ ] **A5**:跑 `go test ./internal/api/ -count=20` 全绿或全 skip,无 Fatal

### 验证维度

- [ ] `go test ./...` 全绿
- [ ] parseSince 行为变化不破坏 `TestListEndedSessions_ValidSincePassesParsing`(7d/12h/24h 仍合法)
- [ ] `parseSince("7d")` 仍返回 7×24h(无回归)
- [ ] ws_ratelimit 测试模式不影响测试在 Redis 可用时仍 PASS(全跑 1 次确认)

### 不在本切片范围

- 用 miniredis 替代真 Redis(留 backlog)
- parseSince 支持 `w`(周)等新单位 — 功能扩展,非 bug 修复
- 其他包的 flaky 测试(若发现)

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 15min |
| parseSince 修复 + 测试 | 30min |
| ws_ratelimit 4 测试 err 处理 | 30min |
| 跑 20x 验证 + 报告 | 30min |
| **合计** | **~1.5h** |

## 完成后预期

- `parseSince("-1d")` 等 4 个边界用例正确拒绝
- ws_ratelimit 测试在 Redis 偶发 hiccup 时优雅 skip 而非误报 fail
- 1ag follow-up bugs 全部关闭
- api 包覆盖率:29.1% → ~30%(新加 non-positive 测试)

## Verification Depth Badge 目标

🟢 touched — 修复 + 新测试 + 20x 验证稳定。

## 关联

- 前置:1ag、1ah
- 触发:1ag 实施过程发现
- 后续:1ai(storage 接口重构,解锁 happy path)
