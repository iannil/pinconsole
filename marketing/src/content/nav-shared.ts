/**
 * 统一的导航配置 — 单一定义源。
 *
 * 所有页面（首页 / SEO 内容页 / 博客页）共用同一套导航。
 * 使用路由前缀（如 `/#data-sovereignty`）而非纯锚点，
 * 确保从 SEO 页点击也能正确跳转到首页对应区块。
 */
import type { PageContent } from './types';

// ── 中文导航 ──
export const zhNavLinks = [
  { label: '首页', href: '/' },
  { label: '数据主权', href: '/#data-sovereignty' },
  { label: '能力', href: '/#features' },
  { label: '路线图', href: '/#roadmap' },
  { label: 'FAQ', href: '/#faq' },
  { label: '博客', href: '/blog/' },
] as const;

export const zhNavCta = '预约咨询';
export const zhLocaleSwitch = { label: 'EN', href: '/en' } as const;

// ── 英文导航 ──
export const enNavLinks = [
  { label: 'Home', href: '/en' },
  { label: 'Data Sovereignty', href: '/en/#data-sovereignty' },
  { label: 'Features', href: '/en/#features' },
  { label: 'Roadmap', href: '/en/#roadmap' },
  { label: 'FAQ', href: '/en/#faq' },
  { label: 'Blog', href: '/en/blog/' },
] as const;

export const enNavCta = 'Request consultation';
export const enLocaleSwitch = { label: '中', href: '/' } as const;

// ── 页面布局用 Header/Footer content shim ──

/** 构造仅包含 Header/Footer 所需字段的最小 PageContent 对象。 */
function makeHeaderContent(
  locale: 'zh' | 'en',
  links: readonly { readonly label: string; readonly href: string }[],
  cta: string,
  localeSwitch: { readonly label: string; readonly href: string },
): PageContent {
  return {
    locale,
    htmlLang: locale === 'zh' ? 'zh-CN' : 'en-US',
    meta: { title: '', description: '', ogTitle: '', ogDescription: '' },
    nav: { links: [...links], cta, localeSwitch: { ...localeSwitch } },
    hero: { eyebrow: '', h1: '', h2: '', cta: { primary: { label: '', href: '' } } },
    features: { eyebrow: '', title: '', subtitle: '', items: [] },
    dataSovereignty: { eyebrow: '', title: '', subtitle: '', pillars: [], architectureAlt: '' },
    selfHost: { eyebrow: '', title: '', subtitle: '', code: '', docsLink: { label: '', href: '' } },
    roadmap: { eyebrow: '', title: '', subtitle: '', columns: [] },
    faq: { eyebrow: '', title: '', subtitle: '', items: [] },
    finalCTA: {
      eyebrow: '', title: '', subtitle: '',
      form: {
        nameLabel: '', namePlaceholder: '', companyLabel: '', companyPlaceholder: '',
        contactLabel: '', contactPlaceholder: '', purposeLabel: '',
        purposes: [], messageLabel: '', messagePlaceholder: '',
        submitLabel: '', privacyNote: '', successMessage: '', errorMessage: '',
      },
    },
    footer: { tagline: '', columns: [], license: '', sourceNote: '' },
  };
}

export const zhContentHeader = makeHeaderContent('zh', zhNavLinks, zhNavCta, zhLocaleSwitch);
export const enContentHeader = makeHeaderContent('en', enNavLinks, enNavCta, enLocaleSwitch);
