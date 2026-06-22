import { defineConfig } from 'astro/config';
import cloudflare from '@astrojs/cloudflare';
import vue from '@astrojs/vue';

export default defineConfig({
  output: 'static',
  site: 'https://pinconsole.example.com',
  adapter: cloudflare({
    platformProxy: { enabled: true },
  }),
  integrations: [vue()],
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
