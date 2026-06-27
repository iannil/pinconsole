import { defineConfig } from 'astro/config';
import cloudflare from '@astrojs/cloudflare';
import vue from '@astrojs/vue';
import sitemap from '@astrojs/sitemap';

export default defineConfig({
  output: 'static',
  site: 'https://pinconsole.com',
  adapter: cloudflare({
    // dev 模式下禁用 miniflare，避免 macOS Sequoia 阻止写 ~/Library/Preferences/.wrangler/registry
    platformProxy: { enabled: process.env.ASTRO_USE_PLATFORM === '1' },
  }),
  integrations: [vue(), sitemap({
    lastmod: new Date(),
    serialize(item) {
      const url = item.url;
      // Homepage
      if (url === 'https://pinconsole.com/') {
        item.priority = 1.0;
        item.changefreq = 'weekly';
      }
      // English homepage
      else if (url === 'https://pinconsole.com/en/') {
        item.priority = 0.9;
        item.changefreq = 'weekly';
      }
      // SEO landing pages (alternatives, category pages)
      else if (url.includes('/alternatives/') || url.includes('/co-browsing/') || url.includes('/session-replay/')) {
        item.priority = 0.8;
        item.changefreq = 'weekly';
      }
      // Blog posts
      else if (url.includes('/blog/')) {
        item.priority = 0.7;
        item.changefreq = 'monthly';
      }
      // All other pages
      else {
        item.priority = 0.7;
        item.changefreq = 'monthly';
      }
      return item;
    },
  })],
  i18n: {
    defaultLocale: 'zh',
    locales: ['zh', 'en'],
    routing: {
      prefixDefaultLocale: false,
    },
  },
  vite: {
    resolve: {
      alias: {
        '@': '/src',
      },
    },
  },
});
