# 切片 1ai-c auth-handler-interface — Implementation

**切片编号**:1ai-c
**类型**:重构 + 测试深化(api 包 Phase 1)
**创建时间**:2026-06-20
**状态**:completed
**关联**:[spec](./2026-06-20-slice-1aic-auth-handler-interface-spec.md)、[1ag impl](../completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md)

## Context

api 包覆盖 29.3%,login happy path 全靠 e2e 兜底。本切片用"accept interfaces, return structs"模式重构 AuthHandler,Phase 1 验证可行性 + 补 happy path。

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **3 个**(happy path:success / wrong password / user not found) |
| 新增测试文件 | 2(`auth_interfaces.go`、`auth_happy_path_test.go`) |
| 修改文件 | 4(auth.go / redis.go / 3 个既有 test 文件 stores→redis 字段) |
| api 包覆盖率 | **29.3% → 31.2%**(+1.9pp) |
| auth.go `login` | **37.8% → 86.5%**(+48.7pp,超 spec 目标 70%) |
| Go 全测试 | ✅ ALL PASS(12 包,无 regression) |
| Mutation 抽样 | ✅ 2/2 KILLED |

## 重构内容

### 1. `internal/api/auth_interfaces.go`(新文件)

定义 2 个最小接口(ISP 原则):

```go
type authUserRepo interface {
    GetUserByEmail(ctx, tenantID, email) (*User, error)
    GetUserByID(ctx, id) (*User, error)
}

type authRedisStore interface {
    Get(ctx, key) ([]byte, error)
    Set(ctx, key, value, ttl) error
    Del(ctx, key) error
    EvalLua(ctx, script, keys, args) (any, error)
    TTL(ctx, key) (time.Duration, error)
}
```

### 2. `internal/storage/redis.go`(+ TTL 方法)

加 `TTL(ctx, key)` 方法,替代 auth.go 直接调 `Client.TTL`。

### 3. `internal/api/auth.go`(字段重构)

```go
type AuthHandler struct {
    userRepo    authUserRepo    // 接口, *Postgres 自动满足
    redis       authRedisStore  // 接口, *Redis 自动满足
    logger      *slog.Logger
    secureCookie bool
}

// NewAuthHandler 签名不变(仍接受 *storage.Stores),内部抽取适配接口。
func NewAuthHandler(stores *storage.Stores, ...) *AuthHandler {
    return &AuthHandler{
        userRepo: stores.PG,
        redis:    stores.Redis,
        ...
    }
}
```

### 4. 既有测试更新

`stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}}` → `redis: &storage.Redis{Client: rdb}`:
- `auth_test.go` (3 处)
- `auth_bcrypt_test.go` (1 处 + 删除 Redis.Client 检查)
- `auth_http_test.go` (1 处)

## 新增测试

### `auth_happy_path_test.go`(3 测试 + 2 mock)

| 测试 | 验证 |
|---|---|
| `TestLogin_Success_Returns200_SetCookie_Body` | mock user + bcrypt 匹配 → 200 + Set-Cookie + meResponse;断言 GetUserByEmail×1 + Set×1 + Del×1 |
| `TestLogin_WrongPassword_Returns401_NoCookie` | mock user + bcrypt 不匹配 → 401 + 无 Set-Cookie;断言 EvalLua×1(recordLoginFailure)、Set×0 |
| `TestLogin_UserNotFound_Returns401_RecordsFailure` | mock 返回 error → 401;断言 EvalLua×1(防字典攻击计数)、Set×0 |

### Mock 实现(手写,~80 行)

- `mockUserRepo`:map 存储,thread-safe,记录调用计数
- `mockRedisStore`:map 存储,thread-safe,记录 Set/Del/EvalLua/Get/TTL 调用

## 覆盖率前后对比

### `internal/api/auth.go`

| 函数 | 1ai-b 后 | 1ai-c 后 |
|---|---|---|
| `login` | 37.8% | **86.5%** |
| `checkLoginThrottle` | 76.5% | 76.5%(Redis fail-open 路径未触达) |
| `recordLoginFailure` | 66.7% | 66.7%(fail-open 路径未触达) |
| `logout` | 85.7% | 85.7% |
| `me` | 40% | 40%(happy path 需 userRepo mock,留 1ai-d) |

### api 包总体

**29.3% → 31.2%**(+1.9pp)

距 spec 目标 35% 差 3.8pp。原因:auth.go 部分 fail-open 路径 + NewAuthHandler(0%)+ RegisterMe(0%) 未覆盖。
但 auth.go `login` 单函数达 86.5%,核心目标已超 spec。

## Mutation 验证(2 项)

| Mutation | 应失败测试 | 结果 |
|---|---|---|
| bcrypt 参数顺序交换(hash, password → password, hash) | `TestLogin_Success` | ✅ KILLED(本应成功 → 失败) |
| 错密码路径 401 → 200 | `TestLogin_WrongPassword_Returns401_NoCookie` | ✅ KILLED |

## 关键模式

### "Accept interfaces, return structs"

接口定义在消费者(api 包),不污染生产者(storage 包)。
`NewAuthHandler` 仍接受具体 `*storage.Stores`,内部抽取 → 调用方零改动。

### 手写 mock vs mockgen

当前 2 接口共 ~80 行 mock 代码,手写可控。
未来接口数 >5 时再评估 mockgen。

### 测试断言 mock 调用计数

```go
if mockRedis.evalLuaCalls != 1 {
    t.Errorf("recordLoginFailure 未被调 — 字典攻击防护破坏")
}
```

捕获"代码改了路径但忘调用关键副作用"的回归,如错密码时忘 recordLoginFailure → throttle 失效。

## Verification Depth Badge

**🟢 touched** — AuthHandler 接口化 + 3 happy path 测试 + 2 mutation KILLED + 0 regression。

切片深度:
- **1h 认证 + 多运营 后端**:🟢 touched → 维持(login happy path 行为级覆盖)

## Follow-up(留 backlog)

1. **me handler happy path** — 需注入 mock userRepo(模式已验证,~30min)
2. **claim/chat/replay/session/command handler 接口化** — 同模式扩展,~6-8h(留 1ai-d+)
3. **NewAuthHandler 覆盖** — 需构造 *storage.Stores 跑测试,简单
4. **mockgen 工具链** — 接口数 >5 时评估

## 提交

建议 4 个 commit:

1. `refactor(1ai-c): storage/redis.go 加 TTL 方法 + auth_interfaces.go 定义接口`
2. `refactor(1ai-c): AuthHandler 字段改接口类型 + 既有测试同步`
3. `test(1ai-c): auth happy path 测试 — success/wrong password/user not found(3 测试 + 2 mock)`
4. `docs: 更新 project-status + 1ai-c impl 报告`

## 下一步

### 立即可做

- 用户审阅 + commit
- 更新 project-status.md 加 1ai-c 行

### 短期 backlog

- **1ai-d**:me handler happy path + logout happy path(同模式,~1h)
- **1ai-e**:claim/chat handler 接口化(中等工作量,~3-4h)
- **1ai-f**:replay/session/command handler 接口化(~3-4h)

### 长期(post-v1)

- 自定义域名 / 页面编辑器 / Tauri
- Mutation CI 集成(R7)
- mockgen 工具链评估
