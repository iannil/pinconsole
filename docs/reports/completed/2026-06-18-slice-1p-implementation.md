# 切片 1p-llm-friendly 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:LLM friendly 维度 P0(P1-17/18/19)+ P1(P1-20)
**深度 badge**:🟢 verified-deep
**叙述免责**:基于实施时状态。

## Summary

补齐 CLAUDE.md "面向大模型的可改写性" 欠债:消除 admin/SDK proto 手写同步重复(改共享 workspace 包)+ 建 IMPLEMENTATION_PLAN.md(CLAUDE.md 第 39 行要求)+ 建 docs/standards/change-safety.md(原则 7 落地)+ naming-conventions.md 加语言惯例差异说明。审计 LLM friendly 评分 5.4/10 → 预估 8/10。

## Changes Delivered

### 三方 proto 共享(消除手写同步重复)

- ✅ `packages/proto/`(新建 workspace package)— `@pinconsole/proto`,含 envelope.ts / events.ts / command.ts(原 admin/src/proto + visitor-sdk/src/proto 合并)
- ✅ `packages/proto/package.json` — workspace package 配置,exports map
- ✅ `packages/proto/tsconfig.json` — 类型检查
- ✅ `pnpm-workspace.yaml` — 加 `packages/*`
- ✅ `admin/package.json` + `visitor-sdk/package.json` — 加 `@pinconsole/proto: workspace:*` dep
- ✅ 删除 `admin/src/proto/` + `visitor-sdk/src/proto/`(原 4 个文件,2 份重复)
- ✅ 重写 14 处 import:`'../proto/envelope'` → `'@pinconsole/proto'`(admin 3 处 + SDK 11 处)
- ✅ admin 现在也能 import command.ts(原 admin 缺,SDK 有)

### IMPLEMENTATION_PLAN.md(CLAUDE.md 要求)

- ✅ `IMPLEMENTATION_PLAN.md`(新建项目根)— rolling 当前状态 + 已交付切片 + 下一步候选 + 决策原则

### change-safety.md(原则 7 落地)

- ✅ `docs/standards/change-safety.md`(新建)— 备份流程 + 阈值表(build/test 错误数) + 废弃函数处理 + 反例(NewRouter / isFullSnapshotEnvelope / pingEvery / Envelope.TraceID)

### naming-conventions.md 同步

- ✅ 加 §4.1 语言惯例差异(Go `New*` vs TS `create*`,两者都正确)
- ✅ 加 §4.2 当前代码与规范的差异(`extractXxx`/`scanXxx`/`buildXxx` 保留,新代码优先用规范前缀)

## Verification

```bash
# 1. workspace dep 已 wire
pnpm install  # 应成功,symlink 创建

# 2. SDK + Admin 编译(共享包被正确解析)
pnpm --filter @pinconsole/visitor-sdk build
pnpm --filter @pinconsole/admin build

# 3. Go 测试不受影响
cd server && go test ./... -count=1

# 4. 验证 admin 现在能 import command 类型
grep "from '@pinconsole/proto'" admin/src/composables/useWs.ts
```

**预期结果**:全部 build + test 通过,SDK + Admin 共享同一份 proto 源。

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ SDK + Admin build 都通过 |
| Negative case | ✅ 删除旧 proto/ 后无残留 import |
| 边界 | ✅ workspace:* 在 pnpm 10.x 下正确解析 |
| 真实集成 | ✅ 实际跑 pnpm install + build,非 mock |
| 可重复运行 | ✅ 多次 build 结果一致 |

**结论**:🟢 verified-deep。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| 未引入 codegen(buf/quicktype) | 协议仍在演进,codegen 是更大架构决策;本切片先消除手写重复 |
| Go 端 proto 不共享 | Go 与 TS 类型系统不同,无法直接共享;Go 端 `internal/proto/` 已是单一源 |
| `/backup` 目录未建立物理目录 | change-safety.md 文档化流程即可,目录按需创建 |

## Follow-ups

- 引入 codegen(协议稳定后):buf/quicktype 从单一 schema 生成 Go + TS,消除语言间漂移
- 历史 `extractXxx`/`scanXxx` 函数评估是否逐步迁移到 `parseXxx`(成本评估)
- `/backup/` 在批量改动时实际使用(下一次跨文件重构验证流程)
- admin 端未利用共享 command.ts 的能力(原 admin 缺),后续切片可加 admin 端命令编辑器

## Notes

- packages/proto 的 `exports` map 允许 `@pinconsole/proto` / `.../envelope` / `.../events` / `.../command` 多入口;主入口 re-export 全部
- Go 端的 internal/proto 和 TS 端的 packages/proto 仍是两份源,需手工保持字段一致;1m 已通过 Envelope.TraceID 接线降低漂移风险
- LLM friendly 评分提升主要来自:三方 proto 单一源(原 P0)+ IMPLEMENTATION_PLAN.md 存在(原 P0)+ change-safety 文档化(原 P0)
