import type { SeoPageContent } from './types';

export const smartlookEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs Smartlook: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with Smartlook. Both offer session replay and analytics. Only PinConsole gives you self-hosting, co-browsing, real-time visitor monitoring, and no per-session pricing.',
    ogTitle: 'PinConsole vs Smartlook — Open Source Session Replay Alternative',
    ogDescription: 'Self-hosted alternative to Smartlook. AGPL-3.0, your data on your servers. Session replay, co-browsing, real-time visitor monitoring, and anti-bot protection.',
  },
  hero: {
    h1: 'PinConsole vs Smartlook: Open Source Alternative',
    subtitle: "Smartlook provides session replay, heatmaps, and product analytics with a generous free tier (up to 3,000 sessions/month). But it's SaaS-only, and paid plans scale with session volume. PinConsole delivers session replay, co-browsing, real-time monitoring, and anti-bot protection — all self-hosted, with no session limits, under AGPL-3.0.",
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud only' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data stored on Smartlook cloud (EU)' },
      { label: 'Session replay', pinconsole: 'rrweb-based, unlimited sessions', competitor: '3k sessions/mo free, paid plans by volume' },
      { label: 'Heatmaps', pinconsole: 'Planned (roadmap)', competitor: 'Click/move/scroll heatmaps' },
      { label: 'Co-browsing', pinconsole: 'Included (two-way click/scroll/fill/navigate)', competitor: 'Not included' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshot', competitor: 'Historical only' },
      { label: 'Popups & proactive chat', pinconsole: 'Push notifications + two-way chat', competitor: 'Not included' },
      { label: 'Anti-bot protection', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not included' },
      { label: 'SDK footprint', pinconsole: '~15KB gzip', competitor: '~25KB+ (with all features)' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'Free (3k sessions/mo). Paid plans start from volume-based pricing.' },
      { label: 'Deployment', pinconsole: 'Docker compose, 5 minutes', competitor: 'SaaS snippet only' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from Smartlook to PinConsole?',
        body: "Smartlook is a solid session replay and product analytics tool with a generous free tier (3,000 sessions/month) and a clean UI. It's particularly popular with product teams and mobile app developers. But Smartlook is SaaS-only — your session data is stored on their EU cloud, and paid plans scale with session volume. There's no co-browsing, no real-time visitor monitoring, and no self-hosted option. PinConsole is a self-hosted alternative that covers session replay, co-browsing, real-time monitoring, and anti-bot protection in a single binary — with no session limits and no recurring fees.",
      },
      {
        heading: 'From session-quota analytics to unlimited recording',
        body: "Smartlook's free tier (3,000 sessions/month) is one of the most generous in the industry — suitable for small projects and early-stage startups. But growing sites quickly exceed this: a site with 10,000 monthly sessions needs a paid plan, and pricing scales with volume. With PinConsole, there are no session quotas — record every visitor interaction without counting credits. Your only cost is your own infrastructure.",
      },
      {
        heading: 'Real-time capabilities Smartlook doesn\'t offer',
        body: "Smartlook excels at historical session replay and product analytics — you watch recordings after they happen. But it doesn't let you see live visitor behavior or interact with users in real time. PinConsole adds three capabilities that Smartlook doesn't have: live visitor monitoring (watch sessions as they happen), proactive popup chat (engage visitors before they bounce), and two-way co-browsing (guide a struggling visitor through a checkout flow). These transform session replay from an analytics tool into a real-time customer operations platform.",
      },
      {
        heading: 'Self-hosted means your data, your control',
        body: "Smartlook stores session data on its EU-based cloud infrastructure. While GDPR-compliant, this still means data leaves your network and is processed by a third party. For organizations with strict data residency requirements (HIPAA, China's PIPL, internal security policies), this can be a blocker. PinConsole runs entirely on your infrastructure — session data, DOM snapshots, and telemetry never leave your servers. You control retention, backups, and deletion.",
      },
      {
        heading: 'Open source, no vendor dependency',
        body: "Smartlook is proprietary — your analytics workflow depends on their continued operation, pricing, and feature decisions. With PinConsole, the AGPL-3.0 license guarantees you can always self-host, audit the code, fork the project, and control your own destiny. No surprise pricing changes, no sunsetted features, no forced migrations. For product teams that value long-term independence, PinConsole offers the core session replay capabilities with additional real-time features — all self-hosted and free.",
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'Smartlook Free: 3,000 sessions/month, 1-month data retention. Paid plans start with volume-based pricing. Mobile SDK support included on higher tiers. All plans are SaaS-only — no self-hosted deployment available.',
        attribution: 'Smartlook Pricing Page (as of June 2026)',
        sourceUrl: 'https://www.smartlook.com/pricing/',
      },
      {
        quote: 'Smartlook provides session replay, heatmaps, and product analytics with mobile app SDKs for iOS and Android. It does not offer co-browsing, live visitor monitoring, or proactive chat — these require separate tools.',
        attribution: 'Smartlook Features Overview',
        sourceUrl: 'https://www.smartlook.com/features/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Can PinConsole replace Smartlook for session replay?',
        answer: "For web session replay — yes, PinConsole provides unlimited rrweb-based recording with pixel-perfect replay. Smartlook has an advantage in mobile app SDK support (iOS/Android) — PinConsole v1 is web-only. For product analytics and heatmaps — not yet; these are on the PinConsole roadmap. PinConsole offers features Smartlook doesn't have: co-browsing, real-time monitoring, and proactive chat.",
      },
      {
        question: 'Does PinConsole support mobile app session recording?',
        answer: "Not in v1 — PinConsole v1 targets web only. Mobile SDK support (iOS/Android) is on the post-v1 backlog. For mobile-only projects, Smartlook remains the better choice. For web-first teams, PinConsole covers session replay plus real-time features that Smartlook doesn't offer.",
      },
      {
        question: 'How does PinConsole compare to Smartlook on data retention?',
        answer: 'PinConsole\'s data retention is entirely under your control — you set the TTL per site (7/30/90 days or forever), and MinIO lifecycle policies handle the rest. Smartlook\'s free tier includes 1-month retention; paid plans offer longer retention. With PinConsole, there are no retention limits based on your subscription tier.',
      },
      {
        question: 'Is PinConsole suitable for product teams that need analytics?',
        answer: 'Yes — PinConsole covers session replay and real-time monitoring, which are core needs for product teams. If you rely heavily on Smartlook\'s product analytics dashboards (funnels, trends, user paths), you may want to keep both during a transition period. PinConsole handles the replay and monitoring side while you evaluate whether the analytics features meet your needs.',
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
