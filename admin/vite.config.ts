/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'node:path';

// 切片 1a：admin Vue3 SPA 的 Vite 配置。
// - dev: HMR + proxy /api 与 /healthz 到后端 8080
// - build: 输出 dist/ 供 Go embed
// - test: vitest + coverage(TS-1 切片配置,thresholds 60% 起步,TS-5 提升到 90%)
export default defineConfig({
  plugins: [vue()],
  base: '/admin/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/readyz': 'http://localhost:8080',
      '/sdk.js': 'http://localhost:5174',
      // /ws/* 必须 enable ws 代理,否则 admin 用 location.host 推断 WS endpoint
      // 会去连 ws://localhost:5173/ws/operator(vite dev server),连接失败一直
      // 卡在 CONNECTING 状态,Dashboard 看不到实时事件。
      '/ws': { target: 'http://localhost:8080', ws: true },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        // 与 base 配合，使生成的 index.html 引用 /admin/assets/*.js
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
      },
    },
  },
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: [
        'node_modules/',
        'dist/',
        '**/*.d.ts',
        '**/*.config.ts',
        '**/*.cjs',
        '**/.eslintrc*',
        'tests/**',
        'src/main.ts',
        'src/env.d.ts',
      ],
      thresholds: {
        lines: 60,
        functions: 50,
        branches: 50,
        statements: 60,
      },
    },
  },
});
