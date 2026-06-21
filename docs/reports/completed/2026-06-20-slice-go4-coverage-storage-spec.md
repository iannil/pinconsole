# Slice Go-4 spec:storage 适配器补测

> **状态**:completed
> **工时预估**:12h | **实际工时**:~2.5h
> **深度 badge**:🟡 touched(全 docker 集成,但未达 90% 目标)

## 范围

| 包 | 切片前 | 切片后 | 提升 |
|---|---|---|---|
| internal/storage | 57.6% | **86.5%** | +28.9pp |

**注**:未达 ≥90% 目标 3.5pp,详见"未达目标说明"。

## 主要工作

新建 `server/internal/storage/adapters_coverage_test.go`(~530 行,~30 测试):

**适配器层(0% → 80%+)**:
- ConnectPostgres / Ping / Close 成功 + bad config + 多次 Close
- ConnectRedis / Ping / Close 成功 + bad addr + 多次 Close + nil Client
- ConnectMinIO / Ping / Close 成功 + bad endpoint + PutBytes/GetBytes round-trip
- Stores.Connect / Close / PingAll 成功 + bad PG/Redis/MinIO 各自 + Close nil safe

**Redis 操作层(0% → 100%)**:
- Set/Get/Del/TTL round-trip
- SetNX 原子首次赢
- EvalLua 简单返回 + compare-and-del(claim release 模式)
- 关闭连接后所有操作返回 error(Get/Set/SetNX/Del/EvalLua/TTL)

**erasure_repo**:
- ListEventBlobKeysBySessions 空输入 + 真实数据(2 keys) + 不匹配返回空

## 未达目标说明

实测 86.5%,差 3.5pp 到 90%。剩余 gap 分布:

| 函数 | 覆盖率 | 剩余 gap |
|---|---|---|
| minio.go:Close | 0% | no-op 空函数,go tool cover 不统计 |
| stores.go:PingAll | 57.1% → 已提升 | (已补 PG/Redis fail 测试) |
| erasure_repo.DeleteVisitorByFingerprint | 67.7% | 多分支删除(5 表),部分 edge case 未触发 |
| session_repo.List*SessionsByTenant | 78.9% | netip.ParseAddr 失败分支(invalid IP) |
| 各 repo 函数 | 80-90% | scan 字段 nil/empty 分支 |

**决策**:按 plan 风险点 #5"不为达 90% 写'为测而测'的弱断言测试",不为刷数字补边缘分支测试(如 seed invalid IP 触发 ParseAddr error)。86.5% 是本切片的真实结果,在 project-status 显式说明。

## 验证

```bash
cd server && go test -count=1 -cover ./internal/storage/...
# ok  coverage: 86.5% of statements
```
