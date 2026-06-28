import type { BlogContent } from './types';

export const businessModelEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Sustainable Open Source: How AGPL-3.0 and Commercial Licensing Work for PinConsole',
    description: 'A transparent look at PinConsole\'s open source business model — AGPL-3.0, commercial licensing as a conversation starter, zero licensing infrastructure, and why we chose pragmatism over process.',
    ogTitle: 'Sustainable Open Source: AGPL-3.0 and Commercial Licensing at PinConsole',
    ogDescription: 'AGPL-3.0, commercial licensing as conversation, zero licensing infrastructure — the pragmatic open source business model behind PinConsole.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '8 min read',
    tags: ['Open Source', 'Business', 'AGPL', 'Licensing'],
  },
  hero: {
    h1: 'Sustainable Open Source: How AGPL-3.0 and Commercial Licensing Work for PinConsole',
    subtitle: 'No licensing servers, no sales pipeline, no contributor agreements. A transparent look at the business model behind a self-hosted open source project.',
  },
  content: {
    sections: [
      {
        heading: 'The Open Source Sustainability Problem',
        body: `Sustaining an open source project is hard. The stats are well known: 94% of open source maintainers are unpaid. Projects with millions of users run on the spare time of a handful of people. Burnout is endemic.\n\nThe standard sustainability playbook has a few well-worn paths:\n\n• Donations (Open Collective, GitHub Sponsors) — works for a niche, rarely covers a salary\n• Managed hosting (WordPress.com, GitLab) — you run the SaaS version of your OSS project\n• Dual licensing (MySQL, MongoDB) — AGPL for open source, commercial license for proprietary use\n• Consulting and support (Red Hat model) — sell expertise around the open source product\n\nPinConsole follows the dual licensing path, but with a twist: we deliberately keep the commercial licensing infrastructure to a minimum. No licensing server, no sales team, no automated key generation. Just an email conversation.\n\nThis is why.`,
      },
      {
        heading: 'What AGPL-3.0 Actually Does (and Doesn\'t Do)',
        body: `The AGPL-3.0 license is the foundation of the model. A quick recap of how it works:\n\nIf you use PinConsole as-is for your own internal operations — monitoring your own website, assisting your own visitors — the AGPL imposes no obligations beyond preserving the license notice. You can deploy it, modify it for internal use, and never share a single line of code.\n\nThe AGPL only activates if you distribute modified versions to others, or if you offer the software as a network service to third parties. The famous "network use is distribution" clause (Section 13) closes the SaaS loophole that the standard GPL leaves open.\n\nThis is why we chose AGPL over MIT or Apache 2.0. A cloud provider cannot take PinConsole, wrap it in a SaaS offering, and compete with the open source project without sharing their modifications. The license prevents commoditization.\n\nBut critically, the AGPL does not create a revenue stream. It only creates a boundary. The revenue opportunity comes from the commercial license — a separate agreement for organizations that cannot or will not comply with the AGPL's requirements.`,
      },
      {
        heading: 'The Commercial License: What It Actually Is',
        body: `The commercial license for PinConsole is remarkably simple. It's not a product with tiers, pricing pages, or feature gating. It's a conversation.\n\nIf your organization needs to:\n\n• Embed PinConsole in a proprietary product that you distribute to customers\n• Use PinConsole in a way that the AGPL's copyleft requirements create compliance friction\n• Have a signed commercial agreement for procurement or legal requirements\n\nYou email the maintainer. We discuss your use case. We agree on terms that work for both parties. You get a license that explicitly permits your use case. The project gets funding.\n\nThere are no feature tiers. The commercial license covers the same software as the AGPL version — same code, same capabilities, same single binary. The difference is the legal terms, not the software.\n\nThis simplicity is intentional. Building a licensing server, implementing key validation, maintaining a customer portal — that's infrastructure that consumes time and attention away from the product itself. For a project at PinConsole's stage, the right model is "make the software great first, figure out enterprise sales infrastructure when it's paying for itself."`,
      },
      {
        heading: 'Why No Contributor License Agreement (CLA)',
        body: `Many open source projects with commercial licensing require a Contributor License Agreement (CLA). This gives the project maintainer the right to relicense contributions under the commercial license.\n\nPinConsole does not have a CLA. There is no contributor agreement, no copyright assignment, no additional paperwork to submit a pull request.\n\nThis is a deliberate choice. CLAs create friction for contributors — especially first-time contributors. They require legal review, signing infrastructure, and a mental model of "I need permission to contribute." For a project that wants to build a community, that friction is a cost.\n\nThe trade-off is real: without a CLA, we cannot relicense community contributions under the commercial license without the contributor's individual permission. If someone contributes a significant feature, we need to go back to them if a commercial licensee needs that feature under the commercial terms.\n\nIn practice, this hasn't been an issue. Contributions that are directly useful to commercial licensees are typically infrastructure-level improvements (better performance, security fixes, deployment enhancements) — the kind of changes that contributors are happy to dual-license when asked. And because the commercial license conversation is a personal email, not an automated system, asking for permission is natural.\n\nWe may add a CLA in the future if the volume of commercial licenses and community contributions grows to the point where individual permission requests become a bottleneck. But we won't add one before it's needed.`,
      },
      {
        heading: 'The Consulting Path: Commercial License as Relationship Entry Point',
        body: `For most open source projects, commercial licensing alone doesn't generate enough revenue to sustain full-time development. The real opportunity is the consulting and implementation services that follow the licensing conversation.\n\nWhen an organization inquires about a commercial license, the conversation naturally leads to:\n\n• "How do we deploy this in production?"\n• "Can you help us customize the co-browsing behavior?"\n• "We need integration with our SSO provider — is that on the roadmap?"\n• "Can you review our deployment architecture for high availability?"\n\nThese conversations are the actual value. The commercial license is the door — the consulting engagement is the room.\n\nThis model aligns incentives well. The maintainer has a direct financial incentive to make PinConsole better, more deployable, and more valuable to organizations. And those organizations get a customized, production-ready deployment that solves their specific problem — not just a binary they downloaded and configured themselves.\n\nNot every open source project can follow this path. It only works if the maintainer has deep expertise in the problem domain that organizations are willing to pay for. For PinConsole — a self-hosted alternative to tools that cost $500+/month — that expertise has clear economic value.`,
      },
      {
        heading: 'What Self-Hosted Users Get (Everything)',
        body: `A concern with dual-licensed projects is that the open source version becomes a "crippled" trial — limited features, missing capabilities, deliberate friction.\n\nPinConsole does not do this. The AGPL-licensed version is the full product:\n\n• Unlimited sessions — no per-session caps\n• Unlimited operators — no per-seat pricing\n• All features — session replay, live monitoring, co-browsing, chat, popups\n• All storage backends — PostgreSQL, Redis, MinIO\n• All integrations — SDK config, widget customization, admin APIs\n\nThere is no "enterprise edition" with features withheld from the community version. There is no hidden pricing page. The only difference between the AGPL version and the commercial license is the legal terms under which you use the same code.\n\nThis is a philosophical choice as much as a practical one. PinConsole exists because we believe self-hosted session replay should be accessible to every team — not just teams with enterprise budgets. The AGPL protects that vision while allowing organizations that need different legal terms to fund the project's development.`,
      },
      {
        heading: 'Comparison With Other Licensing Models',
        body: `The open source licensing landscape has evolved significantly in recent years. Here's how PinConsole's approach compares:\n\n**MongoDB SSPL (Server Side Public License).** Created after AWS offered MongoDB as a managed service without contributing back. The SSPL goes further than AGPL — it requires you to license the entire "management layer" (monitoring, backup, automation) under SSPL if you offer MongoDB as a service. This is more protective than AGPL but has not been approved by OSI and creates compatibility friction with other open source projects. PinConsole sticks with OSI-approved AGPL-3.0.\n\n**Elastic License (ELv2).** Elastic moved from Apache 2.0 to a source-available license that restricts use to "managed services" and "SaaS products." Not OSI-approved, not open source by the strict definition. We prefer AGPL — it IS open source, OSI-approved, and achieves the same practical outcome.\n\n**BSL (Business Source License).** MariaDB's model — open source code that converts from a restricted license to GPL after a change date (typically 3-4 years). Attractive for projects that want a time-limited exclusive window for commercial licensing. Simpler than CLA + dual license in some ways, but adds the complexity of license-version tracking.\n\n**MIT with SaaS offering.** Some projects use MIT for the open source code and monetize a managed SaaS version (GitLab, WordPress). This works when the project has network effects or switching costs that prevent competitors from offering the same SaaS at lower prices. For session replay, switching costs are low — any cloud provider could offer MIT-licensed PinConsole as a $5/month service. AGPL prevents this.\n\nFor PinConsole, AGPL-3.0 + commercial license is the right balance: OSI-approved, legally standard, and effective at preventing SaaS commoditization without the complexity of custom licenses.`,
      },
      {
        heading: 'Transparency: Revenue and Sustainability',
        body: `I believe in being transparent about the business side of open source.\n\nAs of this writing, PinConsole is in its early stage. The commercial licensing infrastructure is minimal — a handful of conversations with organizations evaluating the project. The project is sustained by:\n\n• Direct development work (building features that organizations need)\n• Consulting engagements (deployment, customization, training)\n• Commercial license fees (for organizations that need different legal terms)\n\nSustainability doesn't require millions in VC funding. It requires covering the costs of development, infrastructure, and the maintainer's time. For a self-hosted tool with no cloud infrastructure costs, the bar is low.\n\nThe goal is not to maximize revenue. The goal is to build a session replay platform that teams can actually use, under terms that respect their autonomy over their data — and to sustain that development over the long term. AGPL-3.0 with a pragmatic commercial licensing path achieves that.\n\nIf you're evaluating PinConsole for your organization and have questions about the license, email me. The conversation is the product.`,
      },
    ],
  },
  relatedPosts: [
    { title: 'AGPL-3.0 vs MIT: Why We Chose AGPL for Our Open Source Project', url: '/en/blog/agpl-vs-mit/', description: 'A deep dive into the licensing decision behind PinConsole.' },
    { title: 'Privacy by Design in Open-Source Session Replay', url: '/en/blog/privacy-by-design/', description: 'How PinConsole balances open-source with privacy-first design.' },
  ],
  cta: {
    title: 'Try PinConsole — Free Under AGPL-3.0',
    subtitle: 'Full features, unlimited sessions, no crippled enterprise edition. Self-hosted and open source.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk about commercial licensing', href: '#consult' },
  },
};
