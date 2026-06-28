/**
 * Content schema for blog posts on the marketing site.
 */

export interface BlogSection {
  heading: string;
  body: string;
  code?: string;
  codeLanguage?: string;
}

export interface BlogContent {
  locale: 'en' | 'zh';
  htmlLang: string;
  meta: {
    title: string;
    description: string;
    ogTitle: string;
    ogDescription: string;
  };
  blog: {
    author: string;
    publishedDate: string;
    readingTime: string;
    tags: string[];
  };
  hero: {
    h1: string;
    subtitle: string;
  };
  content: {
    sections: BlogSection[];
  };
  relatedPosts?: { title: string; url: string; description: string }[];
  cta: {
    title: string;
    subtitle: string;
    primary: { label: string; href: string };
    secondary?: { label: string; href: string };
  };
}
