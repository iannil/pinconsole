import type { PageContent } from './types';
import { enNavLinks, enNavCta, enLocaleSwitch } from './nav-shared';

export const en: PageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'PinConsole — Your visitors, your data.',
    description:
      'Open source ToB real-time visitor monitoring + operator interaction + session replay platform. AGPL-3.0, self-hosted, data never leaves your infrastructure. An open-source alternative to Upscope and Cobrowse.io.',
    ogTitle: 'PinConsole — Your visitors, your data.',
    ogDescription:
      'Open source ToB real-time visitor monitoring + operator interaction + session replay. Self-hosted, AGPL-3.0, data never leaves.',
  },
  nav: { links: [...enNavLinks], cta: enNavCta, localeSwitch: { ...enLocaleSwitch } },
  hero: {
    eyebrow: 'AGPL-3.0 · Self-hosted · Data Sovereignty',
    h1: 'Your visitors,\nyour data.',
    h2:
      'PinConsole is an open-source, self-hosted alternative to Upscope and Cobrowse.io for real-time co-browsing, session replay, and visitor monitoring. Commercial customer-ops SaaS at $30-100K/year locks your data, locks features, and raises prices every year.',
    cta: {
      primary: { label: 'Request consultation', href: '#consult' },
      secondary: { label: 'Self-host in 5 min', href: '#data-sovereignty' },
      tertiary: { label: 'GitHub ★', href: 'https://github.com/iannil/pinconsole' },
    },
  },
  features: {
    eyebrow: 'Shipped',
    title: 'Everything the commercial alternatives do, already built',
    subtitle:
      '90+ commits, 65 e2e tests green. The five capabilities below are all end-to-end verified and can run on your server today.',
    items: [
      {
        icon: 'eye',
        title: 'Real-time visitor monitoring',
        description:
          'rrweb captures all DOM mutations + mouse + clicks + scrolls + unfocused form values. 1fps WebP selective screenshots (canvas / WebGL / cross-origin iframes).',
        bullets: [
          'Full DOM + selective screenshots',
          'Behavior serialization (stable rrweb node IDs)',
          'Visitor SDK same-origin, single JS file',
        ],
        screenshot: '/screenshots/dashboard.webp',
      },
      {
        icon: 'arrows-out-cardinal',
        title: 'Two-way co-browsing',
        description:
          'Operators can highlight / click / scroll / fill / navigate. 300ms debounce balances UX and bandwidth. Visitors can emergency-exit any time.',
        bullets: [
          'Node ID selectors (no CSS/XPath)',
          'Form-fill debounce 300ms',
          'Navigation takeover + cross-page session handoff',
        ],
        screenshot: '/screenshots/cobrowse-active.webp',
      },
      {
        icon: 'video',
        title: 'Session replay',
        description:
          'MinIO archives event streams + selective screenshots. 30-day default, configurable. GDPR delete endpoint. Standard rrweb-player replayer.',
        bullets: [
          'Event stream compression (MessagePack)',
          '30-day default retention, configurable',
          'GDPR right-to-be-forgotten one-shot',
        ],
        screenshot: '/screenshots/replay.webp',
      },
      {
        icon: 'chat-circle-dots',
        title: 'Popups + two-way chat',
        description:
          'Message channel persisted to PostgreSQL. Popups support text / links / navigation. Chat history bound to session.',
        bullets: [
          'Popups and chat share one channel',
          'History searchable by session',
          'Operator 1:1 locking to prevent conflicts',
        ],
        screenshot: '/screenshots/chat.webp',
      },
      {
        icon: 'shield-check',
        title: 'Anti-bot + GDPR',
        description:
          'Rate limit + UA blocklist + behavior analysis + canvas/WebGL fingerprint. Consent opt-in + IP truncation + co-browse banner.',
        bullets: [
          'Redis sliding-window rate limit',
          'GDPR consent opt-in default',
          'IP truncation /24, retention configurable',
        ],
        screenshot: '/screenshots/privacy.webp',
      },
    ],
  },
  dataSovereignty: {
    eyebrow: 'Verifiable data sovereignty',
    title: 'Why your data is actually yours',
    subtitle:
      'Three layers of design decisions turn "data sovereignty" from marketing copy into an engineering fact you can verify.',
    pillars: [
      {
        icon: 'scale',
        title: 'AGPL-3.0 strong copyleft',
        description:
          'Any modification to PinConsole must be open-sourced. Cloud vendors cannot turn it into SaaS — this is license-level hard protection. You get real open source, not "source available".',
      },
      {
        icon: 'stack',
        title: 'Standard stack, no lock-in',
        description:
          'PostgreSQL 16 · Redis 7 · MinIO · Go 1.22 · Vue 3. Every layer is industry-standard, the schema is in your hands, data exports anytime.',
      },
      {
        icon: 'certificate',
        title: 'Compliance-ready',
        description:
          'GDPR consent opt-in + right-to-be-forgotten + IP truncation; HttpOnly cookie + bcrypt; command authorization + popup URL allowlist; WS trace_id end-to-end observability.',
      },
    ],
    architectureAlt:
      'PinConsole architecture: visitor SDK → pinconsole-server → PostgreSQL/Redis/MinIO, all inside your infrastructure',
  },
  selfHost: {
    eyebrow: 'Running in 5 minutes',
    title: 'One docker compose,\non your own server',
    subtitle:
      'No cloud vendor dependency, no marketplace integration, no activation flow. Clone, run docker compose, single binary is up.',
    code: `git clone https://github.com/iannil/pinconsole
cd pinconsole
cp .env.example .env

make docker-up build-frontend build
./server/bin/pinconsole-server

# visitor landing → http://localhost:8080/
# operator admin → http://localhost:8080/admin`,
    docsLink: { label: 'Full deployment docs →', href: 'https://github.com/iannil/pinconsole#readme' },
  },
  roadmap: {
    eyebrow: 'Transparent stance',
    title: 'Shipped. What comes next.',
    subtitle:
      'We publicly commit to what we build and what we won\'t. Stance doesn\'t change under commercial pressure — that\'s the point of an OSS alternative.',
    columns: [
      {
        status: 'shipped',
        title: 'Shipped',
        items: [
          'Real-time visitor monitoring (full rrweb)',
          'Two-way co-browsing (cursor/click/scroll/fill/navigate)',
          'Session archive + replay',
          'Popups + two-way chat',
          'Auth + multi-operator claim/release locks',
          'Anti-bot (rate limit + UA + behavior + fingerprint)',
          'GDPR consent + right-to-be-forgotten + IP truncation',
          'Observability (LifecycleTracker + trace_id)',
          'Bilingual zh/en i18n',
          'Docker Compose one-shot deploy',
        ],
      },
      {
        status: 'coming',
        title: 'Planned',
        items: [
          'Custom domains (DNS verification + ACME)',
          'Page editor (low-code / drag-drop)',
          'Tauri desktop (Win + Mac)',
          'SSO / SAML / OIDC (enterprise)',
          'Anti-bot hardening (CAPTCHA + honeypot)',
          'Analytics dashboard (funnel / heatmap)',
          'Redis Pub/Sub multi-instance hub',
        ],
      },
      {
        status: 'out-of-scope',
        title: 'Explicitly not building',
        items: [
          'Multi-tenant SaaS',
          'Subscription billing',
          'Sign-up flow / self-signup',
          'Hosted cloud service',
          'Stance unchanged',
        ],
      },
    ],
  },
  faq: {
    eyebrow: 'Common questions',
    title: 'What decision-makers ask',
    subtitle: 'If you don\'t find your answer here, drop a note and we reply within 48h.',
    items: [
      {
        question: 'Is AGPL-3.0 safe for our commercial use?',
        answer:
          'AGPL requires open-sourcing modifications when you offer the service externally. Internal company use (not exposing the service to the public) does not trigger this. Running PinConsole to serve your own visitors, without selling PinConsole itself, has zero compliance risk. Only cloud vendors trying to SaaS-ify it must open-source — that\'s exactly the license\'s protection.',
      },
      {
        question: 'Solo maintainer — how long can this last?',
        answer:
          'v1 ships 90+ commits of end-to-end slices, with post-v1 roadmap public in PLAN.md §8. Consultation revenue funds continued maintenance. If this project is critical to you, consider paid consulting/customization — your payment is the best guarantee of sustainability.',
      },
      {
        question: 'Can you do custom development?',
        answer:
          'Yes — that\'s the core of consultation. Describe your needs in the form, we respond within 48h with a scoping assessment. Custom dev is billed separately. All code returns to the project under AGPL-3.0 (your private deployment doesn\'t need to be open-sourced).',
      },
      {
        question: 'Can you host our deployment?',
        answer:
          'Not in v1, and not in the long-term stance either. Avoids competing with OSS users. We can recommend deployment partners (your choice), or assist you in standing up your own.',
      },
      {
        question: 'What if 500 concurrent isn\'t enough?',
        answer:
          'v1 is a single-instance hub (process-local map). Multi-instance requires a Redis Pub/Sub bus, which is a post-v1 slice. If your concurrent room demand exceeds 500, mention it in consultation — we can prioritize scheduling or customize.',
      },
      {
        question: 'Can it pass MLPS 2.0 / ISO 27001?',
        answer:
          'Product-layer compliance is ready (GDPR / bcrypt / audit logs / consent). MLPS / ISO 27001 requires evaluating your deployment environment. Consultation can assist with compliance documentation, architecture mapping, and third-party assessment support.',
      },
    ],
  },
  finalCTA: {
    eyebrow: 'Discuss your use case',
    title: 'We don\'t sell subscriptions. We discuss your use case.',
    subtitle:
      'After submission, data lives in the maintainer\'s self-hosted Cloudflare D1 (Asia region), not shared with third parties. We don\'t send marketing email — only contextual replies to your scenario.',
    form: {
      nameLabel: 'Name *',
      namePlaceholder: 'Jane Doe',
      companyLabel: 'Company *',
      companyPlaceholder: 'Acme Inc.',
      contactLabel: 'Contact (phone or email) *',
      contactPlaceholder: '+1 555-0100 or you@company.com',
      purposeLabel: 'Purpose *',
      purposes: [
        { value: 'evaluate', label: 'Evaluate as SaaS alternative' },
        { value: 'self-host', label: 'Self-hosted deployment advisory' },
        { value: 'custom', label: 'Custom development' },
        { value: 'compliance', label: 'Compliance (GDPR/MLPS/ISO)' },
        { value: 'other', label: 'Other' },
      ],
      messageLabel: 'Message (optional)',
      messagePlaceholder: 'What platform are you on? What problem are you solving? Target launch date?',
      submitLabel: 'Submit consultation',
      privacyNote:
        'By submitting you agree we use your message for consultation reply. Data stored in maintainer self-hosted Cloudflare D1, not shared with third parties, deletable by email request.',
      successMessage: 'Received. We\'ll reply to your contact within 48 hours.',
      errorMessage: 'Submission failed — please try again later or email contact@pinconsole.com.',
    },
  },
  footer: {
    tagline: 'Your visitors, your data.',
    columns: [
      {
        title: 'Product',
        links: [
          { label: 'Features', href: '#features' },
          { label: 'Data Sovereignty', href: '#data-sovereignty' },
          { label: 'Roadmap', href: '#roadmap' },
        ],
      },
      {
        title: 'Compare',
        links: [
          { label: 'vs Upscope', href: '/en/alternatives/upscope' },
          { label: 'vs Cobrowse.io', href: '/en/alternatives/cobrowse-io' },
        ],
      },
      {
        title: 'Resources',
        links: [
          { label: 'Self-hosted co-browsing', href: '/en/co-browsing/self-hosted' },
          { label: 'Self-hosted session replay', href: '/en/session-replay/self-hosted' },
          { label: 'GitHub repo', href: 'https://github.com/iannil/pinconsole' },
          { label: 'Deploy docs', href: 'https://github.com/iannil/pinconsole#readme' },
          { label: 'FAQ', href: '#faq' },
          { label: 'Consult', href: '#consult' },
        ],
      },
      {
        title: 'Legal',
        links: [
          { label: 'AGPL-3.0 License', href: 'https://github.com/iannil/pinconsole/blob/master/LICENSE' },
          { label: 'Privacy', href: '#consult' },
        ],
      },
    ],
    license: 'AGPL-3.0-or-later',
    sourceNote: '© 2026 PinConsole. Built with Calm Crafted design system.',
  },
};
