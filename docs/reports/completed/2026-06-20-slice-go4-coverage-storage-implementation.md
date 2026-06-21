# Slice Go-4 implementation:storage 适配器补测

> **状态**:completed
> **深度 badge**:🟡 touched(全 docker 集成;实测 86.5% 未达 90% 目标)
> **实际工时**:~2.5h(预估 12h,因适配器测试模式直接复用 helperPGPool + ConnectX 模式)

## 实施清单

### `server/internal/storage/adapters_coverage_test.go`(新建,~530 行,~30 测试)

#### 新增 helper

- `helperRedis(t) *Redis`:用 ConnectRedis 建立连接,不可用 skip(替代 antiscrape 包私有 skipIfNoRedis)
- `helperPostgresCfg()` / `helperRedisCfg()` / `helperMinIOCfg(t)`:构造测试配置
- `discardLogger()`:用 `io.Discard` 输出

#### 适配器测试

| 测试 | 覆盖 |
|---|---|
| TestConnectPostgres_Success | ConnectPostgres 建立连接 + Ping 通过 |
| TestConnectPostgres_BadConfig | 错误端口/DB 返回 error |
| TestPostgres_Close_Idempotent | 多次 Close 不 panic |
| TestConnectRedis_Success | ConnectRedis + Ping |
| TestConnectRedis_BadAddr | 错误地址返回 error |
| TestRedis_SetGet_Del_TTL | Set→Get→TTL→Del round-trip |
| TestRedis_Get_NonExistingKeyReturnsNil | 不存在 key 返回 nil(非 error) |
| TestRedis_SetNX_AtomicFirstCallWins | SetNX 原子首次赢 |
| TestRedis_EvalLua_PingScript | 简单 Lua return |
| TestRedis_EvalLua_CompareAndDel | compare-and-del claim release 模式 |
| TestRedis_Close_Idempotent / NilClientSafe | nil Client 不 panic |
| TestConnectMinIO_Success | ConnectMinIO 自动建 bucket |
| TestConnectMinIO_BadEndpoint | 错误 endpoint 返回 error |
| TestMinIO_PingPutGet | Ping + Put/Get round-trip |
| TestMinIO_GetBytes_NonExisting | 不存在 object |
| TestMinIO_Close_NoOp | Close 多次 no-op |
| TestStores_ConnectSuccess | Stores.Connect 三连接 + PingAll |
| TestStores_Connect_BadPGReturnsError 等 | bad PG/Redis/MinIO 各自返回 error |
| TestStores_Close_NilSafe / PartialStores | nil 字段不 panic |
| TestStores_PingAll_PGFailureReturnsError / RedisFailure | PG/Redis 关闭后 PingAll 报错 |
| TestRedis_Get/Set/SetNX/Del/EvalLua/TTL_ErrorPathClosedConnection | 关闭后所有操作返回 error |
| TestListEventBlobKeysBySessions_EmptyInputReturnsNil / ReturnsMatchingKeys / NoMatchReturnsEmpty | 空输入 + 真实数据 2 keys + 不匹配 |

## 关键修正(实施过程发现)

### 修正 1:`*redis.Client` vs `*storage.Redis`

初版 `helperRedis` 返回 `*redis.Client`(go-redis 直接类型),但 Get/Set/SetNX/EvalLua 是 `*storage.Redis`(包装层)方法。修正为 `helperRedis` 返回 `*storage.Redis`,用 ConnectRedis 创建。

### 修正 2:event_blobs 表结构猜错

初版 INSERT 用了不存在的 `payload` 列。实际 schema:
- `started_at` / `ended_at` / `event_count` / `minio_object_key` / `size_bytes` / `checksum_sha256`
- 没有 `payload` / `content_type`

**反思**:补 repo 测试前先看 migration schema,不要凭印象写 INSERT。

### 修正 3:Lua 关键字大小写

`RETURN "..."` 在 Lua 中报"nonexistent global variable"。Lua 关键字必须小写:`return "..."`。

### 修正 4:config.Default() 不存在

构造 Config 时初版用 `config.Default()`,实际无此函数。改为 `&config.Config{Postgres: ..., Redis: ..., MinIO: ...}`(Stores.Connect 只用这三个字段)。

## 实测覆盖率(2026-06-20)

| 文件 | 主要函数覆盖率 |
|---|---|
| minio.go | Connect 83.3% / Ping 100% / PutBytes 75% / GetBytes 87.5% / Close 0%(no-op 不统计) |
| postgres.go | ConnectPostgres 84.6% / Ping 100% / Close 100% |
| redis.go | ConnectRedis 100% / Ping 100% / Close 100% / Set/SetNX/Get/Del/EvalLua/TTL 75-100% |
| stores.go | Connect 100% / Close 100% / PingAll ~80% |
| erasure_repo.go | ListEventBlobKeysBySessions 84.6% |
| **整包** | **86.5%** 🟡(差 3.5pp 到 90%) |

## 未达 90% 的说明

剩余 3.5pp gap 分布:

1. **minio.Close 0%**:`func (m *MinIO) Close() {}` 是 no-op 空函数,go tool cover 不统计(测试中调用了但工具显示 0%)。这是工具特性不是真实未覆盖。
2. **各 repo 函数 80-90%**:scan 字段的边缘分支(nil UA / invalid IP / 错误 checksum 等),触发需要构造特殊数据,ROI 低。
3. **erasure_repo.DeleteVisitorByFingerprint 67.7%**:多表 cascade 删除,部分错误分支需要 mock FK 失败,工程上不值得。

**决策**:按 plan 风险点 #5"不为达 90% 写'为测而测'的弱断言测试",接受 86.5%。在 project-status 显式标注 storage 是 Go phase 中唯一未达 90% 的包,留作后续 R&D backlog。

## 副作用与回归

- ✅ 不改业务代码
- ✅ 现有 16 个 storage repo 测试不破坏
- ✅ 全部测试 PASS(无 flaky)

## 同步文档

- ✅ project-status §2.1 storage 行更新 57.6%→86.5%,标注"未达 90% 目标 3.5pp"
- ✅ daily 追加 Go-4 段

## 后续切片解锁

Go-5(recording)/ Go-6(api)可启动。Go-6 api 依赖本切片 + hub(Go-3 已完成)。
