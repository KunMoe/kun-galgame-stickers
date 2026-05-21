import { error } from '@sveltejs/kit'
import { findStickerPack } from '$lib/server/stickers'
import type { PageServerLoad } from './$types'

export const load: PageServerLoad = async ({ params }) => {
  const sid = Number(params.sid)
  if (!Number.isInteger(sid) || sid <= 0) error(404, 'Invalid sticker pack id')

  const stickers = await findStickerPack(sid)
  if (stickers.length === 0) error(404, 'Sticker pack not found')

  return { sid, stickers }
}
