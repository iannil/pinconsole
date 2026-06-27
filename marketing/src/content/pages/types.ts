/**
 * Simplified content schema for SEO landing pages (alternatives, category pages).
 * Not all features of the main PageContent — just meta + hero + body + FAQ + CTA.
 */
export interface SeoPageContent {
  locale: 'zh' | 'en';
  htmlLang: string;
  meta: {
    title: string;
    description: string;
    ogTitle: string;
    ogDescription: string;
  };
  hero: {
    h1: string;
    subtitle: string;
  };
  comparison: {
    rows: { label: string; pinconsole: string; competitor: string; isHighlight?: boolean }[];
  };
  content: {
    sections: {
      heading: string;
      body: string;
    }[];
  };
  evidence?: {
    blocks: {
      quote: string;
      attribution: string;
      sourceUrl?: string;
    }[];
  };
  faq: {
    items: { question: string; answer: string }[];
  };
  cta: {
    title: string;
    subtitle: string;
    primary: { label: string; href: string };
    secondary?: { label: string; href: string };
  };
}
