import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// In docker compose the API is reachable as http://api:8080; locally it is
// on localhost. API_PROXY_TARGET overrides the default.
const apiTarget = process.env.API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
      '/health': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
})
