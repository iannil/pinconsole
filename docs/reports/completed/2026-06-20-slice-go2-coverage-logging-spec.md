# Slice Go-2 spec:logging 包补测

> **状态**:completed
> **工时预估**:4h | **实际工时**:~1h
> **深度 badge**:🟢 touched(纯单测 + stdout 捕获)

## 范围

| 包 | 切片前 | 切片后 | 提升 |
|---|---|---|---|
| internal/logging | 79.6% | **98.0%** | +18.4pp |

## 主要工作

新建 `server/internal/logging/handler_extra_test.go`(8 测试):
- NewLogger dev/prod/test 模式分别用 TextHandler/JSONHandler
- 不同 level(debug/info) 屏蔽 Debug 日志
- AddSource 仅 dev 模式
- slog.SetDefault 副作用
- NewLogger + FromContext 协同

## 不覆盖

- `newID` 75%:crypto/rand.Read 失败 fallback 路径几乎不可能触发(OS 正常情况下),不值得 hack 触发

## 验证

```bash
go test -count=1 -cover ./internal/logging/...
# ok  github.com/iannil/pinconsole/internal/logging  coverage: 98.0% of statements
```
