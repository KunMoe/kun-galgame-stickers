import { kunFetch } from '~/utils/kunFetch'
import type {
  ImageUpload,
  MultilingualText,
  Sticker,
  StickerPack,
  StickerPackDetail
} from './types'

export const fetchPacks = (): Promise<StickerPack[] | null> =>
  kunFetch<StickerPack[]>('/sticker/packs')

export const fetchMyPacks = (): Promise<StickerPack[] | null> =>
  kunFetch<StickerPack[]>('/sticker/me/packs')

export const fetchPack = (sid: number): Promise<StickerPackDetail | null> =>
  kunFetch<StickerPackDetail>(`/sticker/packs/${sid}`)

export const fetchSticker = (sid: number, pid: number): Promise<Sticker | null> =>
  kunFetch<Sticker>(`/sticker/packs/${sid}/${pid}`)

export const createPack = (title: MultilingualText, description?: MultilingualText) =>
  kunFetch<StickerPack>('/sticker/packs', {
    method: 'POST',
    body: { title, description }
  })

export const patchPack = (
  sid: number,
  body: { title?: MultilingualText; description?: MultilingualText; preview_pid?: number }
) => kunFetch<StickerPack>(`/sticker/packs/${sid}`, { method: 'PATCH', body })

export const publishPack = (sid: number) =>
  kunFetch<StickerPack>(`/sticker/packs/${sid}/publish`, { method: 'POST' })

export const unpublishPack = (sid: number) =>
  kunFetch<StickerPack>(`/sticker/packs/${sid}/unpublish`, { method: 'POST' })

export const uploadPackImage = (sid: number, file: File) => {
  const body = new FormData()
  body.append('file', file, file.name)
  return kunFetch<ImageUpload>(`/sticker/packs/${sid}/images`, { method: 'POST', body })
}

export const addSticker = (
  sid: number,
  body: {
    image_hash: string
    game?: MultilingualText
    loli?: MultilingualText
    vndb?: number
    describe?: string
  }
) => kunFetch<Sticker>(`/sticker/packs/${sid}/stickers`, { method: 'POST', body })

export const patchSticker = (
  sid: number,
  pid: number,
  body: {
    image_hash?: string
    game?: MultilingualText
    loli?: MultilingualText
    vndb?: number
    describe?: string
  }
) => kunFetch<Sticker>(`/sticker/packs/${sid}/stickers/${pid}`, { method: 'PATCH', body })

export const deleteSticker = (sid: number, pid: number) =>
  kunFetch(`/sticker/packs/${sid}/stickers/${pid}`, { method: 'DELETE' })
