# 进度文档模板

> 复制此模板到 `docs/progress/{YYYY-MM-DD}-{kebab-case-name}.md`，填入实际内容。
> 完成后移到 `docs/reports/completed/`（保留文件名），并将状态字段改为 "completed"。

---

# {标题}

**状态**：in_progress | blocked | completed
**开始**：YYYY-MM-DD
**完成**：（未完成时空着）
**关联**：[PLAN.md 章节](../../PLAN.md#{anchor}) 或其他上下文链接

## Context

为什么做这件事？要解决什么问题？触发原因是什么？
（一段话，不超过 5 行。让 LLM 能在 30 秒内判断是否相关。）

## Changes

实际做了什么？按可观察的修改单元列。每条链接到具体文件或文件清单。

- [ ] {修改单元 1}（涉及 `path/to/file`）
- [ ] {修改单元 2}（涉及 `path/to/another`）
- [ ] ...

## Status

当前状态：{一句话总结}

剩余工作：
- {待办 1}
- {待办 2}

## Next

下一步要做什么？指向下一个 progress 文档或具体动作。

## Blockers

被什么阻塞？需要谁决策？（无阻塞则写"无"。）

## Notes

补充信息：发现的边界 case、决策依据链接、未来要回头看的点。（可选。）
