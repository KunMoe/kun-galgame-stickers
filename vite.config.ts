import { sveltekit } from '@sveltejs/kit/vite'
import { enhancedImages } from '@sveltejs/enhanced-img'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [tailwindcss(), enhancedImages(), sveltekit()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true
  },
  test: {
    include: ['src/**/*.{test,spec}.{js,ts}']
  }
})
