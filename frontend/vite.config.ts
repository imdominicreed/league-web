import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { readFileSync } from 'fs'

const pkg = JSON.parse(readFileSync('./package.json', 'utf-8'))

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  // const apiUrl = env.VITE_API_URL || 'http://localhost:9999'
  const apiUrl = 'http://localhost:9999'

  return {
    plugins: [react()],
    define: {
      __APP_VERSION__: JSON.stringify(pkg.version),
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      port: 3000,
      allowedHosts: ['.dev.local'],
      proxy: {
        '/api': {
          target: apiUrl,
          changeOrigin: true,
          ws: true,
        },
      },
    },
  }
})
