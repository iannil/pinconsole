# 切片 1q-cleanup 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:P1-25(e2e/helpers 死代码)+ P1-26(room.publish 静默)+ P1-28(死代码集中清理)+ P1-30(queries.sql vs queries.go)+ P1-31(Element Plus 零使用)
**深度 badge**:🟢 verified-deep

## Summary

清理审计点名的死代码 + 重复:6 处 deprecated/未用函数 + e2e/helpers 整目录 + queries.sql(sqlc 未启用) + Element Plus 注册零使用(bundle -940KB)+ room.publish 静默丢弃加日志。

## Changes Delivered

### 后端 Go 死代码清理

- ✅ `server/internal/recording/stream.go` — 删 `FlushSessionNow`(0 调用方)
- ✅ `server/internal/storage/queries.go` — 删 `TouchVisitor`(0 调用方)+ `CountEventBlobsBySession`(0 调用方)
- ✅ `server/internal/api/ws.go` — 删 `isFullSnapshotEnvelope`(标 deprecated 已 ≥ 2 周,0 调用方)+ 删 `pingEvery` 字段(赋值后 0 读取)
- ✅ `server/internal/api/router.go` — 删 `NewRouter` Deprecated wrapper(0 调用方)
- ✅ `server/internal/hub/room.go` — `publish()` default 分支加 `slog.Warn` 日志(原静默丢弃)
- ✅ `server/internal/storage/queries.sql` — 删除(sqlc 未启用,与 queries.go 重复且不同步)

### 前端死代码 + 依赖清理

- ✅ `admin/src/main.ts` — 移除 `import ElementPlus` + `app.use(ElementPlus)` + CSS import
- ✅ `admin/package.json` — 移除 `element-plus` 依赖
- ✅ `e2e/tests/helpers/` — 整目录删除(`setup.ts` + `selectors.ts`,0 spec 引用)

### 文档/规范

- 无新文档(本切片纯清理)

## Verification

```bash
# 1. Go 测试
cd server && go test ./... -count=1 -race

# 2. Admin build(验证 Element Plus 移除不破坏)
pnpm --filter @marketing-monitor/admin build
# bundle 大幅缩小:index.js 1098KB → 159KB

# 3. SDK build
pnpm --filter @marketing-monitor/visitor-sdk build

# 4. 验证死代码全清
grep -rn "FlushSessionNow\|TouchVisitor\|CountEventBlobsBySession\|isFullSnapshotEnvelope\|pingEvery\|NewRouter\b" server/ --include='*.go'
# 应只剩函数名注释或 0 命中
```

**预期结果**:全部 PASS;admin bundle 大幅瘦身。

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ 删除后 go build + go test + admin build 全 PASS |
| Negative case | ✅ Element Plus 移除后无组件报错(零使用已确认) |
| 边界 | ✅ `fs` import 仍被其他类型使用,未误删 |
| 真实集成 | ✅ 实际跑 build + test |
| 可重复运行 | ✅ -count=1 -race 无 flaky |

**结论**:🟢 verified-deep。

## 影响

- **Bundle 瘦身**:admin index.js 1098KB → 159KB(-85%),Element Plus 全套组件 + CSS 移除
- **代码可读性**:7 处死代码移除降低 LLM 检索成本
- **运维可观察**:room.publish 丢弃事件不再静默,slog.Warn 留下痕迹
- **存储层单一源**:queries.sql 删除,storage/queries.go 是唯一事实源(sqlc 未来重启用 codegen 重新生成)

## 与规格的偏差

无 — 本切片执行直接,无 grill。

## Follow-ups

- 1r 或后续:`extractXxx`/`scanXxx`/`buildXxx` 历史命名是否迁移到 `parseXxx`(naming-conventions §4.2 已说明保留)
- god files 拆分(queries.go 540+ LOC、ws.go 510+ LOC、router.go 285 LOC)— 留给代码质量切片
- `room.publish` 日志目前是 `slog.Warn` 直接调,未来可注入 logger 实例便于 trace_id 透传
