import { prisma } from '~/prisma/index'

export interface StickerData {
  sid: number
  pid: number
  game: KunLanguage
  loli: KunLanguage
  vndb: number
  describe: string
}

export const findStickersData = async (sid: number): Promise<StickerData[]> => {
  const data = await prisma.sticker.findMany({
    where: { sid },
    orderBy: { pid: 'asc' },
    select: {
      sid: true,
      pid: true,
      game: true,
      loli: true,
      vndb: true,
      describe: true
    }
  })

  return data.map((sticker) => ({
    ...sticker,
    game: sticker.game as unknown as KunLanguage,
    loli: sticker.loli as unknown as KunLanguage
  }))
}

export const findOneStickerData = async (sid: number, pid: number): Promise<StickerData | null> => {
  const data = await prisma.sticker.findUnique({
    where: {
      sid_pid: {
        sid,
        pid
      }
    },
    select: {
      sid: true,
      pid: true,
      game: true,
      loli: true,
      vndb: true,
      describe: true
    }
  })

  if (!data) {
    return null
  }

  return {
    ...data,
    game: data.game as unknown as KunLanguage,
    loli: data.loli as unknown as KunLanguage
  }
}
