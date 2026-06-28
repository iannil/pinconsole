import rss from '@astrojs/rss';
import type { APIRoute } from 'astro';
import { agplVsMitZh } from '../../content/blog/agpl-vs-mit-zh';
import { fullstoryAlternativeZh } from '../../content/blog/fullstory-alternative-zh';
import { cobrowsingZh } from '../../content/blog/cobrowsing-zh';
import { privacyZh } from '../../content/blog/privacy-zh';
import { securityZh } from '../../content/blog/security-zh';
import { websocketHubZh as websocketZh } from '../../content/blog/websocket-hub-zh';
import { businessModelZh } from '../../content/blog/business-model-zh';

const posts = [
  { content: businessModelZh, slug: 'open-source-business-model-zh' },
  { content: websocketZh, slug: 'websocket-hub-500-concurrent-zh' },
  { content: securityZh, slug: 'defense-in-depth-zh' },
  { content: privacyZh, slug: 'privacy-by-design-zh' },
  { content: cobrowsingZh, slug: 'building-co-browsing-zh' },
  { content: fullstoryAlternativeZh, slug: 'self-hosted-fullstory-alternative' },
  { content: agplVsMitZh, slug: 'agpl-vs-mit-zh' },
];

export const GET: APIRoute = async (context) => {
  const site = context.site ?? 'https://pinconsole.com';
  return rss({
    title: 'PinConsole 博客',
    description: 'PinConsole 官方的技术分享、架构解析与产品更新。',
    site,
    trailingSlash: false,
    items: posts.map(({ content, slug }) => ({
      title: content.meta.title,
      pubDate: new Date(content.blog.publishedDate),
      description: content.meta.description,
      link: `/blog/${slug}/`,
    })),
    customData: '<language>zh-cn</language>',
  });
};
