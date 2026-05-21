import { error } from '@sveltejs/kit'
import { findOneSticker } from '$lib/server/stickers'
import type { PageServerLoad } from './$types'

export const load: PageServerLoad = async ({ params }) => {
  const sid = Number(params.sid)
  const pid = Number(params.pid)
  if (!Number.isInteger(sid) || !Number.isInteger(pid) || sid <= 0 || pid <= 0) {
    error(404, 'Invalid sticker id')
  }

  const sticker = await findOneSticker(sid, pid)
  if (!sticker) error(404, 'Sticker not found')

  return { sticker }
}
