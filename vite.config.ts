import { defineConfig } from 'vite';
import solid from 'vite-plugin-solid';

export default defineConfig({
  root: 'frontend',
  plugins: [solid()],
  server: {
    host: '127.0.0.1',
    port: 1430,
    strictPort: true,
    // The Wails window proxies the page, so point HMR at Vite directly.
    hmr: { host: '127.0.0.1', port: 1430 },
  },
  clearScreen: false,
  build: { outDir: 'dist', emptyOutDir: true },
});
