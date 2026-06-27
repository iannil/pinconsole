import type { SeoPageContent } from './types';

export const fullstoryEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs FullStory: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with FullStory. Both offer session replay and analytics. Only PinConsole gives you self-hosting, co-browsing, visitor monitoring, and no per-session pricing.',
    ogTitle: 'PinConsole vs FullStory — Open Source Session Replay Alternative',
    ogDescription: 'Self-hosted alternative to FullStory. AGPL-3.0, your data on your servers. Session replay, co-browsing, real-time visitor monitoring, and anti-bot protection — all in one binary.',
  },
  hero: {
    h1: 'PinConsole vs FullStory: Open Source Alternative',
    subtitle: "FullStory is the industry standard for session replay and digital experience analytics — but it's SaaS-only, proprietary, and expensive at $500+/month. PinConsole delivers session replay, co-browsing, real-time visitor monitoring, and anti-bot protection in a single self-hosted binary, free under AGPL-3.0.",
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud only' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data stored on FullStory cloud' },
      { label: 'Session replay', pinconsole: 'rrweb-based, self-hosted storage', competitor: 'Full-featured with AI search' },
      { label: 'Co-browsing', pinconsole: 'Included (two-way click/scroll/fill/navigate)', competitor: 'Not included (separate tool needed)' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshot', competitor: 'Historical analytics only' },
      { label: 'Heatmaps / rage clicks', pinconsole: 'Planned (roadmap)', competitor: 'Full-featured' },
      { label: 'Popups & proactive chat', pinconsole: 'Push notifications + two-way chat', competitor: 'Not included' },
      { label: 'Anti-bot protection', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not included' },
      { label: 'SDK footprint', pinconsole: 'Lightweight (~15KB gzip)', competitor: '~50KB+ (heavier)' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'From $500/month (Business plan)' },
      { label: 'Deployment', pinconsole: 'Docker compose, 5 minutes', competitor: 'SaaS snippet only' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from FullStory to PinConsole?',
        body: "FullStory is a powerful digital experience platform with best-in-class session replay, AI-powered search, and analytics. But for many teams, the SaaS-only model presents challenges: your session data lives on FullStory's servers, the cost scales quickly (Business plan starts at $500/month), and there's no co-browsing or real-time visitor monitoring — you'd need additional tools for those capabilities. PinConsole is a self-hosted alternative that covers session replay, co-browsing, real-time monitoring, and anti-bot protection in a single binary, with no recurring per-session fees.",
      },
      {
        heading: 'Self-hosted means your data, your control',
        body: "With FullStory, every session recording is transmitted to and stored on their cloud infrastructure. For organizations bound by GDPR, SOC 2, HIPAA, or China's Personal Information Protection Law, this can be a compliance hurdle. PinConsole runs entirely on your infrastructure — session data, DOM snapshots, and user interactions never leave your servers. You control retention policies, backup schedules, and data deletion — no vendor lock-in, no surprise policy changes.",
      },
      {
        heading: 'More than replays — a complete customer ops suite',
        body: "FullStory excels at session replay and analytics, but it doesn't cover co-browsing, real-time visitor monitoring, or customer interaction tools. PinConsole is designed as a complete customer operations platform: replay past sessions to debug issues, watch live visitors in real time, proactively engage them with popup chat, and assist with co-browsing — all from a single admin interface. Plus, built-in anti-bot protection keeps your analytics clean from automated traffic.",
      },
      {
        heading: 'Open source, auditable, and free',
        body: "FullStory is proprietary — you can't audit the code, self-host, or influence the roadmap. PinConsole is AGPL-3.0 open source: the full source is on GitHub, you can audit every line, fork the project, and self-host indefinitely. No license fees, no per-seat costs, no surprise price hikes. Commercial licenses are available for teams that need proprietary embedding.",
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'FullStory Business plan starts at $599/month — 10,000 sessions included, then $0.04 per additional session. Annual commitment required. No self-hosted option available.',
        attribution: 'FullStory Pricing Page (as of June 2026)',
        sourceUrl: 'https://www.fullstory.com/plans/',
      },
      {
        quote: "FullStory's JavaScript SDK is approximately 50KB+ gzipped and can impact page load performance. Their own documentation recommends async loading to mitigate this.",
        attribution: 'FullStory Developer Docs',
        sourceUrl: 'https://developer.fullstory.com/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Can PinConsole replace FullStory for session replay?',
        answer: "Yes — PinConsole uses rrweb for DOM-based session capture and replay, the same technology used by many production tools. It records all user interactions (clicks, scrolls, inputs) and replays them with pixel-perfect accuracy. FullStory has more advanced AI-powered search and analytics features, but PinConsole covers the core replay use case plus co-browsing and live monitoring that FullStory doesn't offer.",
      },
      {
        question: 'Does PinConsole have heatmaps and rage click detection?',
        answer: "Not yet — heatmaps and rage click analytics are on the PinConsole roadmap. For teams that need these features alongside self-hosted session replay, PinConsole can serve as the core replay and monitoring platform while you continue using FullStory's free tier or a dedicated analytics tool for heatmaps in the interim.",
      },
      {
        question: 'How does PinConsole handle GDPR and data privacy compliance?',
        answer: 'Because PinConsole is self-hosted, all session data stays on your infrastructure — no third-party data processing, no cross-border data transfers. This makes GDPR compliance significantly simpler compared to SaaS-only solutions like FullStory, where data is processed on their US-based cloud.',
      },
      {
        question: 'Is PinConsole suitable for large-scale session recording?',
        answer: 'Yes — PinConsole is built on Go with PostgreSQL, Redis, and MinIO for storage, designed for production workloads. The architecture supports concurrent WebSocket connections (500+/room) and efficient event-stream storage via MinIO. Deployment is straightforward with Docker Compose.',
      },
    ],
  },
  cta: {
    title: 'Self-host PinConsole in 5 minutes',
    subtitle: 'No sign-up wall. No sales call. Just clone, compose, and deploy.',
    primary: { label: 'Get started', href: '#top' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
