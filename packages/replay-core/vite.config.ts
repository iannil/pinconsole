import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'ReplayCore',
      formats: ['es'],
      fileName: () => 'replay-core.js',
    },
    rollupOptions: {
      external: /node_modules/,
      output: {
        globals: {},
      },
    },
    minify: false,
    sourcemap: true,
  },
});
