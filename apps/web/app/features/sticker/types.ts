export type MultilingualText = Record<string, string | undefined>

export interface Sticker {
  sid: number
  pid: number
  game: MultilingualText
  loli: MultilingualText
  vndb: number
  describe: string
  image_hash?: string
  image_url: string
  thumb_url: string
}

export interface StickerPack {
  sid: number
  owner_uid: number
  status: number
  title: MultilingualText
  description: MultilingualText
  preview_pid: number
  preview_url: string
  count: number
  published_at?: string
}

export interface StickerPackDetail extends StickerPack {
  stickers: Sticker[]
}

export interface ImageUpload {
  hash: string
  url: string
  variant_urls?: Record<string, string>
  width: number
  height: number
}

export const PACK_DRAFT = 0
export const PACK_PUBLISHED = 1
export const PACK_HIDDEN = 2

const localeOrder: Record<string, string[]> = {
  'zh-cn': ['zh-cn', 'zh-tw', 'ja-jp', 'en-us'],
  en: ['en-us', 'en', 'ja-jp', 'zh-tw', 'zh-cn'],
  ja: ['ja-jp', 'ja', 'en-us', 'zh-tw', 'zh-cn']
}

export const resolveMultilingual = (
  value: MultilingualText | null | undefined,
  locale: string
): string => {
  if (!value) return ''
  const order = localeOrder[locale] ?? ['zh-cn', 'zh-tw', 'ja-jp', 'en-us']
  for (const key of order) {
    const v = value[key]
    if (v) return v
  }
  return Object.values(value).find(Boolean) ?? ''
}

export const localeField = (locale: string): string => {
  if (locale === 'en') return 'en-us'
  if (locale === 'ja') return 'ja-jp'
  return 'zh-cn'
}

export const stickerOriginalSrc = (sticker: Sticker): string => {
  if (sticker.image_hash) return sticker.image_url
  return `/kun-galgame-stickers/telegram/KUNgal${sticker.sid}/${sticker.pid}.png`
}
