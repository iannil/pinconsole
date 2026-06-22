import type { PageContent } from './types';

export const en: PageContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'pinconsole — Your visitors, your data.',
    description:
      'Open source ToB real-time visitor monitoring + operator interaction + session replay. AGPL-3.0, self-hosted, data never leaves your infrastructure.',
    ogTitle: 'pinconsole — Your visitors, your data.',
    ogDescription:
      'Open source ToB real-time visitor monitoring + operator interaction + session replay. Self-hosted, AGPL-3.0, data never leaves.',
  },
  nav: {
    links: [
      { label: 'Features', href: '#features' },
      { label: 'Data Sovereignty', href: '#data-sovereignty' },
      { label: 'Self-host', href: '#self-host' },
      { label: 'Roadmap', href: '#roadmap' },
      { label: 'FAQ', href: '#faq' },
    ],
    cta: 'Request consultation',
    localeSwitch: { label: '中', href: '/' },
  },
  hero: {
    eyebrow: 'AGPL-3.0 · Self-hosted · Data Sovereignty',
    h1: 'Your visitors,\nyour data.',
    h2:
      'Open source ToB real-time visitor monitoring + operator interaction + session replay. Self-hosted, AGPL-3.0, data never leaves your infrastructure.',
    cta: {
      primary: { label: 'Request consultation', href: '#consult' },
      secondary: { label: 'Self-host in 5 min', href: '#self-host' },
      tertiary: { label: 'GitHub ★', href: 'https://github.com/iannil/pinconsole' },
    },
    videoSrc: '/demo.mp4',
    videoPoster: '/screenshots/dashboard.png',
  },
  problem: {
    eyebrow: 'Why we built this',
    title: 'Three things that hurt with commercial SaaS',
    subtitle:
      "If you're paying 30-100K USD/year for a ToB customer interaction SaaS, you've probably felt all of these.",
    items: [
      {
        icon: 'lock',
        title: 'Your data lives in someone else\'s server',
        description:
          "Visitor behavior, operator conversations, session replays — all on the SaaS vendor's infrastructure. During compliance review you can only hope they don't leak, get breached, or sell upstream.",
      },
      {
        icon: 'lock-open',
        title: 'Features are locked',
        description:
          'Want to add a field, change a workflow, integrate an internal system? You wait for vendor roadmap, pay extra, ship slower. Your product cadence is gated by someone else.',
      },
      {
        icon: 'trending-up',
        title: 'Annual price hikes, migration lock-in',
        description:
          '30-100K USD/year, 15% annual increase on renewal. Want to switch? Data export is broken, team retrains, operations pause for a quarter.',
      },
    ],
  },
  features: {
    eyebrow: 'v1 shipped',
    title: 'What commercial competitors do, v1 does too',
    subtitle:
      '90+ commits, 65 e2e tests green. Five core capabilities in v1, each verified end-to-end.',
    items: [
      {
        icon: 'eye',
        title: 'Real-time visitor monitoring',
        description:
          'rrweb full capture: DOM mutations + mouse + click + scroll + blurred form values. 1fps WebP selective screenshots (canvas / WebGL / cross-origin iframe).',
        bullets: [
          'Full DOM + selective screenshots',
          'Behavior serialization (stable rrweb node ID)',
          'Same-origin SDK distribution, single JS file',
        ],
        screenshot: '/screenshots/dashboard.png',
      },
      {
        icon: 'arrows-out-cardinal',
        title: 'Co-browsing (bidirectional)',
        description:
          'Operators can highlight / click / scroll / fill / navigate. 300ms debounce balances UX and bandwidth. Visitor has emergency exit at all times.',
        bullets: [
          'Node ID selector (no CSS/XPath)',
          'Form-fill debounce 300ms',
          'Navigation takeover + cross-page session resume',
        ],
        screenshot: '/screenshots/cobrowse-active.png',
      },
      {
        icon: 'video',
        title: 'Session replay',
        description:
          'MinIO archive for event stream + selective screenshots. 30-day default retention, configurable. GDPR erasure endpoint. Standard rrweb-player.',
        bullets: [
          'MessagePack-compressed event archive',
          'Default 30-day retention, configurable',
          'GDPR right-to-be-forgotten one-shot',
        ],
        screenshot: '/screenshots/replay.png',
      },
      {
        icon: 'chat-circle-dots',
        title: 'Popup + bidirectional chat',
        description:
          'Message channel persisted to PostgreSQL. Popups support text / link / navigation. Chat history bound to session.',
        bullets: [
          'Shared popup + chat channel',
          'History searchable by session',
          'Operator 1:1 claim lock prevents conflicts',
        ],
        screenshot: '/screenshots/chat.png',
      },
      {
        icon: 'shield-check',
        title: 'Anti-scrape + GDPR',
        description:
          'Rate limit + UA blocklist + behavior analysis + canvas/WebGL fingerprint. Consent opt-in + IP truncation + co-browse banner.',
        bullets: [
          'Redis sliding-window rate limit',
          'GDPR consent opt-in by default',
          'IP truncated /24, behavior log retention configurable',
        ],
        screenshot: '/screenshots/privacy.png',
      },
    ],
  },
  dataSovereignty: {
    eyebrow: 'Our entire stance',
    title: 'Why your data is actually yours',
    subtitle:
      'Three engineering decisions make "data sovereignty" a verifiable fact, not a marketing line.',
    pillars: [
      {
        icon: 'scale',
        title: 'AGPL-3.0 strong copyleft',
        description:
          'Any modification to pinconsole must be open-sourced. Cloud vendors cannot take it to make a SaaS — license-level hard protection. You get real open source, not "source available".',
      },
      {
        icon: 'stack',
        title: 'Standard stack, no lock-in',
        description:
          'PostgreSQL 16 · Redis 7 · MinIO · Go 1.22 · Vue 3. Every layer is industry standard, schema is in your hands, data is exportable anytime.',
      },
      {
        icon: 'certificate',
        title: 'Compliance-ready',
        description:
          'GDPR consent opt-in + right to erasure + IP truncation; HttpOnly cookie + bcrypt; command authorization + popup URL allowlist; end-to-end WS trace_id.',
      },
    ],
    architectureAlt:
      'pinconsole architecture: visitor SDK → pinconsole-server → PostgreSQL/Redis/MinIO, all inside your infrastructure',
  },
  selfHost: {
    eyebrow: 'Running in 5 minutes',
    title: 'One docker compose,\non your own server',
    subtitle:
      'No cloud vendor dependency, no marketplace integration, no "activation" flow. Clone, run docker compose, single binary starts up.',
    code: `git clone https://github.com/iannil/pinconsole
cd pinconsole
cp .env.example .env

make docker-up build-frontend build
./server/bin/pinconsole-server

# Visitor landing → http://localhost:8080/
# Operator admin  → http://localhost:8080/admin`,
    docsLink: {
      label: 'Full deployment docs →',
      href: 'https://github.com/iannil/pinconsole#readme',
    },
  },
  roadmap: {
    eyebrow: 'Transparent stance',
    title: 'v1 done. What\'s next.',
    subtitle:
      'We publicly commit to what we will and won\'t build. The stance doesn\'t change under commercial pressure — that\'s the point of an OSS alternative.',
    columns: [
      {
        status: 'shipped',
        title: '✅ Shipped (v1)',
        items: [
          'Real-time visitor monitoring (full rrweb)',
          'Co-browsing (cursor/click/scroll/fill/navigate)',
          'Session archive + replay',
          'Popup push + bidirectional chat',
          'Auth + multi-operator claim/release lock',
          'Anti-scrape (rate limit + UA + behavior + fingerprint)',
          'GDPR consent + erasure + IP truncation',
          'Observability (LifecycleTracker + trace_id)',
          'Bilingual i18n (zh/en)',
          'Docker Compose one-command deploy',
        ],
      },
      {
        status: 'coming',
        title: '🚧 Planned (post-v1)',
        items: [
          'Custom domain (DNS verification + ACME)',
          'Page editor (low-code / drag-drop)',
          'Tauri desktop (Win + Mac)',
          'SSO / SAML / OIDC (enterprise)',
          'Anti-scrape hardening (CAPTCHA + honeypot)',
          'Analytics dashboard (funnel / heatmap)',
          'Redis Pub/Sub multi-instance hub',
        ],
      },
      {
        status: 'out-of-scope',
        title: '❌ Never doing',
        items: [
          'Multi-tenant SaaS',
          'Subscription billing',
          'Self-signup flow',
          'Managed cloud service',
          '—— stance unchanged',
        ],
      },
    ],
  },
  faq: {
    eyebrow: 'Common questions',
    title: 'What decision makers ask',
    subtitle:
      'If your question isn\'t here, leave a message in the form. We reply within 48h.',
    items: [
      {
        question: 'Is AGPL-3.0 safe for our commercial use?',
        answer:
          'AGPL requires open-sourcing modifications when you offer the service to external users. Internal company use (no external service) does not trigger this. Serving your own visitors via pinconsole is fine — you\'re not selling pinconsole. Only cloud vendors taking it to make a SaaS need to open-source — which is the license\'s protective mechanism.',
      },
      {
        question: 'Solo developer — how long will this be maintained?',
        answer:
          'v1 has 90+ commits covering end-to-end slices. PLAN.md §8 publishes the post-v1 roadmap. Consultation revenue sustains ongoing maintenance. If the project is critical to you, consider paid consultation/customization — your payment is the best guarantee of project sustainability.',
      },
      {
        question: 'Do you offer custom development?',
        answer:
          'Yes — that\'s the core of consultation. Describe your needs in the form, we reply within 48h with scoping. Custom dev is billable, all code returns to the project under AGPL-3.0 (your private deployment doesn\'t need to be open-sourced).',
      },
      {
        question: 'Can you host our deployment?',
        answer:
          'v1 doesn\'t offer hosting, and the long-term stance is to never offer it. We avoid competing with OSS users. We can recommend partner deployment providers (you choose), or assist you in standing up your own.',
      },
      {
        question: 'How do we migrate from X?',
        answer:
          'v1 doesn\'t provide automated migration tools — competitor schema differences are too large. Consultation can include data migration assessment, scoped per case. Operator training cost is mainly the admin UI, about 1-2 hours to onboard.',
      },
      {
        question: '500 concurrent is not enough — what then?',
        answer:
          'v1 is single-instance hub (process-local map). Multi-instance needs a Redis Pub/Sub bus, a post-v1 slice. If your per-room concurrency needs > 500, mention it in consultation — we can prioritize or scope custom work.',
      },
      {
        question: 'Does it run on k8s / intranet / private cloud?',
        answer:
          'Yes. docker-compose is a reference deployment; k8s / bare metal / reverse proxy / TLS / backup / monitoring are all in your control. OSS project doesn\'t provide production topology references, but release binaries are fail-secure and /healthz + /readyz expose dependency health checks.',
      },
      {
        question: 'Can it pass ISO 27001 / SOC 2 / regional compliance?',
        answer:
          'Product-layer compliance is ready (GDPR / bcrypt / audit log / consent). ISO 27001 / SOC 2 / regional certifications need deployment-environment-specific assessment. Consultation can assist with compliance documentation, architecture mapping, and third-party audit support.',
      },
      {
        question: 'Does it support mobile visitors?',
        answer:
          'Yes. rrweb capture covers mobile evergreen browsers. Operator side is desktop-only in v1 (mobile operator app is post-v1).',
      },
      {
        question: 'What if AGPL is incompatible with our internal GPL code?',
        answer:
          'Case-by-case. Consultation can include license compatibility analysis. If AGPL is fully unacceptable, a dual-license path may be enabled in the future (not currently active).',
      },
    ],
  },
  finalCTA: {
    eyebrow: 'Let\'s talk about your use case',
    title: 'We\'ll reply within 48 hours',
    subtitle:
      'Submissions are stored in maintainer-self-hosted Cloudflare D1 (Asia region), not shared with third parties. We don\'t send newsletters — we only reply in the context of your scenario.',
    form: {
      nameLabel: 'Name *',
      namePlaceholder: 'Jane Doe',
      companyLabel: 'Company *',
      companyPlaceholder: 'Acme Inc.',
      contactLabel: 'Contact (phone or email) *',
      contactPlaceholder: '+1 555-xxxx or you@company.com',
      purposeLabel: 'Purpose *',
      purposes: [
        { value: 'evaluate', label: 'Evaluating as SaaS replacement' },
        { value: 'self-host', label: 'Self-hosted deployment consulting' },
        { value: 'custom', label: 'Custom development' },
        { value: 'compliance', label: 'Compliance consulting (GDPR/SOC2/ISO)' },
        { value: 'other', label: 'Other' },
      ],
      messageLabel: 'Message (optional)',
      messagePlaceholder:
        'What platform are you on? What problem are you solving? When do you want to ship?',
      submitLabel: 'Submit',
      privacyNote:
        'By submitting you agree we may use your message to reply. Data stored in maintainer-self-hosted Cloudflare D1, not shared, deletable on request.',
      successMessage: 'Got it. We\'ll reply to your contact within 48 hours.',
      errorMessage:
        'Submission failed. Please try again later or email contact@pinconsole.example.com directly.',
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
          { label: 'Self-host', href: '#self-host' },
          { label: 'Roadmap', href: '#roadmap' },
        ],
      },
      {
        title: 'Resources',
        links: [
          { label: 'GitHub repo', href: 'https://github.com/iannil/pinconsole' },
          { label: 'Deployment docs', href: 'https://github.com/iannil/pinconsole#readme' },
          { label: 'FAQ', href: '#faq' },
          { label: 'Consultation', href: '#consult' },
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
    sourceNote: '© 2026 pinconsole. Built with Calm Crafted design system.',
  },
};
