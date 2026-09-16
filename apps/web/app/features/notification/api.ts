import type { CommentPackRef } from '~/features/comment/feed'
import type { Author } from '~/features/pack/types'

export interface NotificationItem {
  id: number
  kind: number
  thread_id: number
  pack?: CommentPackRef
  actor?: Author
  actor_count: number
  item_count: number
  post_id?: number
  post_number?: number
  first_post_number?: number
  read: boolean
  created_at: string
  updated_at: string
}

export interface NotificationPage {
  items: NotificationItem[]
  next_cursor?: string
  unread_count: number
  enabled: boolean
}

export interface NotificationCount {
  unread_count: number
  enabled: boolean
}

/**
 * community's wire values. 6 (answer_accepted) and 7 (feedback_status) belong
 * to forum shapes this site never renders specifically.
 *
 * Written across lines on purpose: Nuxt's auto-import scanner reads a one-line
 * object literal as a destructuring export and silently skips the export after
 * it.
 */
export const NOTIFICATION_KIND = {
  replied: 1,
  mentioned: 2,
  posted: 3,
  threadCreated: 4,
  liked: 5
} as const

export const fetchNotifications = (cursor?: string): Promise<NotificationPage | null> =>
  kunFetchOrNull<NotificationPage>(
    cursor ? `/me/notifications?cursor=${encodeURIComponent(cursor)}` : '/me/notifications'
  )

export const fetchUnreadNotificationCount = (): Promise<NotificationCount | null> =>
  kunFetchOrNull<NotificationCount>('/me/notifications/unread-count')

export const markNotificationsRead = (
  target: { ids: number[] } | { all: true }
): Promise<NotificationCount> =>
  kunFetch<NotificationCount>('/me/notifications/read', {
    method: 'POST',
    body: target
  })

/**
 * One shared unread count for the whole app, so the header badge and the
 * notifications list agree and only one of them pays for the request.
 */
export const useNotificationBadge = () => {
  const user = useAuthUser()
  const state = useState<NotificationCount>('notification-unread', () => ({
    unread_count: 0,
    enabled: true
  }))
  const loaded = useState('notification-unread-loaded', () => false)

  const refresh = async () => {
    loaded.value = true
    if (!user.value) {
      // Signed out is a real answer, not a failure: there is nothing unread
      // because there is nobody to have read it.
      state.value = { unread_count: 0, enabled: true }
      return
    }
    const count = await fetchUnreadNotificationCount()
    state.value = count ?? { unread_count: 0, enabled: false }
  }

  /** For callers that want the badge filled once, not refreshed on every
   *  mount -- the header asks on every page, the list asks for fresh data. */
  const ensure = async () => {
    if (!loaded.value) await refresh()
  }

  const set = (unread: number) => {
    state.value = { ...state.value, unread_count: unread }
  }

  return { state, refresh, ensure, set }
}
