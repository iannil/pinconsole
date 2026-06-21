# Slice Go-1 implementation:proto + antiscrape + observability 小包冲刺

> **状态**:completed
> **对应 spec**:[spec](./2026-06-20-slice-go1-coverage-misc-spec.md)
> **深度 badge**:🟢 touched
> **实际工时**:~1.5h(预估 4h)
> **Disclaimer**:本切片只补测试,不改业务代码;mock 边界严格遵循"测自己的代码不测上游库"。

## 实施清单

### 1. `server/internal/proto/envelope_extra_test.go`(新建,~100 行)

7 个测试:

| 测试 | 覆盖目标 | 关键断言 |
|---|---|---|
| TestDecodePayload_RoundTrip | HelloPayload 完整 round-trip | visitor/session/sdk_version 字段保留 |
| TestDecodePayload_PrimitivePayload | int 类型 payload 解码 | 5 个 int 子类型(int/int8/16/32/64) |
| TestDecodePayload_StringPayload | string payload | 字符串保留 |
| TestDecodePayload_BoolPayload | bool payload | true 解码 |
| TestDecodePayload_IncompatibleDestination | nil dst | Unmarshal 错误 |
| TestDecodePayload_MarshalFailure | chan payload | Marshal 错误(75%→100% 关键) |

**proto 覆盖率**:88.9% → **100.0%** ✅

### 2. `server/internal/antiscrape/coverage_extra_test.go`(新建,~150 行)

9 个测试:

| 测试 | 覆盖目标 | Redis 依赖 |
|---|---|---|
| TestDefaultRateLimitConfig | 默认值 RequestsPerMin=60 / Window=Minute | 无 |
| TestIsSessionFlagged_NotFlagged | 未标记返回 false | ✅ skipIfNoRedis |
| TestIsSessionFlagged_Flagged | FlagSession 后查到 | ✅ skipIfNoRedis |
| TestExtractXY_InvalidData | 5 种无效 x/y 场景 | 无 |
| TestExtractXY_Valid | happy path | 无 |
| TestObserve_NonRRWebEvent | 非 rrweb 事件被忽略但仍计 totalEvents | 无 |
| TestObserve_RRWebNonIncrementalSnapshot | FullSnapshot(type=2) 不统计 mousemove | 无 |
| TestObserve_NilRRWebData | IncrementalSnapshot 但 data=nil 不 panic | 无 |
| TestObserve_MouseInteractionWithoutXY | 点击但无 x/y 不污染 clickPositions | 无 |
| TestCheckAndFlag_NoReasons | 无启发式满足时不调 FlagSession | ✅ skipIfNoRedis |

**antiscrape 覆盖率**:86.7% → **95.9%** ✅

### 3. `server/internal/observability/lifecycle_extra_test.go`(新建,~120 行)

7 个测试:

| 测试 | 覆盖目标 |
|---|---|
| TestLifecycleWithArgs_OptionCreatesConfig | option 直接调用设置 args |
| TestLifecycle_WithArgsOption | Lifecycle 传 WithArgs 不 panic + 日志含 Function_Start/End |
| TestLogPoint_WithMultipleExtras | 多 extras 字段写入 |
| TestLogExternalCall_WithMultipleExtras | 多 extras 字段写入 |
| TestLogPoint_NoExtras | 无 extras 也能写 |
| TestLogExternalCall_NoExtras | 无 extras 也能写 |
| TestLifecycle_NilLoggerWithOptions | nil logger + opts 不 panic |

**observability 覆盖率**:83.3% → **91.7%** ✅

## 实施过程中的修正

### 修正 1:proto Decode 函数签名误读

初版写 `Decode(raw, &env)` 双参数,实际签名是 `func Decode(b []byte) (Envelope, error)` 单参数返回值。修正为 `env, err := Decode(raw)`。

**反思**:Go 函数签名习惯——`Unmarshal` 类函数用 `(input, &dst)` 写入式;`Decode` 类函数常返回 `(dst, err)` 值式。要按具体函数确认,不要按命名直觉。

### 修正 2:observability EventType 常量值大小写

初版测试用 `strings.Contains(logs[0], "function_start")`,但实际 event_type JSON 值是 `"Function_Start"`(PascalCase)。

**反思**:Go 常量值与常量名命名风格可能不一致。`EventFunctionStart` 是 PascalCase 常量名,值也是 PascalCase 字符串。要直接用 `string(EventFunctionStart)` 比较,不要凭印象写字符串。

### 修正 3:proto DecodePayload 75% 真正 gap

`go tool cover -func` 显示 `DecodePayload 75.0%`。函数体 4 statement:

1. `raw, err := msgpack.Marshal(payload)` — covered
2. `if err != nil { return err }` 中的 `if` — covered(每次跑 if)
3. `return err`(Marshal 失败时) — **NOT covered** ⬅ gap
4. `return msgpack.Unmarshal(raw, dst)` — covered

触发 Marshal 失败需传 msgpack 无法序列化的类型:**chan 是唯一可靠选项**(func 也行但更难验证)。

初版用 nil dst 测试,只触发了 Unmarshal 错误,不覆盖 Marshal 错误路径。修正为加 `TestDecodePayload_MarshalFailure` 用 chan payload。

## 验证(实测)

```bash
cd /Users/rong.zhu/Code/pinconsole/server

# 启动 docker(本切片需要 Redis)
docker compose up -d redis 2>&1 | tail -3
# 等就绪
until docker compose exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; do sleep 1; done

# 跑测试 + 覆盖率
go test -count=1 -cover ./internal/proto/...
# ok  github.com/iannil/pinconsole/internal/proto  coverage: 100.0%

go test -count=1 -cover ./internal/antiscrape/...
# ok  github.com/iannil/pinconsole/internal/antiscrape  coverage: 95.9%

go test -count=1 -cover ./internal/observability/...
# ok  github.com/iannil/pinconsole/internal/observability  coverage: 91.7%
```

**3 个包全部 ≥ 90%**,达成切片目标。

## 副作用与回归

- ✅ 不改业务代码(只增测试)
- ✅ 现有测试仍全绿(`make test-go` 11 包全 PASS)
- ✅ antiscrape Redis 测试用 `skipIfNoRedis`,无 Redis 自动跳过(CI 兼容)

## 同步文档

- ✅ `docs/project-status.md` §2.1 更新 proto/antiscrape/observability 三行覆盖率
- ✅ `memory/daily/2026-06-20.md` 追加 Go-1 段
- ✅ spec + impl 移到 `docs/reports/completed/`

## 后续切片解锁

本切片完成,Go-2(logging)可启动,无依赖关系。
