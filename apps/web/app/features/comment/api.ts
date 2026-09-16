import type { Author } from '~/features/pack/types'

export interface Comment {
  id: number
  post_number: number
  content_html: string
  content_raw: string
  created_at: string
  edited_at?: string
  author: Author
  can_edit: boolean
  can_delete: boolean
  like_count: number
  is_liked: boolean
  /** What this answers, and the top-level comment the exchange hangs under. */
  reply_to?: number
  root_id?: number
  reply_to_name?: string
}

/** A top-level comment with the replies that hang under it. */
export interface CommentNode extends Comment {
  replies: Comment[]
}

export interface LikeResult {
  liked: boolean
  like_count: number
}

/**
 * community's closed vocabulary. The numbers are the wire values, so they
 * cannot be reordered to suit a menu.
 */
export const FLAG_REASONS = [0, 1, 2, 4, 3] as const
export type FlagReason = (typeof FLAG_REASONS)[number]

/**
 * community's notification levels. Commenting subscribes you at `watching`,
 * and community never downgrades a level you set yourself -- muting a thread
 * and then replying to it leaves you muted. `watchingFirstPost` exists only
 * on a followed pack; this site never sets it but may read it back from
 * another client.
 */
export const NOTIFY = {
  muted: 0,
  normal: 1,
  tracking: 2,
  watching: 3,
  watchingFirstPost: 4
} as const
export type NotifyLevel = (typeof NOTIFY)[keyof typeof NOTIFY]

/** The reader's own row on a thread. Absent until they touch it. */
export interface CommentViewerState {
  last_read_post_number: number
  unread_count: number
  notification_level: NotifyLevel
}

export interface CommentPage {
  /** 0 until the first comment creates the thread. */
  thread_id: number
  comments: Comment[]
  total: number
  next_cursor?: string
  /** What a read receipt reports having reached; counts tombstones. */
  highest_post_number: number
  viewer?: CommentViewerState
  /** False when the community service is not configured for this deployment. */
  enabled: boolean
}

export const fetchComments = (packId: string, after?: string): Promise<CommentPage | null> =>
  kunFetchOrNull<CommentPage>(
    `/packs/${packId}/comments${after ? `?after=${encodeURIComponent(after)}` : ''}`
  )

export const addComment = (packId: string, body: string, replyTo?: number): Promise<Comment> =>
  kunFetch<Comment>(`/packs/${packId}/comments`, {
    method: 'POST',
    body: { body, reply_to: replyTo ?? 0 }
  })

export const editComment = (commentId: number, body: string): Promise<Comment> =>
  kunFetch<Comment>(`/comments/${commentId}`, { method: 'PATCH', body: { body } })

export const deleteComment = (commentId: number): Promise<unknown> =>
  kunFetch(`/comments/${commentId}`, { method: 'DELETE' })

export const toggleCommentLike = (commentId: number): Promise<LikeResult> =>
  kunFetch<LikeResult>(`/comments/${commentId}/like`, { method: 'POST' })

export const reportComment = (commentId: number, reason: FlagReason, note: string): Promise<unknown> =>
  kunFetch(`/comments/${commentId}/report`, { method: 'POST', body: { reason, note } })

/**
 * Tells community how far this reader got. Reading is never inferred from a
 * GET upstream -- a read face with a write side effect cannot be cached or
 * prefetched safely -- so the page says so explicitly.
 */
export const markCommentsRead = (threadId: number, postNumber: number): Promise<CommentViewerState> =>
  kunFetch<CommentViewerState>(`/comments/threads/${threadId}/read`, {
    method: 'POST',
    body: { post_number: postNumber }
  })

export const setCommentNotification = (
  threadId: number,
  level: NotifyLevel
): Promise<CommentViewerState> =>
  kunFetch<CommentViewerState>(`/comments/threads/${threadId}/notification`, {
    method: 'POST',
    body: { level }
  })

/**
 * Follows a pack whose comment wall has no thread yet. Once a thread exists,
 * the thread-level call is the one that counts.
 */
export const setPackCommentNotification = (
  packId: string,
  level: typeof NOTIFY.normal | typeof NOTIFY.watching
): Promise<CommentViewerState> =>
  kunFetch<CommentViewerState>(`/packs/${packId}/comments/notification`, {
    method: 'POST',
    body: { level }
  })

/**
 * Lays incoming comments over the ones already shown, keyed by id and kept in
 * post order. The wall is oldest first and the reader's own new comment is
 * shown before the pages between it and them are loaded, so a later page can
 * both overlap what is here and belong above part of it.
 */
export const mergeComments = (shown: Comment[], incoming: Comment[]): Comment[] => {
  const byId = new Map(shown.map((comment) => [comment.id, comment]))
  for (const comment of incoming) byId.set(comment.id, comment)
  return [...byId.values()].sort((a, b) => a.post_number - b.post_number)
}

/**
 * Groups a flat page into one level of nesting. community threads arbitrarily
 * deep -- a reply to a reply keeps the root of the exchange -- but a comment
 * section under a sticker pack reads better as "comment, then answers to it"
 * than as an indent ladder, so everything under a root sits at one depth and
 * says who it answered.
 */
export const nestComments = (comments: Comment[]): CommentNode[] => {
  const roots: CommentNode[] = []
  const byId = new Map<number, CommentNode>()
  for (const comment of comments) {
    if (!comment.root_id) {
      const node = { ...comment, replies: [] }
      byId.set(comment.id, node)
      roots.push(node)
    }
  }
  for (const comment of comments) {
    if (!comment.root_id) continue
    const parent = byId.get(comment.root_id)
    // A reply whose root is on an earlier page has nothing to nest under, so
    // it stands on its own rather than disappearing.
    if (parent) parent.replies.push(comment)
    else roots.push({ ...comment, replies: [] })
  }
  return roots
}

export const MAX_COMMENT_LENGTH = 2000
