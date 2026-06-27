import type { SeoPageContent } from './types';

export const selfHostedCoBrowsingEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Self-Hosted Co-Browsing: Open Source Solution for Secure Screen Sharing',
    description: 'Self-hosted co-browsing with PinConsole. Open-source, AGPL-3.0, deploy on your own infrastructure. Two-way co-browsing, session replay, and visitor monitoring — no data leaves your servers.',
    ogTitle: 'Self-Hosted Co-Browsing — PinConsole Open Source Solution',
    ogDescription: 'Self-host, open-source co-browsing platform. AGPL-3.0, deploy in 5 minutes. Full two-way co-browsing, session replay, and visitor monitoring. Your data, your servers.',
  },
  hero: {
    h1: 'Self-Hosted Co-Browsing: Secure Screen Sharing on Your Infrastructure',
    subtitle: 'Deploy open-source co-browsing on your own servers in 5 minutes. PinConsole gives you two-way co-browsing, session replay, and real-time visitor monitoring — all under AGPL-3.0, with no data leaving your network.',
  },
  comparison: {
    rows: [
      { label: 'Deployment', pinconsole: 'Your own servers (Docker compose)', competitor: 'SaaS cloud or enterprise on-prem' },
      { label: 'Data residency', pinconsole: '100% on your infrastructure', competitor: 'Third-party servers (unless on-prem)' },
      { label: 'Co-browsing', pinconsole: 'Two-way: click, scroll, fill, navigate', competitor: 'Varies by vendor' },
      { label: 'Session replay', pinconsole: 'Included (rrweb-based)', competitor: 'Often separate tool / add-on' },
      { label: 'Visitor monitoring', pinconsole: 'Real-time DOM + 1fps screenshots', competitor: 'Usually not included' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Pricing', pinconsole: 'Free. Commercial license available.', competitor: '$30-100K+ per year' },
      { label: 'Setup time', pinconsole: '5 minutes (docker compose)', competitor: 'SDK integration + vendor onboarding' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'What is self-hosted co-browsing?',
        body: 'Self-hosted co-browsing (also called co-navigation or collaborative browsing) lets support agents see and interact with a visitor\'s browser in real time — highlighting elements, clicking buttons, filling forms, and navigating pages — all running on the company\'s own infrastructure instead of a third-party cloud. This is critical for compliance with GDPR, HIPAA, SOC 2, China\'s Personal Information Protection Law, and other regulations that restrict where customer data can be processed and stored.',
      },
      {
        heading: 'Why companies choose self-hosted over SaaS co-browsing',
        body: 'SaaS co-browsing tools (Upscope, Cobrowse.io, Surfly) are convenient, but they require customer screen data to pass through third-party servers. For finance, insurance, healthcare, and government, this is a dealbreaker. Self-hosted co-browsing solves this by running entirely on your network. PinConsole makes self-hosting practical with a single docker-compose command — no Kubernetes, no cloud services, no vendor dependencies.',
      },
      {
        heading: 'PinConsole: open-source co-browsing with full feature parity',
        body: 'PinConsole is not a stripped-down "open core" version with limited features. The AGPL-3.0 version includes everything: two-way co-browsing (cursor sync, click forwarding, form filling, page navigation), session replay (record and replay any session), real-time visitor monitoring (watch every visitor\'s DOM changes), popup chat (proactive messaging), and anti-bot protection (rate limiting, UA blocking, behavior analysis, fingerprinting). All in a single self-hosted binary.',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'What infrastructure do I need to self-host PinConsole?',
        answer: 'You need Docker (or a Linux server with Go runtime), PostgreSQL 16+, Redis 7+, and MinIO (or S3-compatible storage). The docker-compose.yml file in the repo sets up all dependencies automatically.',
      },
      {
        question: 'Does self-hosted co-browsing work behind a firewall?',
        answer: 'Yes. PinConsole is designed for air-gapped and internal network deployments. The visitor SDK connects to your server via WebSocket — if your website is accessible, co-browsing works. No external service calls are required.',
      },
      {
        question: 'Can I customize the co-browsing experience?',
        answer: 'Since PinConsole is open-source (AGPL-3.0), you can customize anything. The codebase is a clean Go + TypeScript monorepo. Custom development services are available through consultation.',
      },
    ],
  },
  cta: {
    title: 'Deploy self-hosted co-browsing in 5 minutes',
    subtitle: 'No sign-up, no sales, no data leaving your servers.',
    primary: { label: 'Get started', href: '#top' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
