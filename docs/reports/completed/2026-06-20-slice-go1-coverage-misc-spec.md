# Slice Go-1 spec:proto + antiscrape + observability 小包冲刺

> **状态**:completed
> **工时预估**:4h
> **实际工时**:~1.5h
> **深度 badge**:🟢 touched(纯单测 + Redis 集成)
> **Disclaimer**:本切片只补"未达 100%"的函数测试,不重构业务代码。

## 范围

3 个 Go 包从 79-89% 提升到 ≥90%:

| 包 | 切片前 | 切片后(实测) | 提升 |
|---|---|---|---|
| internal/proto | 88.9% | **100.0%** | +11.1pp |
| internal/antiscrape | 86.7% | **95.9%** | +9.2pp |
| internal/observability | 83.3% | **91.7%** | +8.4pp |

## 主要工作

### proto(补 1 个文件,7 测试)
- `envelope_extra_test.go`:DecodePayload 边界(marshal chan 失败、原始类型 payload、nil dst、incompatible destination)

### antiscrape(补 1 个文件,9 测试)
- `coverage_extra_test.go`:DefaultRateLimitConfig、IsSessionFlagged(2 测试,Redis 集成)、extractXY invalid/valid、Observe 边界(非 rrweb、FullSnapshot、nil data、无 x/y)、CheckAndFlag_NoReasons

### observability(补 1 个文件,7 测试)
- `lifecycle_extra_test.go`:LifecycleWithArgs option 直接调用、Lifecycle 传 WithArgs、LogPoint/LogExternalCall 多 extras 路径、nil logger 不 panic

## 关键修正(实施时发现的)

1. proto `Decode(b []byte) (Envelope, error)` 是单参数返回值,不是 `Decode(b, &env)` 双参数(我初版写错了)
2. observability `EventFunctionStart` 实际值是 `"Function_Start"`(大写),不是 `"function_start"`(我初版用小写)
3. proto DecodePayload 75% 的真正 gap 是 `msgpack.Marshal` 失败路径——用 chan payload 触发,不是 nil dst

## 验证

```bash
cd /Users/rong.zhu/Code/pinconsole/server
go test -count=1 -cover ./internal/proto/...         # 100.0%
go test -count=1 -cover ./internal/antiscrape/...    # 95.9%
go test -count=1 -cover ./internal/observability/... # 91.7%
```

## 风险

- 🟢 全部为纯单测 + Redis 集成,无业务逻辑变更
- 🟢 不破坏现有测试(只增不减)
- 🟢 antiscrape Redis 测试用 skipIfNoRedis,无 Redis 时自动跳过
