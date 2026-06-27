# · pinconsole

> **Your visitors, your data.** · [中文](./README.zh.md)

**Open-source self-hosted alternative to FullStory, Hotjar, LogRocket, and Smartlook** — real-time visitor monitoring, co-browsing, and session replay. AGPL-3.0, data never leaves your infra.

[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-0F766E.svg)](./LICENSE)
[![e2e: 70 passed](https://img.shields.io/badge/e2e-70%20passed%20%2F%200%20failed-15803D.svg)](./docs/reports/completed/2026-06-18-v1-e2e-acceptance.md)
[![v1: shipped](https://img.shields.io/badge/v1-shipped-0F766E.svg)](./docs/project-status.md)
[![i18n: zh/en](https://img.shields.io/badge/i18n-zh%20%2F%20en-0E7490.svg)](#)

[![vs FullStory](https://img.shields.io/badge/vs_FullStory-see_comparison-0F766E)](https://pinconsole.com/alternatives/fullstory/)
[![vs Hotjar](https://img.shields.io/badge/vs_Hotjar-see_comparison-0F766E)](https://pinconsole.com/alternatives/hotjar/)
[![vs LogRocket](https://img.shields.io/badge/vs_LogRocket-see_comparison-0F766E)](https://pinconsole.com/alternatives/logrocket/)
[![vs Smartlook](https://img.shields.io/badge/vs_Smartlook-see_comparison-0F766E)](https://pinconsole.com/alternatives/smartlook/)
[![vs Cobrowse.io](https://img.shields.io/badge/vs_Cobrowse.io-see_comparison-0F766E)](https://pinconsole.com/alternatives/cobrowse-io/)

Design decisions:

- **Data sovereignty** — every visitor action, operator chat, and session recording lives in **your** PostgreSQL / Redis / MinIO. No third-party calls, no external dependencies.
- **AGPL-3.0 strong copyleft** — every modification must be open-sourced. Cloud vendors cannot take this and re-sell it as SaaS. License-level hard protection.
- **Standard stack, no lock-in** — Go 1.22 + Vue 3 + PostgreSQL 16 + Redis 7 + MinIO. Industry-standard at every layer. The schema is yours.
- **~15KB gzip SDK** — lightweight visitor SDK, served from your own domain. No CDN dependency, no third-party script loaders.

**Decision-maker?** → [pinconsole.com](https://pinconsole.com) · **Engineer?** → [Self-host](#self-host) · **Compare?** → [FullStory](https://pinconsole.com/alternatives/fullstory/) · [Hotjar](https://pinconsole.com/alternatives/hotjar/) · [LogRocket](https://pinconsole.com/alternatives/logrocket/) · [Smartlook](https://pinconsole.com/alternatives/smartlook/)

---

## Who is this for?

- **Product teams** that need session replay without per-session pricing or data leaving their infrastructure
- **Customer support teams** that want co-browsing capabilities without third-party SaaS subscriptions
- **Compliance-conscious organizations** (GDPR, SOC 2, HIPAA, China PIPL) that need data to stay in-region or on-premise
- **Developer teams** looking for an open-source, auditable alternative to proprietary session replay tools
- **Chinese market teams** that need session replay and co-browsing without cross-border network latency

---

## See it

![Dashboard — real-time visitor monitoring](./marketing/public/screenshots/dashboard.png)

![Co-browsing + live chat](./marketing/public/screenshots/cobrowse-active.png)

---

## Why this exists

SaaS visitor-engagement tools extract every visitor's behavior, every operator chat, every session recording into their cloud. You pay them to ship your data to their infra — they use it to train their models, lock you in, and price-up next year.

pinconsole exists because that data is yours. Run it on your own infra. Audit the code yourself. Leave when you want — your data, schema, and binaries are already in your hands.

**Built as an open-source alternative to:**

| Tool | Why switch |
|---|---|
| [FullStory](https://pinconsole.com/alternatives/fullstory/) | SaaS-only, $599+/month, no co-browsing |
| [Hotjar](https://pinconsole.com/alternatives/hotjar/) | Session caps (35/day free), SaaS-only, no real-time |
| [LogRocket](https://pinconsole.com/alternatives/logrocket/) | Per-session pricing, SaaS-only, no co-browsing |
| [Smartlook](https://pinconsole.com/alternatives/smartlook/) | Session quotas, SaaS-only, no co-browsing |
| [Cobrowse.io](https://pinconsole.com/alternatives/cobrowse-io/) | $30/agent/month, no replay or monitoring included |

---

## Features

- **Real-time visitor monitoring** — full rrweb capture (DOM mutations, mouse, scroll, input)
- **Co-browsing** — bidirectional (cursor / click / scroll / form-fill / navigate); rrweb node IDs, no fragile CSS/XPath selectors
- **Session replay** — MinIO archive + rrweb-player; selective screenshots (canvas / WebGL / cross-origin iframe only, 1 fps WebP q70) to keep size sane
- **Live chat + popup** — operator-initiated popups + bidirectional instant chat
- **Multi-operator claim lock** — 1:1 visitor-operator locking (Redis `SET NX` + Lua release)
- **Anti-bot stack** — rate limit + UA blacklist + behavioral analysis + fingerprint (defense in depth)
- **GDPR-compliant** — consent opt-in + right-to-be-forgotten + IP truncation + co-browse consent banner
- **Bilingual i18n** — zh/en from day 1, no hard-coded strings
- **Unlimited session recording** — no session caps, no per-session fees. Your only cost is your own infrastructure.

---

## Self-host

```bash
git clone https://github.com/iannil/pinconsole
cd pinconsole
cp .env.example .env

# start infra + build the single release binary
make docker-up && make build

./server/bin/pinconsole-server
```

- Visitor landing page: http://localhost:8080/
- Admin console: http://localhost:8080/admin — default `admin@pinconsole.local` (password set via `ADMIN_PASSWORD` env var)

**Production deploy:**

```bash
docker compose --profile prod up -d --build
```

**Custom domains (ACME / Let's Encrypt):**

The server can automatically provision HTTPS certificates for custom domains via certmagic.
Set the following environment variables to enable:

| Variable | Default | Description |
|---|---|---|
| `PLATFORM_DOMAIN` | `""` | Your main domain (e.g. `pinconsole.com`), exempt from Host-header validation |
| `ACME_EMAIL` | `""` | **Required** — Let's Encrypt registration email. Server won't start ACME without it |
| `ACME_STAGING` | `true` | Use Let's Encrypt staging CA (recommended for testing). Set `false` for production |
| `ACME_DATA_DIR` | `./data/certmagic` | certmagic certificate cache directory |
| `ACME_HTTP_PORT` | `80` | HTTP-01 challenge + redirect port |

When `ACME_EMAIL` is set:
- Server listens on **:443** (HTTPS) with auto-provisioned certificates
- Server listens on **:80** for ACME HTTP-01 challenges + 301 redirect to HTTPS
- Admin UI at `/admin/domains` lets you add/remove custom domains
- Platform domain (`PLATFORM_DOMAIN`) and custom domains are validated via Host-header middleware

For full Make command list (`make help`), architecture deep-dive, and ops playbook: see [`docs/project-status.md`](./docs/project-status.md) and [`Makefile`](./Makefile).

---

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│ Visitor SDK  │────▶│  Go Server        │────▶│  PostgreSQL   │
│ (~15KB gzip) │     │  (Gin + WebSocket)│     │  (metadata)   │
└─────────────┘     │                   │     └──────────────┘
                    │  Hub-and-spoke    │     ┌──────────────┐
┌─────────────┐     │  architecture     │────▶│  Redis        │
│ Admin UI     │◀───▶│  All traffic via  │     │  (presence)   │
│ (Vue 3 SPA)  │     │  central server  │     └──────────────┘
└─────────────┘     │                   │     ┌──────────────┐
                    │  Single binary    │────▶│  MinIO        │
                    │  (Go embed)       │     │  (event store) │
                    └──────────────────┘     └──────────────┘
```

Tech stack: **Go 1.22** · **Vue 3** · **PostgreSQL 16** · **Redis 7** · **MinIO** · **rrweb** · **coder/websocket**

See the blog post: [How We Built a Self-Hosted Session Replay Alternative to FullStory](https://pinconsole.com/blog/building-self-hosted-session-replay/)

---

## Known limits (read before production)

1. **Single-instance hub (no horizontal scaling)**
   WebSocket routing uses an in-process map (`server/internal/hub/hub.go`).
   Multi-instance deployment (2+ servers behind a load balancer) silently breaks —
   visitors and operators on different instances can't see each other. The system
   does not error; it just appears broken. To scale horizontally, introduce Redis
   Pub/Sub or NATS as the message bus.

2. **500 WS/room concurrency target is not load-tested**
   PLAN.md drives single-tenant / hub-and-spoke / 1:1 locking decisions off the
   "500 WS/room" target, but v1 has not been load-tested. Defaults
   (`PG_MAX_CONNS=25` / `REDIS_POOL_SIZE=50`) are empirical. Capacity for your
   workload must be verified by the deployer.

3. **OSS project ships no production topology**
   docker-compose `prod` profile is reference only. Real production topology
   (VM / k8s / reverse proxy / TLS / backup / monitoring / log aggregation /
   resource limits) is the deployer's call. This repo guarantees only:
   - Repeatable dev / CI paths
   - Release binary is fail-secure (rejects weak configs by default — see [`docs/audits/`](./docs/audits/))
   - `/healthz` + `/readyz` dependency health checks

4. **Trace_id end-to-end propagation (closed in slice 1z)**
   operator browser → server → visitor SDK → server → operator forms a complete
   trace_id loop:
   - admin SPA injects `X-Trace-Id` on every REST call (`admin/src/api/client.ts`)
   - visitor SDK caches trace_id on operator command, inherits across the next 10 events or 5s (`visitor-sdk/src/transport/ws.ts`)
   - server `TraceMiddleware` + WS handler restore ctx trace_id

---

## Roadmap

v1 is shipped. Post-v1, prioritized (see [`PLAN.md`](./PLAN.md) §8 for full backlog):

1. ✅ **Custom domain** — DNS verification + Let's Encrypt ACME + Host-header routing
2. ✅ **Low-code page editor** — drag-drop / JSON schema → Go template render
3. **Tauri desktop client** — Win + Mac, reuses admin SPA
4. **SSO / SAML / OIDC** — enterprise auth

---

## Blog

- [How We Built a Self-Hosted Session Replay Alternative to FullStory](https://pinconsole.com/blog/building-self-hosted-session-replay/) (EN) · [中文](https://pinconsole.com/blog/self-hosted-fullstory-alternative/)
- [AGPL-3.0 vs MIT: Why We Chose AGPL for Our Open Source Project](https://pinconsole.com/blog/agpl-vs-mit/) (EN) · [中文](https://pinconsole.com/blog/agpl-vs-mit-zh/)

---

## License

AGPL-3.0 — see [`LICENSE`](./LICENSE).

*Built with Go 1.22 · Vue 3 · PostgreSQL · Redis · MinIO · rrweb.*
