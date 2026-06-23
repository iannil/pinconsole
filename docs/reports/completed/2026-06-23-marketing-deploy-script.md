# Marketing 一键部署脚本（Cloudflare Pages 生产）

**状态**：completed
**完成时间**：2026-06-23
**深度 badge**：🟡 verified-shallow

## Summary

把 `marketing/` 此前散落在 `package.json` + README 5 步里的 Cloudflare Pages 生产部署流程，收口到单一可执行脚本 `marketing/scripts/deploy.sh`。脚本覆盖：预检（node/pnpm/wrangler 登录/wrangler.toml database_id/D1 远程可达）→ D1 schema 应用（幂等）→ secrets 交互式检查 → build → `wrangler pages deploy` → 打印 deployment URL + 后续提示。子命令：`deploy`（默认）/`check`/`db`/`secrets`/`build`/`--help`。

同时修复一个潜在的 prod 数据丢失 bug：原 D1 migration `0001-create-leads.sql` 使用 `DROP TABLE IF EXISTS leads; CREATE TABLE ...`，在生产环境重跑会抹掉所有真实 leads。改为 `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`，幂等且零数据损失。

## Changes Delivered

### 新增

- ✅ `marketing/scripts/deploy.sh`（新建，executable）
  - `cmd_check`：node/pnpm/wrangler login/wrangler.toml database_id/D1 远程连接预检
  - `cmd_db`：检查远程 `leads` 表存在性，存在则跳过；不存在则 `wrangler d1 execute --remote --file`
  - `cmd_secrets`：`wrangler pages secret list` 列已配置项，对每个 `RESEND_API_KEY/LEAD_NOTIFY_EMAIL/TURNSTILE_SECRET` 缺失项交互式 `wrangler pages secret put`
  - `cmd_build`：`pnpm build`
  - `cmd_deploy`（默认入口）：`check → db → secrets 提示 → build → pages deploy`，收尾抓 `*.pages.dev` URL + dashboard 链接 + leads 查询命令 + 自定义域名提示
  - `show_help`：彩色 ANSI 帮助
  - 颜色 pattern 与 `ops.sh` 一致（`CYAN='\033[0;36m'` + `echo -e`）

### 修复

- ✅ `marketing/migrations/0001-create-leads.sql`
  - 移除 `DROP TABLE IF EXISTS leads;`
  - `CREATE TABLE leads (...)` → `CREATE TABLE IF NOT EXISTS leads (...)`
  - `CREATE INDEX idx_leads_status ON leads(status);` → `CREATE INDEX IF NOT EXISTS ...`
  - `CREATE INDEX idx_leads_created_at ON leads(created_at DESC);` → 同上
  - 顶部注释加 "Idempotent: safe to re-run on existing prod DB"

### 更新

- ✅ `marketing/package.json`
  - `deploy` 改为 `./scripts/deploy.sh deploy`（此前是 `pnpm build && wrangler pages deploy ./dist`，缺预检 + 缺 D1 检查）
  - 新增 `deploy:check` / `deploy:secrets` / `deploy:db` 三个 pnpm 入口
  - 保留原 `db:apply` / `db:apply:remote`（手动场景仍有用）

- ✅ `marketing/README.md`
  - 删掉旧的 5 步"Cloudflare setup (first time)"段落（步骤 4「Deploy」单独一条 pnpm deploy，缺少 D1 schema 应用；步骤 3 secret 命令 `LEAD_NOTIFY_WEBHOOK` 已过时，实际 secrets 是 `RESEND_API_KEY/LEAD_NOTIFY_EMAIL/TURNSTILE_SECRET`）
  - 替换为「首次设置（3 步）」+「日常部署（一键）」两段
  - 日常部署同时列 `./scripts/deploy.sh` 子命令和 `pnpm deploy:*` 等价入口
  - 显式说明 `deploy.sh` 不读 `.env`，所有敏感配置走 Cloudflare Pages secrets

## Verification

- ✅ `bash -n deploy.sh` 语法通过
- ✅ `./scripts/deploy.sh --help` 渲染正确（ANSI 颜色 + 中文 + 表格对齐）
- ✅ `./scripts/deploy.sh check` 在真实生产账号下端到端跑通：
  - Cloudflare 账号 `zhurongx@gmail.com` / Account `ZHURONG` 已识别
  - `database_id = 17891dd6…a511` 已配置
  - D1 远程连接 OK
- ✅ 修复了一个 heredoc 边界 bug：`$D1_DB_NAME）`（全角右括号紧跟变量名）导致 bash 把多字节字符当变量名一部分，报 unbound variable。改用 `${D1_DB_NAME}` 显式定界。
- ✅ 修复了一个 heredoc 颜色 bug：`cat <<EOF` 不解释 `\033`，输出字面 `\033[0;36m`。改为 `text=$(cat <<EOF...); echo -e "$text"` 两段式。
- ✅ `package.json` JSON 合法性校验通过

## 决策与取舍

- **未自动跑 `wrangler pages deploy`**：用户原话是"更新脚本"，不是"部署"。脚本的 `check` 子命令足以验证可用性；实际 deploy 留给用户在准备好窗口时显式触发（生产环境副作用大）。
- **secrets 子命令不强制阻断 deploy**：`/api/leads` 即使没有邮件 secret 也会入库（D1 INSERT 是主路径），邮件只是通知。脚本在 deploy 流程中只提示，让用户决定是否中断。
- **未引入 dotenv 读取**：`.env` 是 server（Go）那一侧的约定，marketing 走 Cloudflare Pages secrets 体系更原生，避免双轨。
- **保留 `db:apply` / `db:apply:remote` 两个老脚本**：手动 debug D1 时更直接（不经过 deploy.sh 的预检）。新增 `deploy:db` 是部署视角，老的 `db:apply*` 是开发视角，职责不同。
- **migration 改成幂等**：原 SQL 的 `DROP+CREATE` 在 dev 环境便于 reset，但在 prod 是定时炸弹。Claude 没有选择"加新 migration 0002"而是直接改 0001，理由：项目仍在 v1 阶段，prod 从未真正部署过（database_id 是 2026-06-22 才创建），不存在"已经跑过 0001 的 prod 环境"需要保持兼容的对象。

## 后续

- 用户首次执行 `./scripts/deploy.sh`（无参数）即可完成生产首部署
- 自定义域名 `pinconsole.com` 需在 Cloudflare Pages dashboard 手动绑定（脚本只在收尾打印提示）
- 若需要 lead 通知邮件，先 `./scripts/deploy.sh secrets` 配置 `RESEND_API_KEY` + `LEAD_NOTIFY_EMAIL`
