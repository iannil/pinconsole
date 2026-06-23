# @pinconsole/marketing

> Maintainer marketing site for pinconsole. Not part of the OSS distribution — deployed independently on Cloudflare.

## What's here

- **Astro + Vue static landing page** with bilingual (zh/en) content
- **Cloudflare Worker endpoint** for lead form submission (`POST /api/leads`)
- **Cloudflare D1** for lead storage
- **Calm Crafted marketing variant** of `docs/design-system.md`

## What's NOT here

- The OSS product itself — see root `README.md`
- Admin SPA, visitor SDK, server — those are at `../admin/`, `../visitor-sdk/`, `../server/`
- Public demo environment — intentionally not provided (abuse risk)

## Why a separate directory

`marketing/` is **maintainer-only** infrastructure. It deploys to `pinconsole.example.com` (or maintainer's own domain) and exists to harvest ToB consultation leads. OSS users who self-host pinconsole should not need any of this code — that's why `landing/` (the OSS template) and `marketing/` (this directory) are separate.

## Setup

```bash
# from repo root
pnpm install

# local dev (Astro)
pnpm --filter @pinconsole/marketing dev

# build + preview with wrangler (Cloudflare Workers runtime)
pnpm --filter @pinconsole/marketing build
pnpm --filter @pinconsole/marketing preview
```

## Cloudflare 首次设置（一次性，3 步）

1. **登录 Cloudflare + 创建 D1**:
   ```bash
   cd marketing
   pnpm exec wrangler login
   pnpm exec wrangler d1 create pinconsole-leads
   # 把返回的 database_id 粘进 wrangler.toml
   ```

2. **设置生产 secrets（交互式）**:
   ```bash
   ./scripts/deploy.sh secrets
   # 会引导设置 RESEND_API_KEY / LEAD_NOTIFY_EMAIL / TURNSTILE_SECRET
   ```

3. **绑定自定义域名**:
   Cloudflare Pages dashboard → `pinconsole` → Custom domains → Set up `pinconsole.com`
   （首次需在 Cloudflare DNS 加 CNAME）

## 日常部署（一键）

```bash
cd marketing
./scripts/deploy.sh            # 完整流程：预检 → D1 schema → build → deploy
./scripts/deploy.sh check      # 仅预检（不部署）
./scripts/deploy.sh db         # 仅把 D1 schema 应用到远程（幂等）
./scripts/deploy.sh secrets    # 交互式检查 / 设置 secrets
./scripts/deploy.sh --help     # 全部命令
```

等价的 pnpm 入口：

```bash
pnpm deploy              # = ./scripts/deploy.sh deploy
pnpm deploy:check        # 预检
pnpm deploy:secrets      # 设置 secrets
pnpm deploy:db           # D1 schema 应用到远程
```

`deploy.sh` 做的事：
- 预检 `wrangler login` / `wrangler.toml` 的 `database_id` / D1 远程可达
- 远程 `leads` 表不存在则应用 migration（**幂等**，存在则跳过，**绝不** DROP）
- `pnpm build` → `wrangler pages deploy ./dist`
- 收尾打印 deployment URL + leads 查询命令 + 自定义域名提示

## Reading leads

```bash
wrangler d1 execute pinconsole-leads --remote --command "SELECT id, name, company, contact, purpose, created_at FROM leads WHERE status = 'new' ORDER BY created_at DESC LIMIT 50;"
```

A simple admin viewer is intentionally not provided — maintainers should query D1 directly via wrangler. If lead volume grows, build a separate internal tool.

## License

AGPL-3.0-or-later — same as the main repo. See root `LICENSE`.
