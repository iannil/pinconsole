import type { SeoPageContent } from './types';

export const upscopeEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs Upscope: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with Upscope. PinConsole offers co-browsing, session replay, and visitor monitoring — all on your own infrastructure.',
    ogTitle: 'PinConsole vs Upscope — Open Source Co-Browsing Alternative',
    ogDescription: 'Self-hosted alternative to Upscope. AGPL-3.0 licensed, data never leaves your infrastructure. Co-browsing, session replay, and visitor monitoring included.',
  },
  hero: {
    h1: 'PinConsole vs Upscope: Open Source Alternative',
    subtitle: 'Both deliver co-browsing and session replay. The difference: Upscope is SaaS with per-agent pricing. PinConsole is open-source, self-hosted, and AGPL-3.0 — your data never leaves your infrastructure.',
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data stored in Upscope cloud' },
      { label: 'Co-browsing', pinconsole: 'Full two-way (click/scroll/fill/navigate)', competitor: 'Two-way co-browsing' },
      { label: 'Session replay', pinconsole: 'Included (rrweb-based)', competitor: 'Add-on feature' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + screenshot', competitor: 'Basic presence only' },
      { label: 'GDPR compliance', pinconsole: 'Built-in: consent, erasure, IP truncation', competitor: 'Vendor-dependent' },
      { label: 'Anti-bot protection', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not available' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'Per-agent per-month SaaS' },
      { label: 'Self-hosted deploy', pinconsole: 'One docker-compose, 5 minutes', competitor: 'Not possible (SaaS only)' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from Upscope to PinConsole?',
        body: 'Upscope is a capable co-browsing tool, but its SaaS model means your screen-share data, visitor behavior, and session recordings all live on their servers. For compliance-sensitive industries — finance, insurance, healthcare, government — this is a hard blocker. PinConsole gives you the same co-browsing and session replay capabilities, deployed entirely on your own infrastructure. No data leaves your network.',
      },
      {
        heading: 'Self-hosted co-browsing for compliance',
        body: 'When your compliance team says "no SaaS screen-sharing tools," you have two options: build it yourself (3-4 months of engineering) or deploy PinConsole (5 minutes). PinConsole is built on rrweb, the same open-source technology that powers many commercial co-browsing tools, so you get production-grade DOM capture, two-way interaction, and session replay without writing a single line of infrastructure code.',
      },
      {
        heading: 'Start with the free AGPL version, upgrade when needed',
        body: 'The AGPL-3.0 version of PinConsole is fully functional — real-time visitor monitoring, two-way co-browsing, session replay, popup chat, and anti-bot protection. If you need to embed PinConsole in a proprietary product, commercial licenses are available. Contact the maintainer to discuss your use case.',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Is PinConsole a drop-in replacement for Upscope?',
        answer: 'PinConsole covers the core co-browsing and session replay use cases that Upscope serves. The main difference is deployment model: Upscope is SaaS, PinConsole is self-hosted. Migration requires deploying PinConsole on your infrastructure and integrating the visitor SDK into your website or app.',
      },
      {
        question: 'Can I migrate my Upscope session data to PinConsole?',
        answer: 'Direct data migration is not supported in v1. Session data from Upscope cannot be imported into PinConsole, but new sessions will be captured going forward. Contact the maintainer if you need custom migration assistance.',
      },
      {
        question: 'Does PinConsole work with single-page apps (React, Vue, Angular)?',
        answer: 'Yes. PinConsole\'s visitor SDK is framework-agnostic and works with any SPA or MPA. It uses rrweb for DOM capture, which handles dynamic DOM changes reliably.',
      },
    ],
  },
  cta: {
    title: 'Try PinConsole today',
    subtitle: 'Self-host in 5 minutes. No sign-up, no sales call, no data leaving your server.',
    primary: { label: 'Self-host in 5 min', href: '#top' },
    secondary: { label: 'Request consultation', href: '#consult' },
  },
};
