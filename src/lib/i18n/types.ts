export type Locale = 'zh-cn' | 'en-us' | 'ja-jp'
export type Theme = 'light' | 'dark' | 'system'

export const LOCALES: readonly Locale[] = ['zh-cn', 'en-us'] as const
export const DEFAULT_LOCALE: Locale = 'zh-cn'

export const URL_PREFIX_TO_LOCALE: Record<string, Locale> = {
  en: 'en-us',
  ja: 'ja-jp'
}

export const LOCALE_TO_URL_PREFIX: Record<Locale, string> = {
  'zh-cn': '',
  'en-us': 'en',
  'ja-jp': 'ja'
}

export const LOCALE_NATIVE_NAMES: Record<Locale, string> = {
  'zh-cn': '中文',
  'en-us': 'English',
  'ja-jp': '日本語'
}

export interface MultilingualText {
  'zh-cn'?: string
  'zh-tw'?: string
  'en-us'?: string
  'ja-jp'?: string
}
