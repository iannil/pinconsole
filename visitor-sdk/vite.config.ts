import { defineConfig } from 'vite';
import { resolve } from 'node:path';

// 切片 1a：visitor-sdk 的 Vite 配置。
// - dev: playground/ 作为测试页（含 <script src="/src/index.ts">），HMR
// - build: 库模式，输出单文件 dist/sdk.js（IIFE 自动初始化）
export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'MarketingMonitorSDK',
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
});
