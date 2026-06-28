import type { PageContent } from '../../content/types';

export const enAbout: PageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'About PinConsole — Open-Source Self-Hosted Session Replay & Co-Browsing',
    description: 'PinConsole is an open-source, self-hosted real-time visitor monitoring and co-browsing platform. AGPL-3.0 licensed. Learn about our mission, values, and open-source commitment.',
    ogTitle: 'About PinConsole — Open-Source Self-Hosted Visitor Monitoring',
    ogDescription: 'Open-source (AGPL-3.0) self-hosted session replay, co-browsing, and real-time visitor monitoring. Data sovereignty first.',
  },
  hero: {
    eyebrow: 'About',
    h1: 'About PinConsole',
    subtitle: 'Open source, self-hosted, data sovereignty first. Building the open-source alternative to commercial visitor monitoring tools.',
  },
  sections: [
    {
      heading: 'Our Mission',
      body: 'PinConsole\'s mission is simple: provide a truly open-source, self-hosted alternative to commercial ToB real-time visitor monitoring, co-browsing, and session replay platforms.\n\nWe believe in data sovereignty — your visitor data should belong to you, not a SaaS vendor\'s cloud. We believe in open source — communities should be able to audit, modify, and trust the infrastructure they depend on. We believe in sustainability — funding open-source development through commercial licensing without compromising the AGPL-3.0 core.',
    },
    {
      heading: 'Tech Stack',
      body: 'Backend: Go + Gin + coder/websocket + PostgreSQL + Redis + MinIO\nFrontend: Vue 3 + TypeScript + Vite + Pinia\nVisitor SDK: TypeScript + rrweb (full DOM capture)\nDeployment: Single binary (Go embed), Docker Compose one-click start',
    },
    {
      heading: 'Core Features',
      body: '• Real-time visitor monitoring — see what visitors are doing on your site\n• Two-way co-browsing — operators can directly interact with visitor browsers\n• Session replay — full DOM recording with playback controls\n• Proactive popup chat — engage visitors at the right moment\n• Privacy protection — selective masking, GDPR compliance, right to be forgotten\n• Defense-in-depth — four-layer anti-bot protection system',
    },
    {
      heading: 'Open Source Commitment',
      body: 'PinConsole is released under the AGPL-3.0 license. This means:\n\n• You can self-host it freely\n• You can audit every line of code\n• You can fork and maintain your own branch\n• If you offer a modified version as a SaaS service, you must release your modifications under the same license\n\nFor teams that need to embed PinConsole in proprietary products, standard commercial licenses are available.',
    },
    {
      heading: 'Contact the Maintainer',
      body: 'PinConsole is independently developed and maintained by Rong Zhu.\n\n• GitHub: https://github.com/iannil/pinconsole\n• Technical consulting, compliance assessment, commercial licensing: reach out via the form below',
    },
  ],
  cta: {
    title: 'Get Started with PinConsole',
    subtitle: 'Self-hosted in 5 minutes. AGPL-3.0. Your data, your servers.',
    primary: { label: 'Start on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Contact the maintainer', href: '#consult' },
  },
};
