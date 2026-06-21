# Slice TS-3 spec:admin 纯逻辑层补测

> **状态**:completed
> **实际工时**:~1h
> **深度 badge**:🟡 touched

## 范围

| 包 | 切片前 | 切片后 |
|---|---|---|
| admin | 82.17% | **85.67%** |

| 文件 | 切片前 | 切片后 |
|---|---|---|
| src/api/sessions.ts | 1.69% | **100%** |
| src/utils/time.ts | 11.11% | **100%** |

## 主要工作

新建 2 个测试文件,共 20 测试:

### `tests/sessions_coverage.test.ts`(12 测试)
- mock apiJson,验证 sessions API client 5 个函数的 URL 拼接 + 参数序列化
- listEndedSessions:默认参数 / 自定义 since / 30d range
- getSessionReplay:默认 offset/limit / 自定义 / encodeURIComponent 特殊字符
- sendCommand:POST + body / 8 个命令类型
- listMessages:默认 sinceId / 自定义
- sendMessage:默认 sender=operator / 显式 visitor

### `tests/time_coverage.test.ts`(8 测试)
- formatRelative 5 个时间区间 + 3 个边界(59s / 59min / 23h)
- vi.useFakeTimers + setSystemTime 固定时间,避免 flaky

## 未达 90% 说明

admin 85.67%,差 4.33pp。剩余 gap:

| 文件 | 覆盖率 | 原因 |
|---|---|---|
| App.vue | 0% | 根组件,渲染入口,Vue 生态不单测 |
| FloatingInput.vue | 0% | co-browse 浮动输入框,e2e 兜底 |
| ReplayViewer.vue | 0% | rrweb-player 集成,需 mock rrweb-player 库 |
| LoginView.vue | 84.84% | 表单提交路径,e2e 已覆盖 |
| useWs.ts | 80.57% | WebSocket composable,需 mock WS 完整流程 |
| ReplayList.vue | 92.5% | 已接近达标 |

**决策**:剩余 gap 多为 .vue 组件(Vue 生态不单测 + e2e 兜底)和复杂集成(useWs.ts),按 plan 风险点 #5 接受 85.67%。

## 验证

```bash
cd admin && pnpm exec vitest run --coverage
# All files: 85.67% lines / 80.36% branches / 75.92% functions
# sessions.ts: 100% / time.ts: 100%
```
