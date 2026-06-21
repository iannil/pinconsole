# Slice Go-5 implementation:recording 包补测

> **状态**:completed
> **深度 badge**:🟡 touched
> **实际工时**:~2h(预估 12h)

## 实施清单

### `server/internal/recording/coverage_extra_test.go`(新建,~570 行,~16 测试)

#### 新增 helper

- `helperRedisStore(t) *storage.Redis`:用 ConnectRedis,不可用 skip
- `helperStoresRec(t) *storage.Stores`:建立 PG+Redis+MinIO 全连接,MinIO bucket 随机
- `recDiscardLogger()`:io.Discard
- `minioRemoveOpts()`:minio.RemoveObjectOptions{}

#### 测试矩阵

| 测试 | 覆盖 |
|---|---|
| TestSnapshotKey_Format | 纯函数 key 格式 |
| TestNewSnapshotCache | 构造函数 |
| TestSnapshotCache_SetGetDelete | Redis round-trip |
| TestSnapshotCache_Get_NonExistingReturnsNil | non-existing 返回 nil |
| TestDefaultGCConfig | 默认值 |
| TestNewGC | 构造函数 |
| TestGC_Stop_Idempotent | Stop 多次不 panic |
| TestGC_StartStop_QuickExit | Start+Stop 不阻塞 |
| TestDefaultConfig | flusher 默认值 |
| TestFlusher_RegisterUnregister | map 操作 + Unregister 触发 flush |
| TestFlusher_Unregister_NonExistingSession | 不存在不 panic |
| TestFlusher_Stop_Idempotent / StartStop_QuickExit | Stop 幂等 + Start+Stop |
| TestStream_Delete / Delete_NonExisting | Redis DEL stream |
| TestStream_Len_NilStreamReturnsZero | non-existing 返回 0 |
| TestStream_Range_NilStreamReturnsEmpty | non-existing 返回空 |
| TestStream_AppendTrimRange | Append+Trim+Range round-trip |
| TestGC_RunOnce_DeletesOldEventBlob | seed 旧 event_blob+MinIO,runOnce 删 |
| TestGC_RunOnce_KeepsRecentEventBlob | seed 未过期,runOnce 不删 |
| TestFlusher_FlushSession_RealData | Append+Unregister 触发 flush,PG event_blob 创建 |

## 关键修正

### 修正 1:NewStream 签名

初版 `NewStream(rdb)`(传 `*storage.Redis`)编译错。实际签名 `NewStream(rdb *redis.Client, logger *slog.Logger)`。修正:`NewStream(rdb.Client, recDiscardLogger())`(从 storage.Redis 取底层 Client)。

### 修正 2:stream.Append 不是 stream.Add

初版用 `stream.Add`,实际方法名是 `Append`。

### 修正 3:minioRemoveObjectOpts 是变量不是类型

初版 `func minioRemoveOpts() minioRemoveObjectOpts { return minioRemoveObjectOpts{} }`,实际 `minioRemoveObjectOpts` 是包级变量。修正为返回 `minio.RemoveObjectOptions{}`。

### 修正 4:Trim MAXLEN APPROX 异步

XTRIM MAXLEN APPROX 是异步近似,等 50ms 不足。改测试不严格断言具体值(避免 flaky)。

### 修正 5:visitors INSERT placeholder

初版 SQL `INSERT INTO visitors (id, $2, ...)` 在列名用 placeholder 报错。改为显式列名 + VALUES placeholder。

## 实测覆盖率(2026-06-21)

| 文件 | 主要函数 |
|---|---|
| snapshot.go | SnapshotKey/NewSnapshotCache/Set/Get/Delete 全 100% |
| gc.go | DefaultGCConfig/NewGC/Start/Stop 89-100% / runOnce 54.5% |
| stream.go | Stream.Delete/Len/Range/Append 75-100% / Flusher.Register/Unregister/Start/Stop 100% / flushSession 65.5% / tick 75% |
| **整包** | **77.7%** 🟡(差 12.3pp 到 90%) |

## 未达 90% 的说明

剩余 12.3pp gap:
- gc.runOnce 5 表 cascade 路径(event_blob 已覆盖,chat_messages/co_browsing_commands/sessions/visitors 各分支需 seed + 验证)
- flushSession MinIO/PG 失败补偿路径(需 mock 故障)
- stream 各方法错误分支(redis.Nil 已部分覆盖,网络错误未模拟)

**决策**:按 plan 风险点 #5"不为达 90% 写为测而测的弱断言测试",接受 77.7%。recording 是 Go phase 第二个未达 90% 的包(storage 86.5% + recording 77.7%),留 R&D backlog。

## 副作用与回归

- ✅ 不改业务代码
- ✅ 现有 4 个 recording 测试不破坏(encode_test/flusher_wiring/observability_wiring/snapshot_wiring)
- ✅ 全部测试 PASS(无 flaky,Trim 测试改为宽松断言)

## 同步文档

- ✅ project-status §2.1 recording 行更新 48.0%→77.7%,标注"未达 90% 目标 12.3pp"
- ✅ daily 追加 Go-5 段(跨日 2026-06-21)

## 后续切片解锁

Go-6(api)可启动,依赖 hub(✅ Go-3)+ storage(✅ Go-4)+ recording(✅ Go-5)全部完成。
