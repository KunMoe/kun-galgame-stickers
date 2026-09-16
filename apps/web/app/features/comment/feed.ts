import type { Author, MultilingualText } from '~/features/pack/types'

/** The pack a comment was left on, as much of it as a list row needs. */
export interface CommentPackRef {
  id: string
  title: MultilingualText
  cover_thumb_url?: string
}

/** A comment seen from outside its pack: readable, clickable, not editable. */
export interface CommentFeedItem {
  id: number
  post_number: number
  content_html: string
  created_at: string
  author: Author
  pack: CommentPackRef
}

export interface CommentFeed {
  items: CommentFeedItem[]
  next_cursor?: string
  enabled: boolean
}

export interface UnreadPack {
  thread_id: number
  pack: CommentPackRef
  unread_count: number
  posts_count: number
  last_posted_at?: string
}

export interface UnreadComments {
  packs: UnreadPack[]
  /** The red dot. Counts only what this site has a page for. */
  total: number
  enabled: boolean
}

/** A real answer that happens to be empty: nobody has commented yet. */
const EMPTY_FEED: CommentFeed = { items: [], enabled: true }

/**
 * A request that did not come back. This is deliberately not EMPTY_FEED: a
 * comment service that is down must not read as a site where nobody has
 * commented. `enabled: false` is the same cue a deployment without community
 * configured gives, and the page says "unavailable" for both.
 */
const UNAVAILABLE_FEED: CommentFeed = { items: [], enabled: false }

export const fetchLatestComments = async (cursor?: string): Promise<CommentFeed> => {
  const path = cursor ? `/comments/latest?cursor=${encodeURIComponent(cursor)}` : '/comments/latest'
  return (await kunFetchOrNull<CommentFeed>(path)) ?? UNAVAILABLE_FEED
}

/**
 * Searches the markdown a commenter typed, not the HTML it was cooked into.
 * Upstream this is a trigram scan over Postgres rather than a search engine, so
 * it is a real round trip -- callers debounce it rather than firing per key.
 */
export const searchComments = async (q: string, cursor?: string): Promise<CommentFeed> => {
  const trimmed = q.trim()
  if (trimmed.length < 2) return EMPTY_FEED
  const page = cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''
  const path = `/comments/search?q=${encodeURIComponent(trimmed)}${page}`
  return (await kunFetchOrNull<CommentFeed>(path)) ?? UNAVAILABLE_FEED
}

export const fetchUnreadComments = async (): Promise<UnreadComments> =>
  (await kunFetchOrNull<UnreadComments>('/me/comments/unread')) ?? {
    packs: [],
    total: 0,
    enabled: false
  }

/**
 * One shared unread count for the whole app, so the header badge and the
 * comments page agree and only one of them pays for the request.
 */
export const useUnreadComments = () => {
  const user = useAuthUser()
  const state = useState<UnreadComments>('comment-unread', () => ({
    packs: [],
    total: 0,
    enabled: true
  }))
  const loaded = useState('comment-unread-loaded', () => false)

  const refresh = async () => {
    loaded.value = true
    if (!user.value) {
      // Signed out is a real answer, not a failure: there is nothing unread
      // because there is nobody to have read it.
      state.value = { packs: [], total: 0, enabled: true }
      return
    }
    state.value = await fetchUnreadComments()
  }

  /** For callers that want the badge filled once, not refreshed on every
   *  mount -- the header asks on every page, the list asks for fresh data. */
  const ensure = async () => {
    if (!loaded.value) await refresh()
  }

  /** Drops a pack from the badge the moment its wall is opened, rather than
   *  waiting for the next refresh to tell us what we already know. */
  const clear = (threadId: number) => {
    const kept = state.value.packs.filter((row) => row.thread_id !== threadId)
    if (kept.length !== state.value.packs.length) {
      state.value = { ...state.value, packs: kept, total: kept.length }
    }
  }

  return { state, refresh, ensure, clear }
}
