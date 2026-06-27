import type { BlogContent } from './types';

export const agplVsMitEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'AGPL-3.0 vs MIT: Why We Chose AGPL for Our Open Source Project',
    description: 'A honest look at why PinConsole chose the AGPL-3.0 license over MIT or Apache 2.0 — the strategic reasoning, the trade-offs, and how commercial licensing works for proprietary use.',
    ogTitle: 'AGPL-3.0 vs MIT: Why We Chose AGPL for PinConsole',
    ogDescription: 'Why AGPL-3.0 is the right license for an open-source self-hosted alternative to SaaS tools like FullStory and Hotjar.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-27',
    readingTime: '6 min read',
    tags: ['Open Source', 'Licensing', 'AGPL', 'Business'],
  },
  hero: {
    h1: 'AGPL-3.0 vs MIT: Why We Chose AGPL for Our Open Source Project',
    subtitle: 'A transparent look at the licensing decision behind PinConsole — and what it means for users, contributors, and commercial adopters.',
  },
  content: {
    sections: [
      {
        heading: 'The short answer',
        body: "We chose AGPL-3.0 because PinConsole is positioned as an open-source alternative to proprietary SaaS tools like FullStory, Hotjar, and LogRocket. If we used MIT or Apache 2.0, a cloud provider could take PinConsole, wrap it in a SaaS offering, and compete directly with the open-source project — without contributing anything back. AGPL-3.0 prevents this by requiring anyone who modifies and distributes the software (including over a network) to release their modifications under the same license.\n\nIn short: AGPL protects the project from being commoditized by SaaS providers while keeping it fully open and free for self-hosted users.",
      },
      {
        heading: 'Why not MIT?',
        body: "MIT is the most permissive license — you can do almost anything with MIT-licensed code, including using it in proprietary closed-source products. This is great for libraries and frameworks (React is MIT, Vue is MIT), but it's dangerous for server-side applications that are sold as a service.\n\nConsider what happened with MongoDB and Elasticsearch. Both were originally Apache 2.0. Cloud providers (AWS, Azure) packaged them as managed services and competed with the companies that built them — without sharing revenue or code improvements. Both projects eventually moved to source-available licenses (SSPL, Elastic License).\n\nWe wanted to avoid this dynamic from day one. PinConsole is a complete application, not a library. If we used MIT, a cloud provider could offer \"PinConsole as a Service\" tomorrow, undercutting our ability to offer commercial licenses and fund continued development.\n\nAGPL-3.0 closes this loophole: if you modify PinConsole and offer it as a network service, you must distribute your modifications. This doesn't prevent self-hosted use — it prevents SaaS commoditization.",
      },
      {
        heading: 'Why not Apache 2.0?',
        body: "Apache 2.0 is similar to MIT in permissiveness, with an added patent grant clause. For PinConsole, the patent clause is unnecessary (we have no patent portfolio to protect or grant). And like MIT, Apache 2.0 doesn't address the SaaS loophole — a cloud provider could offer PinConsole as a service without contributing code changes back.\n\nWe considered Apache 2.0 briefly but concluded that AGPL-3.0's network-use provision is essential for our business model: the software is free for anyone to self-host, and commercial licenses cover proprietary embedding or SaaS distribution.",
      },
      {
        heading: 'What AGPL-3.0 means for you',
        body: "If you're a self-hosted user, AGPL-3.0 changes nothing about your day-to-day experience:\n\n• You can deploy PinConsole on your own infrastructure, for free, forever.\n• You can modify the code for internal use without sharing your changes.\n• You can audit every line of code for security and compliance.\n• You can fork the project and maintain your own fork.\n\nThe only restriction: if you modify PinConsole and offer it as a SaaS service to third parties, you must release your modifications under AGPL-3.0. If you just want to use it internally for your own website analytics, you're completely free.\n\nFor teams that need to embed PinConsole in a proprietary product, we offer standard commercial licenses. Contact the maintainer for details.",
      },
      {
        heading: 'What about contributors?',
        body: "We require all contributors to sign a Contributor License Agreement (CLA) that grants PinConsole a perpetual, royalty-free license to use their contributions under both AGPL-3.0 and commercial terms. This is standard practice for AGPL projects (MySQL/MariaDB, SugarCRM, etc.) and ensures that:\n\n1. The project can relicense contributions for commercial licensees\n2. Contributors retain full ownership of their code\n3. The project remains sustainable long-term\n\nIf you're contributing a bug fix or small improvement, the CLA is a one-time process. For larger feature contributions, we'll work with you to ensure the licensing is clear.",
      },
      {
        heading: 'AGPL vs SSPL vs BSL — why we didn\'t go further',
        body: "We evaluated stronger source-available licenses like SSPL (MongoDB) and BSL (Business Source License, used by MariaDB and CockroachDB). These go beyond AGPL by explicitly restricting cloud providers from offering the software as a service.\n\nWe chose AGPL-3.0 because:\n\n• It's a well-known, OSI-approved open-source license — no controversy about \"open source\" labeling\n• The network-use clause is sufficient for our needs — we're not worried about a company distributing modified copies of PinConsole, we're worried about SaaS commoditization\n• It's compatible with the broader open-source ecosystem — AGPL code can be combined with Apache 2.0 and MIT code\n• It's familiar to enterprise legal teams — unlike SSPL, AGPL has established precedent\n\nIf we encounter specific SaaS abuse in the future, we can always add a Commons Clause or transition to a different license, but AGPL-3.0 is the right starting point.",
      },
      {
        heading: 'The bottom line',
        body: "AGPL-3.0 aligns our incentives with our users'. Self-hosted users get a free, open, auditable platform. Commercial users get a licensing path for proprietary use. And the project is protected from the \"open core, but AWS hosts it\" dynamic that has hurt so many open-source companies.\n\nWe believe this is the most sustainable model for a project that positions itself as an \"open-source alternative\" to proprietary SaaS — and we hope you agree.",
      },
    ],
  },
  cta: {
    title: 'Self-host PinConsole — it\'s AGPL-3.0 free',
    subtitle: 'No registration required. No session limits. Your data, your servers.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk about commercial licensing', href: '#consult' },
  },
};
