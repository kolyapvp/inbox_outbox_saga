import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // Backend API
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''), // Remove /api prefix if backend doesn't expect it.
        // Wait, current router uses /orders directly.
        // If I call /api/orders, rewrite removes /api -> /orders. Correct.
      },
    },
  },
})
