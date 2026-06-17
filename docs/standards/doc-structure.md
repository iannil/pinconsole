# 文档结构规范

> 本文件落地 [`CLAUDE.md`](../../CLAUDE.md) 中"项目指南 → 文档约定"小节。任何对仓库文档的修改都应遵循此规范。

## 1. 目录布局

```
docs/
├── README.md                # 文档索引（必读，所有子文档一句话索引）
├── project-status.md        # rolling 项目状态快照（LLM 友好）
├── progress/                # 进行中的修改（一改一文）
├── reports/
│   └── completed/           # 已完成的修改报告
├── audits/                  # 审计发现（冗余/过期/错误梳理）
├── standards/               # 规范（命名、结构、流程）
└── templates/               # 各类文档模板
```

## 2. 各子目录用途

| 目录 | 用途 | 何时写 |
|---|---|---|
| `progress/` | 记录"进行中"的修改 | 开始一项非平凡修改时立即创建；每次有进展追加 |
| `reports/completed/` | 标记"已交付"的修改 + 验收证据 | 修改完成且通过验收后；与对应 progress 文件同名 |
| `audits/` | 审计发现的冗余/过期/错误清单 + 处理方案 | 周期性审计、重大重构前、用户要求时 |
| `standards/` | 落地 CLAUDE.md 约定的可执行规范 | 约定稳定后；约定变化时更新 |
| `templates/` | 各类文档的标准模板 | 引入新文档类型时 |

**注**：`docs/reports/`（顶层，非 `completed/` 子目录）保留给"验收"类报告——对一组已完成修改的整体验收。

## 3. 文件命名规范

详细规则见 [`naming-conventions.md`](./naming-conventions.md)。

简表：

- 时间戳文档（progress / completed / audits）：`{YYYY-MM-DD}-{kebab-case-name}.md`
- Rolling 文档（status / standards / templates）：`{kebab-case-name}.md`
- 中划线分隔单词，全小写，不用下划线、空格、中文文件名

## 4. 文档生命周期

```
progress/{date}-{name}.md
  ↓ 修改完成 + 验收通过
reports/completed/{date}-{name}.md （与 progress 同名，引用 progress）
  ↓
（progress 文件保留，作为决策上下文的历史档案；不删除）
```

如果修改被放弃（未完成），progress 文件保留并在 frontmatter 状态字段写 `abandoned`，加注放弃原因。

## 5. LLM 友好原则

每份文档应满足：

1. **头部摘要**：开篇 30 秒能判断相关性（Context 节不超过 5 行）
2. **显式状态**：用固定字段（`状态`、`开始`、`完成`）而非隐式暗示
3. **可追溯链接**：所有引用用 markdown 链接（`[text](path)`），不要写"参见某文档"
4. **结构稳定**：固定章节顺序，让 LLM 能按结构化方式提取
5. **修改单元粒度**：每个 Change 单元对应可独立观察的文件改动

## 6. 语言约定

- **文档语言**：中文（per [`CLAUDE.md`](../../CLAUDE.md) "项目指南 → 语言约定"）
- **代码块语言**：保留英文（命令、代码、变量名）
- **跨文档引用**：用相对路径，例如 `[PLAN.md](../../PLAN.md)`
