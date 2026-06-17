# 完成报告模板

> 复制此模板到 `docs/reports/completed/{YYYY-MM-DD}-{kebab-case-name}.md`（与对应 progress 文件同名），填入实际内容。
> 完成报告 = 对一个变更的"已交付"声明 + 验收证据 + 后续追踪项。

---

# {标题}

**状态**：completed
**完成时间**：YYYY-MM-DD
**对应 progress**：[docs/progress/{同名}.md](../../progress/{同名}.md)

## Summary

一句话：做了什么，产生了什么事实？（让 LLM 能据此判断要不要读全文。）

## Changes Delivered

实际交付的修改单元，每个带文件路径与一句话描述。

- ✅ {修改单元 1} — `path/to/file`：{描述}
- ✅ {修改单元 2} — `path/to/another`：{描述}
- ✅ ...

## Verification

如何验证变更确实生效？必须是可重复执行的检查命令或可观察的行为。

```bash
# 验证命令 1
{command}

# 验证命令 2
{command}
```

**预期结果**：
- {预期 1}
- {预期 2}

## Follow-ups

本次未覆盖、但相关的后续工作（链接到新建的 progress 或 issue）。

- {后续 1} → {链接或下一步}
- {后续 2} → ...

## Notes

补充信息：决策依据、边界 case、风险记录。（可选。）
