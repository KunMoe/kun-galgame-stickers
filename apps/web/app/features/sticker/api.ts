import { kunFetch } from '~/utils/kunFetch'
import type { Sticker, StickerPack, StickerPackDetail } from './types'

export const fetchPacks = (): Promise<StickerPack[] | null> => kunFetch<StickerPack[]>('/sticker/packs')

export const fetchPack = (sid: number): Promise<StickerPackDetail | null> =>
  kunFetch<StickerPackDetail>(`/sticker/packs/${sid}`)

export const fetchSticker = (sid: number, pid: number): Promise<Sticker | null> =>
  kunFetch<Sticker>(`/sticker/packs/${sid}/${pid}`)
