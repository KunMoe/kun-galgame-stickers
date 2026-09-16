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
