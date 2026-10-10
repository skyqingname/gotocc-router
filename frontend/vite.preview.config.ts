import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'

// Standalone fixture preview. This entry is never included in the production app.
export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '@': resolve(__dirname, 'src') } },
  server: { host: '127.0.0.1', port: 4173, strictPort: true },
  build: { rollupOptions: { input: resolve(__dirname, 'routing-preview.html') } },
})
