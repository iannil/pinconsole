import type { BlogContent } from './types';

export const fullstoryAlternativeEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'How We Built a Self-Hosted Session Replay Alternative to FullStory',
    description: 'A technical deep-dive into building PinConsole — an open-source, self-hosted session replay and co-browsing platform. Go backend, rrweb DOM capture, MinIO storage, and a single-binary deployment model.',
    ogTitle: 'How We Built a Self-Hosted Session Replay Alternative to FullStory',
    ogDescription: 'Go + rrweb + MinIO: the architecture behind PinConsole, an open-source self-hosted alternative to FullStory.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-27',
    readingTime: '8 min read',
    tags: ['Engineering', 'Architecture', 'Self-Hosted', 'Open Source'],
  },
  hero: {
    h1: 'How We Built a Self-Hosted Session Replay Alternative to FullStory',
    subtitle: 'Why we built PinConsole, the architecture decisions we made, and how you can deploy it in 5 minutes.',
  },
  content: {
    sections: [
      {
        heading: 'The Problem with SaaS Session Replay',
        body: `FullStory is a fantastic product. Its session replay is polished, its AI-powered search is impressive, and its heatmaps and rage-click analytics are best-in-class. But for many teams, FullStory (and similar tools like Hotjar, LogRocket, and Smartlook) share a fundamental limitation: they are SaaS-only.\n\nThis means your session data — every click, scroll, and input your users make — is transmitted to and stored on a third-party cloud. For organizations with strict data sovereignty requirements (GDPR, SOC 2, HIPAA, or China's Personal Information Protection Law), this creates a compliance headache. The enterprise on-premise plans exist, but they come with enterprise pricing: FullStory Business starts at $500/month and scales quickly.\n\nBeyond cost and compliance, there's another gap: none of these tools offer co-browsing or real-time visitor monitoring. If you need to see what a visitor is doing right now and proactively help them, you need a separate tool like Cobrowse.io or Upscope — another subscription, another integration.\n\nWe built PinConsole to solve all of these problems in one self-hosted binary.`,
      },
      {
        heading: 'Why Go?',
        body: `The decision to build the backend in Go was driven by three requirements:\n\n1. Single-binary deployment — we wanted users to deploy PinConsole with nothing more than Docker Compose. Go compiles to a statically-linked binary with zero runtime dependencies.\n\n2. Concurrency — session replay involves hundreds of concurrent WebSocket connections. Go's goroutine model handles this gracefully without the complexity of thread pools or event loops.\n\n3. Performance — the event pipeline (receive rrweb events → store in MinIO → broadcast to admin) needs to be fast and predictable. Go's runtime delivers consistent latency under load.\n\nWe target 500 concurrent WebSocket connections per room as our baseline — that's one operator serving a mid-size e-commerce site's traffic in real time.`,
      },
      {
        heading: 'Architecture Overview',
        body: `PinConsole follows a hub-and-spoke architecture. All traffic flows through the central server — there are no P2P connections, no WebRTC, no third-party relay.\n\nThe stack is straightforward:\n\n• PostgreSQL — metadata storage (sites, sessions, users, operator accounts)\n• Redis — presence tracking, rate limiting, hot cache\n• MinIO — rrweb event stream storage + selective screenshots\n• Go server — Gin HTTP router + coder/websocket hub\n\nThe admin frontend is a Vue 3 SPA embedded via Go's //go:embed directive. The visitor SDK is a TypeScript library served from the same origin. Everything — server, admin UI, SDK — is a single binary.`,
        code: `docker compose up -d\n# PostgreSQL, Redis, and MinIO start as sidecars\n# The PinConsole binary binds to :8080\n# Admin UI: https://app.yourdomain.com\n# SDK: https://app.yourdomain.com/sdk.js`,
        codeLanguage: 'bash',
      },
      {
        heading: 'DOM Capture with rrweb',
        body: `For session recording, we use rrweb — the same open-source library that powers many production session replay tools. rrweb captures DOM mutations as a serialized event stream: full DOM snapshots at the start, then incremental mutations (click, scroll, input, resize, etc.) as they happen.\n\nWhy rrweb over alternatives?\n\n• It's battle-tested — used by projects with millions of recorded sessions\n• The event format is well-specified and serializable (JSON)\n• It handles iframes, shadow DOM, canvas/WebGL snapshots (with capture plugins)\n• It's MIT-licensed — compatible with our AGPL-3.0 stack\n\nWe did need to build some custom infrastructure around rrweb. The raw event stream from a busy site can generate hundreds of events per second. Rather than storing each event as an individual object, we batch events into time-windowed chunks and store them as MinIO objects. This reduces storage operations by 10-100x and makes replay sequential reads fast.\n\nFor privacy, we implemented a selective capture mode. Instead of recording the full DOM, operators can configure CSS selectors to mask sensitive elements (password fields, credit card inputs, PII containers). The masked data is never sent to the server — it's stripped at the SDK level before transmission.`,
      },
      {
        heading: 'Real-Time WebSocket Hub',
        body: `The real-time layer is the heart of PinConsole's co-browsing and live monitoring features. We chose coder/websocket over alternatives for a simple reason: it's a minimal, idiomatic Go library with no global state, no implicit goroutine spawning, and a clean API.\n\nThe hub pattern works like this:\n\n• Each site has a "room" identified by site ID\n• Visitor SDKs connect to their site's room\n• Operators connect to the same room\n• Messages are routed: visitor → hub → operator (and vice versa for co-browsing)\n\nFor live monitoring, the SDK sends a DOM snapshot every 100ms (throttled) plus incremental events. The operator's admin panel renders this as a live preview. For co-browsing, operator actions (click, scroll, form fill) are serialized as rrweb mutation events and forwarded to the visitor's browser — with a 300ms debounce to prevent conflicts.\n\nOne detail that mattered: we do NOT broadcast all events to all operators. Each visitor stream is 1:1 locked to one operator (claim/release pattern). This avoids the overhead of fan-out and ensures clear ownership — no two operators fighting to control the same co-browsing session.`,
      },
      {
        heading: 'Event Storage with MinIO',
        body: `Session replay data is write-heavy and append-only. A 10-minute session can generate 50,000+ rrweb events (roughly 2-5MB of serialized JSON). Storing this in PostgreSQL would work but isn't ideal — blob storage is cheaper, faster for sequential reads, and doesn't compete with your OLTP workload.\n\nMinIO (S3-compatible object storage) is a natural fit:\n\n• Each session's events are stored as a sequence of objects in a "session" prefix\n• The first object is the full DOM snapshot (snapshot.json)\n• Subsequent objects are 5-second time-windowed event batches (events-{timestamp}.json)\n• Screenshot mode: when canvas/WebGL/cross-origin iframe is detected, a 1fps WebP screenshot (quality 70) is stored alongside the event stream\n\nFor replay, the server reads the event objects in order and streams them to the admin UI. Gap detection handles missing batches gracefully — if a batch is unavailable, replay shows a "buffering" indicator and skips ahead.\n\nWe also implemented a retention policy system: operators can set TTL per site (7/30/90 days or forever). MinIO's lifecycle policies handle the actual deletion — we just set the object tags.`,
      },
      {
        heading: 'The SDK: Tiny, Modular, and Self-Hosted',
        body: `The visitor SDK is built with TypeScript + Vite. It ships as a single /sdk.js file served from the same origin as the PinConsole admin panel. No CDN, no third-party script loaders — if your server is reachable, the SDK works.\n\nThe SDK weighs approximately 15KB gzipped (compared to FullStory's ~50KB+). We achieved this by:\n\n• Only bundling the rrweb core recorder (not the replayer — that's server-side)\n• Using a lightweight WebSocket client instead of HTTP long-polling\n• Making features opt-in via the SDK configuration object\n\nThe SDK lifecycle:\n\n1. Page loads → SDK initializes and establishes a WebSocket connection\n2. rrweb starts recording DOM mutations with selective masking\n3. Events are buffered locally and flushed every 200ms\n4. If the connection drops, events are queued and replayed on reconnect\n5. When the visitor leaves, a "session-end" marker is sent\n\nWe deliberately avoid any form of user identification (no fingerprinting beyond what's needed for bot detection). The session is identified by a server-generated UUID, not by tracking the visitor across sites.`,
      },
      {
        heading: 'Security and Anti-Bot Design',
        body: `Since PinConsole is self-hosted and designed for production traffic, we built defense-in-depth from day one.\n\nThe anti-bot system operates at four layers:\n\n1. Rate limiting — per-IP and per-session event rate caps (configurable)\n2. User-Agent blacklist — known bot UAs are rejected at the HTTP level\n3. Behavioral analysis — the server monitors event patterns. A session that generates 1000 clicks/second is probably a bot, not a human\n4. TLS fingerprint — passive JA3 fingerprinting of incoming WebSocket connections\n\nFor the admin panel, all authenticated routes require an HttpOnly session cookie. WebSocket connections for operators use the same cookie for authentication — no token in the URL, no token in the query string.\n\nWe also added a Content Security Policy that's strict by default: only the self-origin can load scripts, only challenges.cloudflare.com is allowed for Turnstile (if enabled), and inline styles are hashed.`,
      },
      {
        heading: 'FullStory Feature Parity: What We Have vs What\'s Coming',
        body: `We're honest about where we stand:\n\n✓ Session replay (rrweb-based, pixel-perfect)\n✓ Real-time visitor monitoring (DOM + screenshot)\n✓ Two-way co-browsing (click, scroll, form fill, navigation)\n✓ Proactive popup chat\n✓ Anti-bot protection\n✓ Self-hosted (your servers, your data)\n✓ AGPL-3.0 (free) with commercial license available\n\n△ Heatmaps and rage clicks — on the roadmap (est. Q3 2026)\n△ AI-powered session search — on the roadmap\n△ Funnel and conversion analytics — on the roadmap\n\n✗ Native mobile SDKs — not in v1 scope. PinConsole v1 targets web only.\n✗ Custom domain per site — on the post-v1 backlog\n\nIf you need FullStory's advanced analytics layer today, PinConsole works great as the core session replay and monitoring platform while you keep FullStory for heatmaps. But for teams that prioritize data sovereignty, cost control, and open source — PinConsole already covers the critical use cases.`,
      },
      {
        heading: 'Deploying in Production',
        body: `Deploying PinConsole is intentionally boring:\n\n1. Clone the repository\n2. Configure .env with your secrets\n3. docker compose up -d\n4. Add the SDK snippet to your website\n\nFor production, we recommend:\n\n• PostgreSQL 15+ with connection pooling (PgBouncer recommended for 100+ concurrent sites)\n• Redis 7+ for presence and rate limiting\n• MinIO in distributed mode for HA object storage\n• A reverse proxy (Nginx/Caddy) for TLS termination\n• Regular backups of PostgreSQL and MinIO buckets\n\nThe single-binary model means upgrades are atomic: pull the new image, restart, done. Database migrations run automatically on startup. No package managers, no runtime upgrades, no dependency conflicts.`,
        code: `git clone https://github.com/iannil/pinconsole\ncd pinconsole\ncp .env.example .env\n# Edit .env with your secrets\ndocker compose up -d`,
        codeLanguage: 'bash',
      },
    ],
  },
  cta: {
    title: 'Try PinConsole Today',
    subtitle: 'Self-hosted in 5 minutes. AGPL-3.0. Your data, your servers.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
