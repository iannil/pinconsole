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

## Cloudflare setup (first time)

1. **Create D1 database**:
   ```bash
   cd marketing
   wrangler d1 create pinconsole-leads
   # paste the returned database_id into wrangler.toml
   ```

2. **Apply schema**:
   ```bash
   wrangler d1 execute pinconsole-leads --local --file ./migrations/0001-create-leads.sql
   wrangler d1 execute pinconsole-leads --remote --file ./migrations/0001-create-leads.sql
   ```

3. **(Optional) Set notification secrets**:
   ```bash
   wrangler secret put LEAD_NOTIFY_WEBHOOK   # enterprise-wechat/slack webhook
   ```

4. **Deploy**:
   ```bash
   pnpm deploy
   # or: wrangler pages deploy ./dist
   ```

5. **Bind custom domain** via Cloudflare Pages dashboard (add the route in `wrangler.toml` first).

## Reading leads

```bash
wrangler d1 execute pinconsole-leads --remote --command "SELECT id, name, company, contact, purpose, created_at FROM leads WHERE status = 'new' ORDER BY created_at DESC LIMIT 50;"
```

A simple admin viewer is intentionally not provided — maintainers should query D1 directly via wrangler. If lead volume grows, build a separate internal tool.

## License

AGPL-3.0-or-later — same as the main repo. See root `LICENSE`.
