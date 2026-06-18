# 命名规范

> 本文件落地 [`CLAUDE.md`](../../CLAUDE.md) 中"项目指南 → 面向大模型的可改写性"小节的命名要求。

## 1. 目录命名

- 全部 kebab-case：`docs/reports/completed/`，**不**用 `Completed/`、`completed-reports/`、`completed_reports/`
- 不用缩写除非已是行业通用：`api` 可以，`cfg` 不行（用 `config`）
- 单数 vs 复数：用复数表"集合"（`reports/`、`templates/`），单数表"单一概念"（`progress/`、`status/`）

## 2. Markdown 文件命名

### 时间戳文档（progress / completed / audits）

```
{YYYY-MM-DD}-{kebab-case-name}.md
```

- 日期是事件触发日期（progress 是开始日期，completed 是完成日期）
- name 是 kebab-case，简短描述修改单元
- 例：`2026-06-17-v1-slice-plan.md`、`2026-06-17-initial-cleanup.md`

### Rolling 文档（status / standards / templates）

```
{kebab-case-name}.md
```

- 例：`project-status.md`、`doc-structure.md`、`progress.md`

### 禁止

- ❌ 中文文件名（不便于 shell 操作）
- ❌ 下划线（与代码标识符冲突）
- ❌ 空格（需转义）
- ❌ 大写字母（与 Unix 大小写敏感问题）
- ❌ 版本号在文件名（用 git 历史而非 `v2`）

## 3. 代码命名

详见 [`CLAUDE.md`](../../CLAUDE.md) "项目指南 → 面向大模型的可改写性 → 可搜索性"。简表：

| 类型 | 规范 | 示例 |
|---|---|---|
| Go 文件 | snake_case | `session_manager.go` |
| Go 包名 | single word lowercase | `auth`、`hub` |
| Go 类型 | PascalCase | `HubServer`、`SessionManager` |
| TypeScript 文件 | kebab-case | `session-manager.ts` |
| TypeScript 类型 | PascalCase | `SessionManager`、`HubClient` |
| 常量 | UPPER_SNAKE 或 PascalCase | `MAX_CONNECTIONS`（Go）、`MaxConnections`（TS） |
| CSS 类 | kebab-case | `.visitor-card` |
| Vue 组件 | PascalCase | `VisitorCard.vue` |

## 4. 函数命名约定（增强 LLM 检索）

为降低人类与 LLM 的检索成本，以下函数前缀**保持一致**：

| 前缀 | 含义 | 示例 |
|---|---|---|
| `parseXxx` | 字符串 → 结构化对象 | `parseSessionToken` |
| `safeXxx` | 安全版本（带错误处理） | `safeJsonParse`、`safeQuerySelector` |
| `assertXxx` | 断言（失败 panic/throw） | `assertNever`、`assertTenant` |
| `createXxx` | 工厂 | `createHubServer`、`createSession` |
| `withXxx` | 装饰器 / 上下文 | `withTraceId`、`withTenant` |
| `isXxx` / `hasXxx` | 谓词 | `isAuthenticated`、`hasScope` |
| `mustXxx` | 强制成功版（panic on error） | `mustConnect` |

### 4.1 语言惯例差异(1p 同步)

不同语言生态有不同的工厂函数惯例,**两者都正确**,不要强行统一:

| 语言 | 工厂惯例 | 示例 |
|---|---|---|
| **Go** | `NewXxx` (PascalCase) | `NewHub()`、`NewClient()`、`NewAuthHandler()` |
| **TypeScript** | `createXxx` (camelCase) | `createRouter()`、`createPinia()`、`createStore()` |

Go 端保留 `New*`(标准库惯例 + Go community 一致);TS 端用 `create*`(Vue/Pinia/Router 一致)。

### 4.2 当前代码与规范的差异(1p 标记)

- Go 端:多数函数用 `extractXxx` / `scanXxx` / `buildXxx`,与 `parseXxx` / `safeXxx` 不完全对齐
- 这部分是 1b/1c 时建立的模式,迁移成本高于收益,**保留**
- 新代码建议优先用本表前缀(尤其纯函数)

## 5. Git 分支命名

| 类型 | 前缀 | 示例 |
|---|---|---|
| 切片 | `slice/` | `slice/1a-skeleton` |
| 修复 | `fix/` | `fix/ws-reconnect` |
| 文档 | `docs/` | `docs/initial-reorg` |
| 重构 | `refactor/` | `refactor/hub-routing` |
| 实验 | `exp/` | `exp/sqlc-vs-gorm` |

## 6. 提交信息

Conventional Commits：

```
{type}({scope}): {subject}

{body}

{footer}
```

`type`：`feat` / `fix` / `docs` / `refactor` / `chore` / `test` / `perf` / `build` / `ci`

例：`docs(slice-1a): add project status snapshot and progress index`
