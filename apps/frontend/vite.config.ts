import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const envDir = path.resolve(__dirname, '../..')
  const env = loadEnv(mode, envDir, '')
  const gatewayKey = env.AIHUB_GATEWAY_API_KEY ?? ''
  const adminKey = env.AIHUB_ADMIN_API_KEY ?? ''

  return {
    envDir,
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      strictPort: true,
      proxy: {
        '/healthz': { target: 'http://127.0.0.1:8080', changeOrigin: true },
        '/readyz': { target: 'http://127.0.0.1:8080', changeOrigin: true },
        '/diagnostics': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
          headers: gatewayKey ? { Authorization: `Bearer ${gatewayKey}` } : {},
        },
        '/v1': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
          headers: gatewayKey ? { Authorization: `Bearer ${gatewayKey}` } : {},
          configure: (proxy) => {
            proxy.on('proxyReq', (proxyReq) => {
              proxyReq.removeHeader('origin')
            })
          },
        },
        '/api': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
          headers: adminKey ? { Authorization: `Bearer ${adminKey}` } : {},
          configure: (proxy) => {
            proxy.on('proxyReq', (proxyReq) => {
              proxyReq.removeHeader('origin')
            })
          },
        },
      },
    },
  }
})
