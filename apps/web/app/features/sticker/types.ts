export type MultilingualText = Record<string, string | undefined>

export interface Sticker {
  sid: number
  pid: number
  game: MultilingualText
  loli: MultilingualText
  vndb: number
  describe: string
}

export interface StickerPack {
  sid: number
  preview_pid: number
  count: number
}

export interface StickerPackDetail {
  sid: number
  stickers: Sticker[]
}

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
