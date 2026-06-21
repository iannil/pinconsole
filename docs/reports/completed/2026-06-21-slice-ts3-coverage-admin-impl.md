# Slice TS-3 implementation:admin 纯逻辑层

> **状态**:completed
> **深度 badge**:🟡 touched
> **实际工时**:~1h(预估 8h,因 sessions.ts + time.ts 都是纯函数)

## 实施清单

### `tests/sessions_coverage.test.ts`(新建,~140 行,12 测试)

mock apiJson,验证 5 个函数:

| 函数 | 测试 case |
|---|---|
| listEndedSessions | 默认(24h/200) / 自定义(7d/50) / 30d range |
| getSessionReplay | 默认(0/10000) / 自定义(100/50) / encodeURIComponent('s/special') |
| sendCommand | POST + body / 8 个命令类型 cursor_highlight~chat_message |
| listMessages | 默认 sinceId=0 / 自定义=10 |
| sendMessage | 默认 sender=operator / 显式 visitor |

### `tests/time_coverage.test.ts`(新建,~75 行,8 测试)

formatRelative 5 区间 + 3 边界:

| 区间 | 测试 | i18n key |
|---|---|---|
| < 1s | just_now | time.just_now |
| < 60s | seconds_ago(30s/59s) | time.seconds_ago |
| < 60min | minutes_ago(5m/59m) | time.minutes_ago |
| < 24h | hours_ago(3h/23h) | time.hours_ago |
| >= 24h | fallback_date(48h) | time.fallback_date |

## 关键技巧

### 技巧 1:vi.mock 模块级 mock

sessions.ts 调 `apiJson`,测试 mock 整个 `../src/api/client` 模块:

```typescript
vi.mock('../src/api/client', () => ({
  apiJson: vi.fn(),
}));
```

每个测试 `mockResolvedValueOnce` 设置返回值 + `toHaveBeenCalledWith` 验证调用参数。

### 技巧 2:vi.useFakeTimers + setSystemTime

time.ts 的 formatRelative 依赖 `Date.now()`,测试用 fake timer 固定时间避免 flaky:

```typescript
beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-06-21T12:00:00Z'));
});
afterEach(() => vi.useRealTimers());
```

### 技巧 3:mockT 模拟 vue-i18n 的 ComposerTranslation

formatRelative 接受 vue-i18n 的 t 函数,测试用 mockT 模拟:

```typescript
const mockT = vi.fn((key, params) => {
  if (key === 'time.seconds_ago') return `${params.n}s ago`;
  // ...
});
```

## 实测覆盖率(2026-06-21)

```bash
$ cd admin && pnpm exec vitest run --coverage
All files     | 85.67 | 80.36 | 75.92 | 85.67
sessions.ts   |   100 |   100 |   100 |   100
time.ts       |   100 |   100 |   100 |   100
```

## 未达 90% 的说明

admin 85.67%,差 4.33pp。剩余 .vue 组件 + useWs.ts 复杂集成,按 plan 风险点 #5 接受。

## 副作用与回归

- ✅ admin 现有 9 个测试不破坏
- ✅ 新测试全 PASS(sessions 12 + time 8 = 20 测试)
- ✅ coverage threshold 60% 通过

## 同步文档

- ✅ project-status §2.1 admin 行更新 82.17%→85.67%
- ✅ daily 追加 TS-3 段

## 后续切片

- TS-4:3 个 .vue 组件测试(VisitorList/ChatPanel/LoginView)~6h
- TS-5:门槛拉到 90% ~4h
- visitor-sdk src/index.ts backlog ~6-8h
