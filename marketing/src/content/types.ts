/**
 * Content schema for pinconsole marketing site.
 * Both zh and en must satisfy this contract — type-driven i18n.
 */

export type LeadPurpose =
  | 'evaluate'
  | 'self-host'
  | 'custom'
  | 'compliance'
  | 'other';

export interface NavLink {
  label: string;
  href: string;
}

export interface CTA {
  primary: { label: string; href: string };
  secondary?: { label: string; href: string };
  tertiary?: { label: string; href: string };
}

export interface Feature {
  icon: string;
  title: string;
  description: string;
  bullets: string[];
  screenshot?: string;
}

export interface RoadmapColumn {
  status: 'shipped' | 'coming' | 'out-of-scope';
  title: string;
  items: string[];
}

export interface FAQItem {
  question: string;
  answer: string;
}

export interface PageContent {
  locale: 'zh' | 'en';
  htmlLang: string;
  meta: {
    title: string;
    description: string;
    ogTitle: string;
    ogDescription: string;
  };
  nav: {
    links: NavLink[];
    cta: string;
    localeSwitch: { label: string; href: string };
  };
  hero: {
    eyebrow: string;
    h1: string;
    h2: string;
    cta: CTA;
  };
  features: {
    eyebrow: string;
    title: string;
    subtitle: string;
    items: Feature[];
  };
  dataSovereignty: {
    eyebrow: string;
    title: string;
    subtitle: string;
    pillars: { icon: string; title: string; description: string }[];
    architectureAlt: string;
  };
  selfHost: {
    eyebrow: string;
    title: string;
    subtitle: string;
    code: string;
    docsLink: { label: string; href: string };
  };
  roadmap: {
    eyebrow: string;
    title: string;
    subtitle: string;
    columns: RoadmapColumn[];
  };
  faq: {
    eyebrow: string;
    title: string;
    subtitle: string;
    items: FAQItem[];
  };
  finalCTA: {
    eyebrow: string;
    title: string;
    subtitle: string;
    form: {
      nameLabel: string;
      namePlaceholder: string;
      companyLabel: string;
      companyPlaceholder: string;
      contactLabel: string;
      contactPlaceholder: string;
      purposeLabel: string;
      purposes: { value: LeadPurpose; label: string }[];
      messageLabel: string;
      messagePlaceholder: string;
      submitLabel: string;
      privacyNote: string;
      successMessage: string;
      errorMessage: string;
    };
  };
  footer: {
    tagline: string;
    columns: { title: string; links: NavLink[] }[];
    license: string;
    sourceNote: string;
  };
}
