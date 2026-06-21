# Slice Go-5 spec:recording 包补测

> **状态**:completed
> **工时预估**:12h | **实际工时**:~2h
> **深度 badge**:🟡 touched(全 docker 集成;实测 77.7% 未达 90% 目标)

## 范围

| 包 | 切片前 | 切片后 | 提升 |
|---|---|---|---|
| internal/recording | 48.0% | **77.7%** | +29.7pp |

**注**:未达 ≥90% 目标 12.3pp,详见"未达目标说明"。

## 主要工作

新建 `server/internal/recording/coverage_extra_test.go`(~570 行,~16 测试):

**纯函数 + 构造函数(0% → 100%)**:
- DefaultGCConfig / DefaultConfig 默认值
- SnapshotKey 格式
- NewGC / NewSnapshotCache 返回非 nil

**Redis 集成(0% → 100%)**:
- SnapshotCache Set/Get/Delete round-trip + non-existing 返回 nil
- Stream Append/Trim/Range/Len/Delete round-trip + non-existing 处理

**Flusher map 操作(0% → 100%)**:
- Register/Unregister(map 操作)+ 同 session 不重复 + Unregister 不存在不 panic
- Start/Stop 后台 ticker 快速退出 + Stop 幂等

**GC.Start/Stop(0% → 89%)**:
- Start+Stop 快速退出 + Stop 幂等
- runOnce 真实数据:seed 旧 event_blob + MinIO 对象,runOnce 删除;未过期 event_blob 保留

**Flusher.flushSession 真实数据**(已有部分覆盖):
- Append 5 条 + Unregister 触发 flush,验证 PG event_blob 创建

## 未达目标说明

实测 77.7%,差 12.3pp 到 90%。剩余 gap:

| 函数 | 覆盖率 | 剩余 gap |
|---|---|---|
| gc.runOnce | 54.5% | event_blob 已覆盖,但 chat_messages/co_browsing_commands/sessions/visitors 各分支未单独触发 |
| stream.flushSession | 65.5% | 错误路径(MinIO fail / PG fail 补偿) |
| stream.Append/Range/Len | 57-77% | redis.Nil 等错误分支 |
| stream.tick | 75% | 阈值触发 flush 的真实数据路径 |

**决策**:剩余 gap 都是复杂多表 seed + 错误路径模拟,ROI 低(类似 storage Go-4)。按 plan 风险点 #5 接受 77.7%,在 project-status 显式标注 recording 是第二个未达 90% 的包。

## 验证

```bash
cd server && go test -count=1 -cover ./internal/recording/...
# ok  coverage: 77.7% of statements
```
