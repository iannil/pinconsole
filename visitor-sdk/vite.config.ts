/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import { resolve } from 'node:path';

// 切片 1a：visitor-sdk 的 Vite 配置。
// - dev: playground/ 作为测试页（含 <script src="/src/index.ts">），HMR
// - build: 库模式，输出单文件 dist/sdk.js（IIFE 自动初始化）
// - test: vitest + coverage(TS-1 切片配置,thresholds 15% 起步因 src/index.ts 400 行未测,
//   TS-2 提升至 90%)
//
// 注:lib.name 在 2026-06-20 rename 重构中从 MarketingMonitorSDK 改为 PinconsoleSDK。
export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'PinconsoleSDK',
      fileName: () => 'sdk.js',
      formats: ['iife'],
    },
    outDir: 'dist',
    sourcemap: true,
    // 库模式默认不打包 CSS（rrweb 不需要 CSS）
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        // 单文件，依赖内联（rrweb 也内联到 sdk.js）
        inlineDynamicImports: true,
      },
    },
  },
  server: {
    port: 5174,
    strictPort: true,
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
        'playground/**',
      ],
      thresholds: {
        lines: 15,
        functions: 15,
        branches: 15,
        statements: 15,
      },
    },
  },
});
