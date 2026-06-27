import type { SeoPageContent } from './types';

export const selfHostedSessionReplayEn: SeoPageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Self-Hosted Session Replay: Open Source Alternative to FullStory & Hotjar',
    description: 'Self-hosted session replay with PinConsole. Open-source, AGPL-3.0, deploy on your own infrastructure. Record, store, and replay every visitor session — data never leaves your servers.',
    ogTitle: 'Self-Hosted Session Replay — PinConsole Open Source Solution',
    ogDescription: 'Self-hosted session replay platform. AGPL-3.0, deploy in 5 minutes. Record every visitor session with rrweb, store in your own MinIO, replay anytime. No data leaves your infrastructure.',
  },
  hero: {
    h1: 'Self-Hosted Session Replay: Record and Replay on Your Infrastructure',
    subtitle: 'Deploy open-source session replay on your own servers in 5 minutes. PinConsole records every visitor session using rrweb, stores events in your MinIO, and replays them through a standard rrweb player — with no data leaving your network.',
  },
  comparison: {
    rows: [
      { label: 'Deployment', pinconsole: 'Self-hosted (your servers)', competitor: 'SaaS cloud' },
      { label: 'Data storage', pinconsole: 'Your own MinIO / S3 storage', competitor: 'Vendor\'s cloud' },
      { label: 'Retention control', pinconsole: 'Configurable (default 30 days)', competitor: 'Vendor plan limits' },
      { label: 'Recording engine', pinconsole: 'rrweb (open source DOM capture)', competitor: 'Proprietary' },
      { label: 'Selective screenshots', pinconsole: '1fps WebP (canvas/WebGL/iframe)', competitor: 'Full-page screenshots' },
      { label: 'Privacy controls', pinconsole: 'GDPR consent + erasure + IP truncation', competitor: 'Varies by vendor' },
      { label: 'Co-browsing included', pinconsole: 'Yes (two-way)', competitor: 'No (separate tool)' },
      { label: 'Real-time monitoring', pinconsole: 'Yes (live DOM stream)', competitor: 'Historical only' },
      { label: 'License', pinconsole: 'AGPL-3.0 (open source)', competitor: 'Proprietary' },
      { label: 'Pricing', pinconsole: 'Free. Commercial license available.', competitor: '$30-100K+ per year' },
    ],
  },
  content: {
    sections: [
      {
        heading: 'What is self-hosted session replay?',
        body: 'Session replay (also called session recording) captures every interaction a visitor makes on your website — mouse movements, clicks, scrolls, form inputs — and reconstructs them as a video-like playback. Self-hosted session replay means all this data is stored and processed on your own infrastructure, not a third-party analytics cloud. This is essential for companies that handle sensitive customer data under GDPR, HIPAA, SOC 2, or China\'s Personal Information Protection Law.',
      },
      {
        heading: 'Why self-host instead of SaaS (FullStory, Hotjar, LogRocket)?',
        body: 'SaaS session replay tools like FullStory, Hotjar, and LogRocket charge based on monthly pageviews or recordings. At scale, costs grow into six figures annually. More importantly, they store your visitors\' behavioral data on their servers — every click, scroll, and form input. For B2B SaaS companies serving finance, healthcare, or government clients, this creates an unacceptable data residency risk. Self-hosted session replay eliminates both the cost scaling and the data residency concern.',
      },
      {
        heading: 'PinConsole: session replay + co-browsing in one self-hosted stack',
        body: 'Unlike FullStory or Hotjar which only offer historical analytics, PinConsole combines session replay with real-time co-browsing. You can watch a visitor live, then seamlessly transition into a co-browsing session to assist them — all without switching tools. Session data is stored in your own MinIO, with configurable retention (default 30 days), GDPR erasure support, and IP truncation.',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'How does PinConsole\'s session replay compare to FullStory?',
        answer: 'PinConsole uses rrweb for DOM capture, which provides pixel-perfect replay through a standard rrweb player. The main difference is deployment: PinConsole is self-hosted with your own storage, while FullStory is SaaS. PinConsole also includes co-browsing and visitor monitoring — features FullStory doesn\'t offer.',
      },
      {
        question: 'How much storage do I need for session recordings?',
        answer: 'A typical session recording (rrweb event stream + selective screenshots) is approximately 100-500 KB per session. At 1,000 sessions/day, you would need roughly 3-15 GB/month of storage in your MinIO. Retention is configurable.',
      },
      {
        question: 'Can I export session data from PinConsole?',
        answer: 'Session recordings are stored as standard rrweb event data in your own MinIO bucket. You can access, export, or backup them using any S3-compatible tool. The metadata (sessions, visitors) is in your PostgreSQL database.',
      },
    ],
  },
  cta: {
    title: 'Deploy self-hosted session replay in 5 minutes',
    subtitle: 'No sign-up, no sales, no data leaving your servers.',
    primary: { label: 'Get started', href: '#top' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
