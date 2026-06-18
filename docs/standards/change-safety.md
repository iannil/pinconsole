# 变更安全策略

> CLAUDE.md "面向大模型的可改写性" §7 要求:批量程序性改动前先将原文件备份至 `/backup`;若错误数异常上升,立即回滚备份。
> 本文档落地该原则,给出可操作流程。

## 适用范围

- **批量程序性改动**:跨多文件的重命名、模式替换、import 路径迁移、API 签名变更
- **不适用**:单文件局部修改、新文件创建、明确小范围 PR

## 流程

### Step 1:备份

批量改动前,把受影响文件复制到 `/backup/{YYYY-MM-DD}/{description}/`:

```bash
BACKUP_DIR="backup/$(date +%Y-%m-%d)/${DESCRIPTION}"
mkdir -p "$BACKUP_DIR"
# 备份受影响文件(保持原相对路径)
for f in server/internal/api/*.go admin/src/components/*.vue; do
  mkdir -p "$BACKUP_DIR/$(dirname "$f")"
  cp "$f" "$BACKUP_DIR/$f"
done
```

### Step 2:执行改动

按计划修改文件。每步可独立验证(go build / pnpm build)。

### Step 3:验证 + 决策

| 指标 | 阈值 | 决策 |
|---|---|---|
| `go build` 错误数 | > 改动前 + 5 | **立即回滚**(`cp -r $BACKUP_DIR/* .`) |
| `go test` 失败数 | > 改动前 + 2 | **立即回滚** |
| `pnpm build` 错误数 | > 改动前 + 3 | **立即回滚** |
| 单测覆盖率 | < 改动前 - 5% | **警告** + 人工评估 |

### Step 4:完成 + 清理

- 改动验证通过 → 提交 commit(message 引用备份目录便于追溯)
- 1 周后(或下个稳定 tag 后)清理 `/backup/{date}/` 目录
- 重要变更(架构级)可永久保留备份,作为人工回滚兜底

## 废弃函数处理

代码改动中产生的"已废弃但仍保留"的函数(如 `Deprecated` 注释的):

1. **立即标记**:加 `// Deprecated: 用 xxx 替代` 注释 + 调用方迁移
2. **删除窗口**:2 周内未被任何代码引用 → 直接删除
3. **超期保留**:若超过 2 周仍存在,在 audit 中点名(违反"变更安全策略")

## 反例(违反本策略)

- 1p 之前的 `NewRouter`:标 Deprecated 但保留 ≥ 2 周,未被使用也未删
- 1p 之前的 `isFullSnapshotEnvelope`:注释"已废弃"但函数体保留,无删除时间表
- 1p 之前的 `pingEvery`:struct 字段赋值但全代码 0 引用
- 1p 之前的 `Envelope.TraceID`:schema 字段定义但 server 内 0 调用(1m 已修)

## 工具

未来可在 pre-commit hook 加检测:
- `grep -r "Deprecated:" --include="*.go"` 列出所有 deprecated,提醒清理
- `git log --since="2 weeks ago" --grep="Deprecated"` 追踪新 deprecated

## 历史

- **2026-06-18**:本文档建立(1p-llm-friendly 切片)
- 历史反例(见上)在 1p/1m 中部分清理,剩余在 follow-up
