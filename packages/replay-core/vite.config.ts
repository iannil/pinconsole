import { defineConfig } from 'vite';
import { resolve } from 'path';

/// <reference types="vitest/config" />

export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'ReplayCore',
      formats: ['es', 'iife'],
      fileName: (format) => format === 'iife'
        ? 'replay-core.iife.js'
        : format === 'es'
          ? 'replay-core.js'
          : 'replay-core.[ext]',
    },
    rollupOptions: {
      // Bundle everything for self-contained deployment
      external: [],
      output: {
        globals: {},
      },
    },
    minify: false,
    sourcemap: true,
  },
  test: {
    environment: 'jsdom',
    include: ['tests/**/*.test.ts'],
    setupFiles: ['./tests/setup/jsdom-polyfills.ts'],
  },
});