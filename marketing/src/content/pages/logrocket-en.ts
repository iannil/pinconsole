import type { SeoPageContent } from './types';

export const logrocketEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs LogRocket: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with LogRocket. Both offer session replay and frontend monitoring. Only PinConsole gives you self-hosting, co-browsing, real-time visitor monitoring, and no per-session pricing.',
    ogTitle: 'PinConsole vs LogRocket — Open Source Session Replay Alternative',
    ogDescription: 'Self-hosted alternative to LogRocket. AGPL-3.0, your data on your servers. Session replay, co-browsing, real-time visitor monitoring, and anti-bot protection — all in one binary.',
  },
  hero: {
    h1: 'PinConsole vs LogRocket: Open Source Alternative',
    subtitle: "LogRocket combines session replay with frontend monitoring (console logs, network requests, JS errors) — a powerful combo for developers. But it's SaaS-only, pricing scales with sessions, and there's no self-hosted option. PinConsole delivers session replay, co-browsing, real-time monitoring, and anti-bot protection in a single self-hosted binary, free under AGPL-3.0 with no session limits.",
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud only' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data stored on LogRocket cloud' },
      { label: 'Session replay', pinconsole: 'rrweb-based, unlimited sessions', competitor: '1k sessions/mo free, 5k ($99), 15k ($249)' },
      { label: 'Frontend monitoring', pinconsole: 'Planned (roadmap)', competitor: 'Console logs, network, JS errors, performance' },
      { label: 'Co-browsing', pinconsole: 'Included (two-way click/scroll/fill/navigate)', competitor: 'Not included' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshot', competitor: 'Historical replay only' },
      { label: 'Popups & proactive chat', pinconsole: 'Push notifications + two-way chat', competitor: 'Not included' },
      { label: 'Anti-bot protection', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not included' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'Free (1k sessions/mo). From $99/mo (5k sessions), $249/mo (15k sessions)' },
      { label: 'Deployment', pinconsole: 'Docker compose, 5 minutes', competitor: 'SaaS snippet only' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from LogRocket to PinConsole?',
        body: "LogRocket is a developer-friendly session replay tool with unique frontend monitoring capabilities — console logs, network requests, and JS errors are captured alongside the replay. This is genuinely useful for debugging. But LogRocket's SaaS-only model means your session data lives on their cloud, and the pricing is per-session: 1,000 free sessions per month, then $99 for 5,000 sessions, $249 for 15,000. High-traffic teams can face significant costs. PinConsole offers unlimited session replay with no per-session fees, plus co-browsing and real-time visitor monitoring that LogRocket doesn't have — all in a self-hosted binary.",
      },
      {
        heading: 'From session-capped debugging to unlimited recording',
        body: "LogRocket's free tier (1,000 sessions/month) is reasonable for small projects, but production sites with meaningful traffic will quickly exceed it. At $249/month for 15,000 sessions, a site with 50,000 monthly visitors would need multiple plans or face significant costs. With PinConsole, there are no session caps — every visitor interaction is recorded. Whether you have 1,000 or 100,000 monthly sessions, the cost is the same: zero license fees. Your only cost is the infrastructure (a few dollars/month on a VPS for small sites).",
      },
      {
        heading: 'LogRocket lacks real-time interactions',
        body: "LogRocket excels at recording what happened — console errors, slow network requests, rage clicks. But it's fundamentally a historical tool. You can replay a session to debug an issue, but you can't see what's happening on your site right now, and you can't interact with visitors. PinConsole fills this gap with three real-time features: live visitor monitoring (watch every visitor's session as it happens), proactive popup chat (engage visitors before they bounce), and two-way co-browsing (take control of a session to guide the user). These transform session replay from a debugging tool into a customer operations platform.",
      },
      {
        heading: 'Self-hosted means your data, your control',
        body: "LogRocket processes and stores all session data — including console logs, network requests, and DOM mutations — on their cloud infrastructure. For security-conscious teams or organizations with data residency requirements, this can be a blocker. PinConsole runs entirely on your infrastructure. Session data, console-like telemetry, and DOM snapshots never leave your servers. You define retention policies, backup schedules, and deletion rules — with no vendor access to your data.",
      },
      {
        heading: 'Open source, developer-friendly, and free',
        body: "LogRocket is proprietary — you can't audit the recording logic, extend the platform, or control the roadmap. PinConsole is AGPL-3.0 open source: the full source is on GitHub, you can audit every line, fork the project, and self-host indefinitely. For developer teams that value code transparency and self-determination, PinConsole offers the same session replay capabilities as LogRocket plus real-time features — without the per-session pricing or vendor lock-in.",
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'LogRocket Free: 1,000 sessions/month, 3-day retention. Professional: $99/month — 5,000 sessions, 7-day retention. Enterprise: $249/month — 15,000 sessions, 30-day retention. All plans include session replay, console logs, and network monitoring.',
        attribution: 'LogRocket Pricing Page (as of June 2026)',
        sourceUrl: 'https://logrocket.com/pricing/',
      },
      {
        quote: 'LogRocket captures console.log, console.warn, console.error, unhandled JS exceptions, fetch/XHR network requests and responses, and performance metrics — all alongside the session replay timeline.',
        attribution: 'LogRocket Documentation',
        sourceUrl: 'https://docs.logrocket.com/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Can PinConsole replace LogRocket for frontend debugging?',
        answer: "For session replay — yes, PinConsole provides unlimited rrweb-based recording with the same DOM-level capture as LogRocket. For frontend monitoring (console logs, network requests, JS errors) — not yet; these are on the PinConsole roadmap. During the transition, you can use PinConsole for unlimited session replay while keeping LogRocket's free tier for frontend monitoring.",
      },
      {
        question: 'Does PinConsole capture console logs and network requests like LogRocket?',
        answer: "Not in v1 — PinConsole v1 focuses on DOM-level session capture via rrweb. Console log capture and network request monitoring are on the roadmap. However, PinConsole offers features that LogRocket doesn't have: co-browsing, real-time visitor monitoring, and proactive chat. Depending on your primary use case, PinConsole may already cover what you need.",
      },
      {
        question: 'How does PinConsole compare on session storage costs?',
        answer: 'PinConsole uses MinIO (S3-compatible object storage) for event streams, which is significantly cheaper than LogRocket\'s per-session pricing. A typical session costs fractions of a cent in S3-compatible storage. For high-traffic sites, the infrastructure cost of self-hosting PinConsole is usually far lower than LogRocket\'s subscription fees.',
      },
      {
        question: 'Is PinConsole suitable for developer teams that need debugging capabilities?',
        answer: 'Yes — PinConsole\'s session replay is built on rrweb, the same technology used by many production debugging tools. Developers can replay sessions, inspect DOM state at any point, and use the timeline to understand user behavior. For teams that need both session replay and frontend monitoring, PinConsole covers the replay side while you use browser DevTools or a lightweight monitoring tool for console logs.',
      },
    ],
  },
  cta: {
    title: 'Self-host PinConsole in 5 minutes',
    subtitle: 'No sign-up wall. No session limits. No sales call.',
    primary: { label: 'Get started', href: '#top' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
