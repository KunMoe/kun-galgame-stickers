import { sveltekit } from '@sveltejs/kit/vite'
import { enhancedImages } from '@sveltejs/enhanced-img'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [tailwindcss(), enhancedImages(), sveltekit()],
  test: {
    include: ['src/**/*.{test,spec}.{js,ts}']
  }
})
