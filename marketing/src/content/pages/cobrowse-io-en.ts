import type { SeoPageContent } from './types';

export const cobrowseIoEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole vs Cobrowse.io: Open Source Self-Hosted Alternative',
    description: 'Compare PinConsole (open-source, self-hosted, AGPL-3.0) with Cobrowse.io. Both offer co-browsing. Only PinConsole gives you full data sovereignty, session replay, and no per-seat pricing.',
    ogTitle: 'PinConsole vs Cobrowse.io — Open Source Co-Browsing Alternative',
    ogDescription: 'Self-hosted alternative to Cobrowse.io. AGPL-3.0, your data on your servers. Co-browsing, session replay, visitor monitoring, and anti-bot protection.',
  },
  hero: {
    h1: 'PinConsole vs Cobrowse.io: Open Source Alternative',
    subtitle: 'Cobrowse.io is the leading co-browsing SDK, but it\'s proprietary and SaaS-only. PinConsole gives you the same co-browsing capabilities — plus session replay and visitor monitoring — all self-hosted under AGPL-3.0.',
  },
  comparison: {
    rows: [
      { label: 'Hosting model', pinconsole: 'Self-hosted (your infrastructure)', competitor: 'SaaS cloud + on-prem (enterprise)' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary SDK' },
      { label: 'Data sovereignty', pinconsole: 'Data never leaves your servers', competitor: 'Data via cloud relay (unless on-prem plan)' },
      { label: 'Co-browsing', pinconsole: 'Full two-way (click/scroll/fill/navigate)', competitor: 'Full two-way co-browsing' },
      { label: 'Session replay', pinconsole: 'Included (rrweb-based)', competitor: 'Not included (separate tool needed)' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshot', competitor: 'Not included' },
      { label: 'Popups & chat', pinconsole: 'Push notifications + two-way chat', competitor: 'Not included' },
      { label: 'Anti-bot', pinconsole: 'Rate limit + UA + behavior + fingerprint', competitor: 'Not included' },
      { label: 'Pricing', pinconsole: 'Free (AGPL-3.0). Commercial license available.', competitor: 'Per-agent per-month or enterprise annual' },
      { label: 'Deployment', pinconsole: 'Docker compose, 5 minutes', competitor: 'SaaS SDK snippet or on-prem setup' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'Why switch from Cobrowse.io to PinConsole?',
        body: 'Cobrowse.io is a polished co-browsing SDK, but its SaaS model means your customers\' screen data routes through their cloud. If your compliance framework (GDPR, SOC 2, HIPAA, or China\'s Personal Information Protection Law) requires data to stay in-region or on-premise, Cobrowse.io\'s on-prem plan can help — but it comes with enterprise pricing. PinConsole is free and open-source, with the same co-browsing capabilities plus session replay and visitor monitoring included at no extra cost.',
      },
      {
        heading: 'More than co-browsing — a complete customer-ops platform',
        body: 'Cobrowse.io focuses on co-browsing only. PinConsole is a full customer operations platform: real-time visitor monitoring (see what every visitor does on your site), session replay (record and replay any session), popup chat (proactive messaging), and anti-bot protection. All in a single self-hosted binary.',
      },
      {
        heading: 'Open source means no vendor lock-in',
        body: 'With Cobrowse.io, you\'re tied to their pricing, their feature roadmap, and their compliance certifications. With PinConsole, the AGPL-3.0 license guarantees you can always self-host, audit the code, and fork if needed. Commercial licenses are available for proprietary embedding.',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'Does PinConsole support the same integration methods as Cobrowse.io?',
        answer: 'PinConsole uses a visitor SDK snippet (similar to Cobrowse.io\'s snippet) that you add to your website. It works with any framework and does not require native mobile SDKs in v1.',
      },
      {
        question: 'Can I use both PinConsole and Cobrowse.io simultaneously?',
        answer: 'Technically yes, but there\'s no need — PinConsole covers co-browsing, session replay, monitoring, and chat in one self-hosted stack. Running both would add complexity with no benefit.',
      },
      {
        question: 'Is PinConsole\'s co-browsing as reliable as Cobrowse.io?',
        answer: 'PinConsole uses rrweb for DOM capture, the same technology used by many production co-browsing tools. The two-way co-browsing (cursor sync, click forwarding, form filling, page navigation) has been end-to-end tested with 65+ e2e tests passing.',
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
