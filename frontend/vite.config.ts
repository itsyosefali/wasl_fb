import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { fileURLToPath, URL } from 'node:url';

const apiTarget = process.env.VITE_API_PROXY_TARGET || 'http://localhost:8080';
const apiProxy = {
  target: apiTarget,
  changeOrigin: true,
  ws: true,
};

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    host: true,
    allowedHosts: ['meta.itsyosefali.cloud', '.traefik.me', 'localhost'],
    proxy: {
      '/pages': apiProxy,
      '/conversations': apiProxy,
      '/messages': apiProxy,
      '/auth': apiProxy,
      '/events': apiProxy,
      '/connect': apiProxy,
      '/health': apiProxy,
      '/webhooks': apiProxy,
      '/comments': apiProxy,
      '/contacts': apiProxy,
      '/actions': apiProxy,
    },
  },
});
