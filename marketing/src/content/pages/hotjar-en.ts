import type { SeoPageContent } from './types';

export const hotjarEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs Hotjar: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with Hotjar. Both offer session replay and heatmaps, but only PinConsole gives you self-hosting, co-browsing, and no session limits.',
    ogTitle: 'PinConsole vs Hotjar — Open Source Session Replay & Heatmap Alternative',
    ogDescription: 'Self-hosted alternative to Hotjar. AGPL-3.0, your data on your servers. Session replay, heatmaps (roadmap), co-browsing, real-time visitor monitoring, and anti-bot protection.',
  },
  hero: {
    h1: 'PinConsole vs Hotjar: Open Source Alternative',
    subtitle: "Hotjar is one of the most popular tools for session replay and heatmaps — but its free tier limits you to 35 recorded sessions per day, paid plans start at $32/month, and there's no self-hosted option. PinConsole delivers session replay, co-browsing, real-time monitoring, and anti-bot protection in a single self-hosted binary, free under AGPL-3.0 with no session caps.",
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud only' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data stored on Hotjar cloud (EU/US)' },
      { label: 'Session replay', pinconsole: 'rrweb-based, unlimited sessions', competitor: '35/day on free plan, paid plans scale' },
      { label: 'Heatmaps', pinconsole: 'Planned (roadmap Q3 2026)', competitor: 'Full-featured click/move/scroll heatmaps' },
      { label: 'Co-browsing', pinconsole: 'Included (two-way click/scroll/fill/navigate)', competitor: 'Not included' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshot', competitor: 'Historical only' },
      { label: 'Popups & surveys', pinconsole: 'Push notifications + two-way chat', competitor: 'Widgets & surveys included' },
      { label: 'Anti-bot protection', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not included' },
      { label: 'SDK footprint', pinconsole: '~15KB gzip', competitor: '~35KB (Tracking + Identify + Events)' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'Free tier (35 sessions/day). From $32/month (Plus), $83/month (Business), $171/month (Scale)' },
      { label: 'Deployment', pinconsole: 'Docker compose, 5 minutes', competitor: 'SaaS snippet only' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from Hotjar to PinConsole?',
        body: "Hotjar is beloved for its intuitive heatmaps and session replay — it's often the first analytics tool teams adopt. But as your site grows, the limitations become clear: the free tier caps at 35 recorded sessions per day, paid plans get expensive quickly (Scale plan at $171/month), and you're limited to historical analysis only. There's no co-browsing, no real-time visitor monitoring, and no self-hosted option. PinConsole is a self-hosted alternative that covers session replay, co-browsing, real-time monitoring, and anti-bot protection in a single binary — with no session limits and no recurring fees.",
      },
      {
        heading: 'From session caps to unlimited recording',
        body: "Hotjar's free tier is generous for startups — but 35 daily recordings means you're only seeing a tiny fraction of your traffic. The Plus plan ($32/month) increases this to 100 daily sessions, Business ($83/month) to 500, and Scale ($171/month) to 2,000. If you have 10,000 daily visitors, you're seeing 0.35% to 20% of their behavior depending on your plan tier. With PinConsole, there are no session caps — every visitor interaction is recorded and available for replay. Your storage limit is determined by your MinIO and PostgreSQL capacity, not your subscription tier.",
      },
      {
        heading: 'Real-time capabilities Hotjar doesn\'t offer',
        body: "Hotjar is designed for historical analysis — you record sessions today, analyze them tomorrow. It excels at this, but it doesn't let you see what's happening on your site right now. PinConsole adds three real-time capabilities that Hotjar doesn't offer: live visitor monitoring (see every visitor's DOM in real time), proactive popup chat (engage visitors before they leave), and two-way co-browsing (take over a visitor's session to help them). These features turn PinConsole from an analytics tool into a customer operations platform.",
      },
      {
        heading: 'Self-hosted means no data leaves your infrastructure',
        body: "Hotjar stores your session data on their EU or US cloud. For organizations under GDPR, SOC 2, HIPAA, or China's Personal Information Protection Law, this creates additional compliance overhead. With PinConsole, all session data — recordings, heatmap data, DOM snapshots — stays on your own infrastructure. You control data retention, backup policies, and deletion schedules. No third-party processor, no cross-border transfer concerns.",
      },
      {
        heading: 'Open source, no vendor lock-in',
        body: "Hotjar was acquired by Contentsquare in 2021 — product direction and pricing are now controlled by a larger enterprise analytics company. With PinConsole, the AGPL-3.0 license guarantees you can always self-host, audit the code, fork the project, and control your own roadmap. No surprise pricing changes, no feature removals, no forced migrations. Commercial licenses are available for teams that need proprietary embedding.",
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'Hotjar Plus plan: $32/month — 100 daily sessions recorded, 1-year data retention. Business: $83/month — 500 daily sessions. Scale: $171/month — 2,000 daily sessions. All plans include heatmaps, session replay, and surveys.',
        attribution: 'Hotjar Pricing Page (as of June 2026)',
        sourceUrl: 'https://www.hotjar.com/pricing/',
      },
      {
        quote: 'Hotjar\'s tracking script weighs approximately 35KB gzipped, covering page views, session recording, heatmaps, and form analysis via three separate script components.',
        attribution: 'Hotjar Documentation',
        sourceUrl: 'https://help.hotjar.com/hc/en-us/articles/360019339974',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Can PinConsole replace Hotjar for session replay and heatmaps?',
        answer: "For session replay — yes, PinConsole provides unlimited rrweb-based recording with pixel-perfect replay. For heatmaps — not yet; heatmaps and rage click analytics are on the PinConsole roadmap. If you need both today, PinConsole can serve as your core replay and monitoring platform while you continue using Hotjar's free tier for heatmaps during the transition.",
      },
      {
        question: 'Does PinConsole support surveys and feedback widgets like Hotjar?',
        answer: "Not in v1 — PinConsole focuses on session replay, co-browsing, and real-time monitoring. For surveys and feedback widgets, you can continue using Hotjar's free tier or a dedicated tool. PinConsole's popup chat system can handle proactive messaging, but it's designed for real-time engagement rather than async surveys.",
      },
      {
        question: 'Is PinConsole suitable for teams that are just starting with analytics?',
        answer: 'Absolutely — PinConsole is designed to be simple to deploy (Docker compose, 5 minutes) and use. The setup cost is zero: no monthly fees, no per-session charges, no commitment. Teams can start with unlimited session recording from day one and grow without worrying about plan tiers.',
      },
      {
        question: 'How does PinConsole compare to Hotjar on page performance impact?',
        answer: 'PinConsole\'s SDK is approximately 15KB gzipped — roughly half the size of Hotjar\'s combined tracking scripts. Both use asynchronous loading to minimize page impact. With PinConsole, you also get the benefit of serving the SDK from your own domain (no additional DNS lookups).',
      },
    ],
  },
  cta: {
    title: 'Self-host PinConsole in 5 minutes',
    subtitle: 'No sign-up wall. No session caps. No sales call.',
    primary: { label: 'Get started', href: '#top' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
