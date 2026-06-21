# Slice Go-2 implementation:logging 包补测

> **状态**:completed
> **深度 badge**:🟢 touched
> **实际工时**:~1h(预估 4h)

## 实施清单

### 1. `server/internal/logging/handler_extra_test.go`(新建,~170 行)

8 个测试覆盖 `NewLogger` 全分支:

| 测试 | 验证点 |
|---|---|
| TestNewLogger_DevReturnsTextHandler | dev 模式输出 TextHandler 格式(msg=key=value,非 JSON) |
| TestNewLogger_ProdReturnsJSONHandler | prod 模式输出 JSONHandler 格式(以 `{` 开头) |
| TestNewLogger_SetsDefault | NewLogger 副作用:slog.SetDefault 被调用 |
| TestNewLogger_DevLevelDebug | dev+debug level 能输出 Debug 日志 |
| TestNewLogger_ProdLevelInfo | prod+info level 屏蔽 Debug 日志 |
| TestNewLogger_DevAddSource | dev 模式 AddSource=true 输出 source= 字段 |
| TestNewLogger_ProdNoAddSource | prod 模式 AddSource=false 不输出 source |
| TestNewLogger_TestEnvUsesJSON | env=test/staging/production/"" 全部走 JSONHandler 分支 |
| TestNewLogger_FromContextAfterInit | NewLogger + FromContext 协同(trace_id 正确注入) |

## 关键技巧

### 技巧 1:stdout 重定向捕获 slog 输出

NewLogger 内部写 `os.Stdout`,测试需要捕获。用 `os.Pipe()` 创建管道:

```go
origStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w
defer func() { os.Stdout = origStdout }()

logger := NewLogger("info", "dev")
logger.Info("test_msg")

w.Close()
var buf bytes.Buffer
_, _ = buf.ReadFrom(r)
out := buf.String()
```

### 技巧 2:slog.SetDefault 副作用清理

NewLogger 调用 `slog.SetDefault(logger)` 修改全局状态。每个测试必须 defer 恢复,避免污染后续测试:

```go
origDefault := slog.Default()
defer slog.SetDefault(origDefault)
```

### 技巧 3:TextHandler vs JSONHandler 区分

- TextHandler 输出 `msg="..."` (key=value 格式)
- JSONHandler 输出 `{"msg":"..."}` (JSON 格式)

判断方法:`strings.HasPrefix(out, "{")` 区分。

## 不覆盖的 newID fallback

`newID` 75% 覆盖,缺的是 `crypto/rand.Read` 失败时的 fallback:

```go
if _, err := rand.Read(b); err != nil {
    return time.Now().Format("20060102150405.000000000")
}
```

OS 正常情况下 crypto/rand 不会失败(只有极端场景如 fd 耗尽)。强制触发需要:
- 关闭 /dev/urandom(破坏性,不可行)
- mock crypto/rand(包级函数,Go 不支持)

**决策**:接受 75% 覆盖,logging 整体 98% 远超 90% 目标。fallback 路径在生产环境也是 best-effort,不影响主流程。

## 实测覆盖率

```bash
$ go test -count=1 -cover ./internal/logging/...
ok  github.com/iannil/pinconsole/internal/logging  0.691s  coverage: 98.0% of statements
```

| 函数 | 切片前 | 切片后 |
|---|---|---|
| NewLogger | 0% | 100% |
| parseLevel | 100% | 100% |
| newID | 75% | 75%(fallback 不覆盖) |
| WithTraceID/WithSpanID/TraceID/SpanID | 100% | 100% |
| FromContext | 100% | 100% |
| TraceMiddleware | 100% | 100% |

## 副作用与回归

- ✅ 不改业务代码
- ✅ 现有测试不破坏(每测试 defer 恢复 stdout + slog.Default)
- ✅ 不依赖外部基础设施(纯单测)

## 同步文档

- ✅ project-status §2.1 logging 行更新 79.6%→98.0%
- ✅ daily 追加 Go-2 段

## 后续切片解锁

Go-3(hub)可启动,无依赖。
