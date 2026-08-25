import { page } from '$app/state'
import zhCN from './messages/zh-cn'
import enUS from './messages/en-us'
import type { Messages } from './messages/shape'
import { DEFAULT_LOCALE, type Locale, type MultilingualText, URL_PREFIX_TO_LOCALE } from './types'

export type { Messages }

const dictionaries: Record<Locale, Messages> = {
  'zh-cn': zhCN,
  'en-us': enUS,
  'ja-jp': zhCN
}

export const getMessages = (locale: Locale): Messages =>
  dictionaries[locale] ?? dictionaries[DEFAULT_LOCALE]

/** Reactive accessor that reads the current locale from `$app/state`. Use inside .svelte files. */
export const m = (): Messages => getMessages((page.data?.lang as Locale) ?? DEFAULT_LOCALE)

/**
 * Strip a leading locale prefix (e.g. `/en/about` → `{ locale: 'en-us', pathname: '/about' }`).
 * The default locale has no prefix, so `/about` → `{ locale: 'zh-cn', pathname: '/about' }`.
 */
export const detectLocaleFromPath = (pathname: string): { locale: Locale; pathname: string } => {
  const match = pathname.match(/^\/([a-z]{2})(\/.*)?$/i)
  if (match) {
    const locale = URL_PREFIX_TO_LOCALE[match[1].toLowerCase()]
    if (locale) {
      return { locale, pathname: match[2] || '/' }
    }
  }
  return { locale: DEFAULT_LOCALE, pathname }
}

/** Prepend the locale prefix to a path for navigation links. */
export const localizedPath = (locale: Locale, pathname: string): string => {
  const prefix = locale === DEFAULT_LOCALE ? '' : `/${locale.slice(0, 2)}`
  const path = pathname.startsWith('/') ? pathname : `/${pathname}`
  if (path === '/') return prefix || '/'
  return `${prefix}${path}`
}

/** Resolve a multilingual JSON value (from the DB) by preferred locale order. */
export const resolveMultilingual = (
  value: MultilingualText | null | undefined,
  locale: Locale
): string => {
  if (!value) return ''
  const order: Record<Locale, (keyof MultilingualText)[]> = {
    'zh-cn': ['zh-cn', 'zh-tw', 'ja-jp', 'en-us'],
    'en-us': ['en-us', 'ja-jp', 'zh-tw', 'zh-cn'],
    'ja-jp': ['ja-jp', 'en-us', 'zh-tw', 'zh-cn']
  }
  for (const key of order[locale]) {
    const v = value[key]
    if (v) return v
  }
  return ''
}

export { DEFAULT_LOCALE, LOCALES, LOCALE_NATIVE_NAMES } from './types'
export type { Locale, Theme, MultilingualText } from './types'
