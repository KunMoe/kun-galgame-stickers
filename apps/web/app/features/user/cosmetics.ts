import type { KunAvatarDecoration, KunUser } from '@kungal/ui-core'
import type { Author } from '~/features/pack/types'

export interface Decoration {
  item_id: number
  name: string
  static_url: string
  animated_url?: string
}

export interface Cosmetics {
  avatar_frame?: Decoration
  profile_background?: Decoration
}

export const avatarDecoration = (cosmetics?: Cosmetics): KunAvatarDecoration | null => {
  const frame = cosmetics?.avatar_frame
  if (!frame) return null
  return {
    src: frame.static_url,
    animatedSrc: frame.animated_url
  }
}

export const toKunUser = (author: Author): KunUser => ({
  id: author.id,
  name: author.name,
  avatar: author.avatar,
  avatarDecoration: avatarDecoration(author.cosmetics)
})
