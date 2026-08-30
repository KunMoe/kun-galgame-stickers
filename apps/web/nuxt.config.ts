import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'

const rootEnv = fileURLToPath(new URL('../../.env', import.meta.url))
if (existsSync(rootEnv)) {
  for (const line of readFileSync(rootEnv, 'utf8').split('\n')) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const eq = trimmed.indexOf('=')
    if (eq < 1) continue
    const key = trimmed.slice(0, eq)
    if (process.env[key] === undefined) process.env[key] = trimmed.slice(eq + 1)
  }
}

export default defineNuxtConfig({
  extends: ['@kungal/ui-nuxt'],

  compatibilityDate: '2026-08-08',
  devtools: { enabled: false },

  modules: ['@nuxt/eslint', '@nuxtjs/color-mode', '@nuxtjs/i18n', '@nuxtjs/sitemap'],

  css: ['~/assets/css/main.css'],

  vite: {
    plugins: [tailwindcss()]
  },

  imports: {
    dirs: ['~/features/**'],
    presets: [{ from: '@kungal/ui-core', imports: ['cn'] }]
  },

  devServer: {
    host: '127.0.0.1',
    port: 5173
  },

  runtimeConfig: {
    apiBaseUrl: process.env.NUXT_API_BASE_URL || 'http://127.0.0.1:9421',
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://127.0.0.1:9421',
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'https://sticker.kungal.com',
      oauthServerUrl:
        process.env.NUXT_PUBLIC_OAUTH_SERVER_URL || 'http://127.0.0.1:9277/api/v1',
      oauthFrontendUrl:
        process.env.NUXT_PUBLIC_OAUTH_FRONTEND_URL || 'http://127.0.0.1:9420',
      oauthClientId: process.env.NUXT_PUBLIC_OAUTH_CLIENT_ID || '',
      oauthRedirectUri:
        process.env.NUXT_PUBLIC_OAUTH_REDIRECT_URI || 'http://127.0.0.1:5173/auth/callback'
    }
  },

  i18n: {
    strategy: 'prefix_except_default',
    defaultLocale: 'zh-cn',
    langDir: 'locales',
    baseUrl: process.env.NUXT_PUBLIC_SITE_URL || 'https://sticker.kungal.com',
    locales: [
      { code: 'zh-cn', language: 'zh-CN', name: '中文', file: 'zh-cn.json' },
      { code: 'en', language: 'en-US', name: 'English', file: 'en.json' },
      { code: 'ja', language: 'ja-JP', name: '日本語', file: 'ja.json' }
    ],
    detectBrowserLanguage: false
  },

  colorMode: {
    preference: 'system',
    fallback: 'light',
    classPrefix: 'kun-',
    classSuffix: '-mode',
    storageKey: 'kun-sticker-theme'
  },

  site: {
    url: process.env.NUXT_PUBLIC_SITE_URL || 'https://sticker.kungal.com'
  },

  sitemap: {
    exclude: ['/auth/**'],
    sources: ['/api/__sitemap__/urls'],
    excludeAppSources: true,
    cacheMaxAgeSeconds: 60 * 60 * 6,
    defaults: { changefreq: 'weekly', priority: 0.7 }
  },

  app: {
    head: {
      link: [{ rel: 'icon', type: 'image/webp', href: '/favicon.webp' }]
    }
  },

  eslint: {
    config: { stylistic: false }
  }
})
