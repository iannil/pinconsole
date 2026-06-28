import rss from '@astrojs/rss';
import type { APIRoute } from 'astro';
import { agplVsMitEn } from '../../../content/blog/agpl-vs-mit-en';
import { fullstoryAlternativeEn } from '../../../content/blog/fullstory-alternative-en';
import { cobrowsingEn } from '../../../content/blog/cobrowsing-en';
import { privacyEn } from '../../../content/blog/privacy-en';
import { securityEn } from '../../../content/blog/security-en';
import { websocketEn } from '../../../content/blog/websocket-hub-en';
import { businessModelEn } from '../../../content/blog/business-model-en';

const posts = [
  { content: businessModelEn, slug: 'open-source-business-model' },
  { content: websocketEn, slug: 'websocket-hub-500-concurrent' },
  { content: securityEn, slug: 'defense-in-depth' },
  { content: privacyEn, slug: 'privacy-by-design' },
  { content: cobrowsingEn, slug: 'building-co-browsing' },
  { content: fullstoryAlternativeEn, slug: 'building-self-hosted-session-replay' },
  { content: agplVsMitEn, slug: 'agpl-vs-mit' },
];

export const GET: APIRoute = async (context) => {
  const site = context.site ?? 'https://pinconsole.com';
  return rss({
    title: 'PinConsole Blog',
    description: 'Technical deep-dives, architecture decisions, and product updates from the PinConsole team.',
    site,
    trailingSlash: false,
    items: posts.map(({ content, slug }) => ({
      title: content.meta.title,
      pubDate: new Date(content.blog.publishedDate),
      description: content.meta.description,
      link: `/en/blog/${slug}/`,
    })),
    customData: '<language>en-us</language>',
  });
};
