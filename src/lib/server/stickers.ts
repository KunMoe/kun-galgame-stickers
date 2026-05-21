import { prisma } from './prisma'
import type { MultilingualText } from '$lib/i18n/types'

export interface Sticker {
  sid: number
  pid: number
  game: MultilingualText
  loli: MultilingualText
  vndb: number
  describe: string
}

const STICKER_SELECT = {
  sid: true,
  pid: true,
  game: true,
  loli: true,
  vndb: true,
  describe: true
} as const

export const findStickerPack = async (sid: number): Promise<Sticker[]> => {
  const rows = await prisma.sticker.findMany({
    where: { sid },
    orderBy: { pid: 'asc' },
    select: STICKER_SELECT
  })
  return rows.map(toSticker)
}

export const findOneSticker = async (sid: number, pid: number): Promise<Sticker | null> => {
  const row = await prisma.sticker.findUnique({
    where: { sid_pid: { sid, pid } },
    select: STICKER_SELECT
  })
  return row ? toSticker(row) : null
}

const toSticker = (row: {
  sid: number
  pid: number
  game: unknown
  loli: unknown
  vndb: number
  describe: string
}): Sticker => ({
  sid: row.sid,
  pid: row.pid,
  game: (row.game ?? {}) as MultilingualText,
  loli: (row.loli ?? {}) as MultilingualText,
  vndb: row.vndb,
  describe: row.describe
})
