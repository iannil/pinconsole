# 切片 1x-login-throttle 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应 spec**:[2026-06-18-slice-1x-login-throttle.md](./2026-06-18-slice-1x-login-throttle.md)
**对应审计**:[2026-06-18-deep-audit.md](../audits/2026-06-18-deep-audit.md) P1-3
**深度 badge**:🟢 verified-deep

## Summary

修审计 P1-3:`POST /api/auth/login` 无 brute-force 防护。加 Redis 计数器(email+IP 双 key),5 次失败后锁定 15 分钟,返回 429 + Retry-After header。fail-open 策略(Redis 故障仅 warn,与 1i 反爬虫一致)。结合 1k(默认 admin password 已修)构成完整登录安全栈。

## Changes Delivered

### 后端 Go(改 1 文件)

- ✅ `server/internal/api/auth.go`:
  - 常量 `loginMaxAttempts=5`、`loginLockoutWindow=15*time.Minute`
  - `login` handler 入口调 `checkLoginThrottle`,锁定时返 429 + Retry-After header + JSON `{"error":"too_many_attempts","retry_after":N}`
  - 失败路径调 `recordLoginFailure`(Lua 原子 INCR + 首次失败时 EXPIRE)
  - 成功路径调 `Del` 清零(防偶然失败锁定正常用户)
  - 新 helper:`checkLoginThrottle`、`recordLoginFailure`、`loginThrottleKey`
  - Redis 故障 fail-open(仅 warn,不阻断)

### Go 单测(1 新建)

- ✅ `server/internal/api/auth_test.go`(新建):
  - `TestLoginThrottleKey_Format`:key 格式契约
  - `TestLoginThrottle_Constants`:常量未误改
  - `TestLoginThrottle_LockAfter5Failures`:5 次失败后第 6 次 locked=true + TTL 在窗口内
  - `TestLoginThrottle_DelOnSuccess`:成功后清零
  - `TestLoginThrottle_IsolationByIpAndEmail`:不同 IP / 不同 email 独立计数
  - 编译契约:`storage.Stores.Redis` / `storage.Redis.EvalLua`

## Verification

```bash
# 1. Go 测试
cd server && go test ./... -count=1
# 预期:12 packages ALL PASS(新增 auth_test.go 5+ 测试)

# 2. go vet
cd server && go vet ./...
# 预期:clean

# 3. Runtime:启动 + 7 次错误密码
./ops.sh start
for i in 1 2 3 4 5 6 7; do
  curl -A 'Mozilla/5.0' -X POST http://localhost:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"test@example.com","password":"wrong"}' \
    -w "attempt $i: %{http_code}\n" -o /dev/null -s
done
# 预期:1-5 → 401 invalid_credentials
#      6-7 → 429 too_many_attempts + Retry-After: 900
```

**实测结果**:
- ✅ Go 测试:12 packages ALL PASS
- ✅ go vet:clean
- ✅ Runtime 端到端:
  - attempt 1-5:HTTP 401 invalid_credentials
  - attempt 6:HTTP 429 + body `{"error":"too_many_attempts","retry_after":900}`
  - attempt 7:仍 429(锁定持续)
  - 日志:`level=WARN msg="login rejected: throttle locked" client_ip=::1 retry_after_s=900`
  - Redis key:`auth:throttle:test@example.com:::1`(IP 维度隔离)

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ 5 次失败后第 6 次 429 + 真实 Retry-After header |
| Negative case | ✅ 不同 IP / 不同 email 独立计数(不互相锁定) |
| 边界 | ✅ 成功登录清零 + Redis 故障 fail-open + key 格式契约 |
| 真实集成 | ✅ 真 Redis + 真 server + 真 curl 7 次连续请求 |
| 可重复运行 | ✅ -count=1 多次无 flaky |

**结论**:🟢 verified-deep。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| 仅 email+IP 双 key,未加全局 IP 限速 | 全局 IP 限速由 1i rate limit middleware 覆盖(60/min)。本切片聚焦"同一 email 的字典攻击" |
| 锁定期间正确密码也返 429 | 防时间侧信道 + 防止"暴力试到正确密码的最后 1 次"绕过锁定 |

## Follow-ups

- **CAPTCHA 触发**(P3):失败 3 次后要求 CAPTCHA,5 次后完全锁定。需要前端配合
- **audit log**(P2):登录失败/锁定事件写 PG 审计表,供 admin 查看
- **可配置阈值**(P3):环境变量 `LOGIN_MAX_ATTEMPTS` / `LOGIN_LOCKOUT_WINDOW`(目前硬编码)

## Notes

- 本切片是 audit P1-3 直接产物,跳过 grill-me
- 与 1i rate limit 形成纵深防御:1i = 全局请求频率,1x = 同一账号字典攻击
- fail-open 与 1i 一致(Redis 故障不阻断业务,与 CLAUDE.md "反爬虫 fail-open" 一致)
- 锁定维度选 email+IP 而非仅 email:防同一 IP 多账号爆破,也防同一账号换 IP(后者部分缓解,完全防需配合 WAF)
