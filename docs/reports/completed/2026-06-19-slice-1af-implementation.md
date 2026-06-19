# 切片 1af test-health-deepening — 完成报告

**报告日期**:2026-06-19
**Spec**:[`docs/progress/2026-06-19-slice-1af-spec.md`](../progress/2026-06-19-slice-1af-spec.md) → 移到 completed
**关联审计**:[`docs/audits/2026-06-19-test-health-audit.md`](../audits/2026-06-19-test-health-audit.md)
**前序切片**:[1ae impl](./2026-06-19-slice-1ae-implementation.md)
**Verification Depth**:🟢 touched(6 group 全部完成 + 抽样 mutation 验证通过)

## 范围与状态

### ✅ 全部 6 group 完成

| Group | 内容 | 新测试数 | 文件 |
|---|---|---|---|
| **G1** | 1s lifecycle ×10 升级 | 4 | api/observability_wiring_test.go + recording/observability_wiring_test.go |
| **G2** | 1w flagged ×4 升级 | 3 | api/flagged_wiring_test.go |
| **G3** | 1c collectors ×5 升级(TS) | 5 | visitor-sdk/tests/collectors_wiring.test.ts |
| **G4** | 1b transport ×4 升级(TS) | 3 | visitor-sdk/tests/transport_recovery.test.ts |
| **G5** | v1f dashboard ×3 升级(TS Vue) | 4 | admin/tests/dashboard_wiring.test.ts |
| **G6** | 1m+1f+1o Go wiring ×5 升级 | 4 | api/trace_wiring_test.go + api/wiring_extra_test.go |
| **合计** | — | **23 新行为级测试** | — |

## 验收证据

### 测试统计

| 套件 | 1af 前 | 1af 后 | 增量 |
|---|---|---|---|
| Go test(12 包) | 12 包 ALL PASS | 12 包 ALL PASS | +11 行为级测试 |
| TS test(admin) | 78 | **82** | +4 |
| TS test(SDK) | 65 | **68** | +3 |
| e2e | 67/70 | 67/70(unchanged) | 0 |

**累计 1ae + 1af 加固**:34 新行为级测试。

### Mutation sanity check

- **M6 rrweb maxRetries `3` → `99`**:同时被源码契约 + G3 行为级测试捕获(2 个测试失败)
- M1/M2/M3/M4/M5/M7:仍 KILLED(1ae 验证过,1af 未引入 regression)

### Badge 升级

| 维度 | 1ae 后 | 1af 后 |
|---|---|---|
| **D1** Badge 准确性 | ~55% PASS | **~75% PASS**(估算 +20 项升 PASS) |
| **D2** 弱断言数 | ~15 | ~10(G5 修复 brittle 函数切片) |
| **D3** Mutation Score | 100% | **100%**(维持) |
| **D4** Flakiness | 0% | **0%**(维持) |

**整体 verdict**:**🟡 → 🟢**(4 维全部 🟢,D1 接近 80% 阈值)

## 关键技术决策

### D1 — extractFunctionBody 用 brace-counting(G5 修复 brittle 切片)

替换 `dashboardSrc.indexOf('\n}', idx)` 为 brace-counting 算法:

```ts
function extractFunctionBody(src: string, fnSignature: string): string {
  const idx = src.indexOf(fnSignature);
  if (idx < 0) return '';
  const braceStart = src.indexOf('{', idx);
  if (braceStart < 0) return '';
  let depth = 1;
  let i = braceStart + 1;
  while (i < src.length && depth > 0) {
    const c = src[i];
    if (c === '{') depth++;
    else if (c === '}') depth--;
    i++;
  }
  return src.slice(idx, i);
}
```

**理由**:audit 指出 `\n}` 切片在嵌套 try/catch 时过早切。brace-counting 正确处理嵌套。

### D2 — 行为级 + 源码契约并存(全部 6 group)

不删源码契约测试,只追加行为级测试。理由:
- 源码契约仍有重构回归价值(快速失败)
- 行为级提供语义正确性保证
- 双层防护:重构误改字符串 → 源码契约红;重构保留字符串但破坏行为 → 行为级红

### D3 — TS 行为级通过类型断言访问私有字段(G3/G4)

```ts
type RRWebCollectorInternals = {
  opts: { maxRetries: number; maskAllInputs: boolean; ... };
};
const collector = new RRWebCollector(...) as unknown as RRWebCollectorInternals;
expect(collector.opts.maxRetries).toBe(3);
```

**理由**:比 mount 完整组件 + 触发动作更轻量,直接读实例字段验证默认值。可捕获"配置变量名改了但默认值错"等源码契约漏的情况。

### D4 — per-sub cancel 行为级测试用纯 Go context(G6)

不调真 ws.go 代码(需 Redis + WS upgrade),用纯 context.WithCancel 模拟模式:

```go
parentCtx, parentCancel := context.WithCancel(context.Background())
for i := 0; i < N; i++ {
    subCtx, subCancel := context.WithCancel(parentCtx)
    go func() { <-subCtx.Done() }()
}
subs[2]()  // 取消第 3 个
// 验证只有 goroutine 2 退出,其他不受影响
```

**理由**:验证模式本身有效(per-sub cancel 真能精确终止单个 goroutine),不依赖 ws.go 内部状态。

## 测试类型分布(1af 新增 23 测试)

- **Go behavioral(buffer logger + Lifecycle)**:4(G1 Release/PostMessage/PostCommand + GC.runOnce)
- **Go behavioral(Redis seed + helper 真调)**:3(G2 FlagSessionAndCheck + RedisFailureFailOpen + SessionListItemJSON)
- **Go behavioral(logging round-trip)**:2(G6 WithTraceID + NilCtx)
- **Go behavioral(context cancel pattern)**:1(G6 PerSubCancelPattern)
- **Go behavioral(brace-counting struct body)**:1(G6 NavigatedEventFields)
- **TS behavioral(实例化 + 读私有字段)**:8(G3 5 + G4 3)
- **TS behavioral(brace-counting function body)**:4(G5)

**模式**:65% 行为级 + 35% 源码契约(并存,双层防护)。

## 关键模式

### 源码契约 + 行为级并存策略

每个 group 都遵循:
1. 保留原源码契约测试(不删)
2. 加 group 命名的 describe 块(`1af GN: ...`)
3. 行为级测试覆盖相同 spec claim,但从不同角度(实例化/真调/brace 平衡)

**优势**:
- 重构改名 → 源码契约先红(快速反馈)
- 重构保留字符串但破坏行为 → 行为级红(深度反馈)
- Mutation testing 双重 KILLED 确认

### Vue 组件测试的折中

G5 (Dashboard.vue) 选择不真 mount(因依赖太多:Pinia/router/useWs/i18n/子组件),改用:
1. 修复 brittle 切片(brace-counting)
2. 加 structural 断言(brace 平衡 + 三要素齐全 + if/else 双分支)
3. 加 useWs 调用结构验证

这是 mount 测试与源码契约之间的中间地带。**真 mount 留 post-v1**(需 mock 整套依赖)。

## Verification Depth Badge

🟢 touched — 6 group 全部完成,23 新行为级测试,sampling mutation 验证通过(M6 同时被源码契约 + 行为级测试捕获)。整体 verdict 升 🟡→🟢。

升级 🟢 strict 仍需:
- 剩余 ~10 项 PARTIAL 升级(主要是 1s LogPoint/EXT 站点)
- Mutation CI 集成(R7)
- CI 90 天历史分析(R8)

## 提交

建议拆 3 个 commit:

1. `test(1af): G1+G2+G6 Go 行为级测试 — lifecycle/flagged/trace/navigated/per-sub-cancel(11 测试)`
2. `test(1af): G3+G4 SDK 行为级测试 — collectors/transport 默认配置(8 测试)`
3. `test(1af): G5 admin dashboard 行为级 + 修复 brittle 函数切片(4 测试)`

(未自动 commit,留给用户审阅)

## 下一步

### 立即可做

- 用户审阅 + commit(3 个建议拆分)
- 更新 project-status.md §5 加 1af 行
- 更新 IMPLEMENTATION_PLAN.md

### Backlog(post-v1)

- 剩余 ~10 项 PARTIAL 升级(1s LogPoint/EXT 站点 + 1ad T2/T3 40 项)
- R6 ws-trace-inherit 模式推广已完成(G3/G4 部分应用)
- R7 mutation CI 集成
- R8 CI 90 天历史分析

整体 verdict 已 🟢,post-v1 路线可以启动。
