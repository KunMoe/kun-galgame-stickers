import type { PageLoad } from './$types'

const PACK_PREVIEWS: { sid: number; previewPid: number }[] = [
  { sid: 1, previewPid: 1 },
  { sid: 2, previewPid: 18 },
  { sid: 3, previewPid: 35 },
  { sid: 4, previewPid: 52 },
  { sid: 5, previewPid: 69 },
  { sid: 6, previewPid: 6 },
  { sid: 7, previewPid: 12 }
]

export const load: PageLoad = () => ({ packs: PACK_PREVIEWS })
