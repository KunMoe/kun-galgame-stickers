<script setup lang="ts">
import type { NotificationItem } from '~/features/notification/api'

const { t } = useI18n()
const mutate = useMutation()
const { set: setBadge } = useNotificationBadge()

// The list is the reader's own and the tab only exists once they are signed
// in, so it is fetched on the client rather than carried in the SSR payload.
const items = ref<NotificationItem[]>([])
const cursor = ref('')
const unread = ref(0)
const down = ref(false)
const loading = ref(true)
const loadingMore = ref(false)
const marking = ref(false)

const syncUnread = (count: number) => {
  unread.value = count
  setBadge(count)
}

onMounted(async () => {
  const page = await fetchNotifications()
  loading.value = false
  // null is a request that did not come back, and enabled: false a deployment
  // without community; neither is "you have no notifications".
  if (!page?.enabled) {
    down.value = true
    return
  }
  items.value = page.items
  cursor.value = page.next_cursor ?? ''
  syncUnread(page.unread_count)
})

const loadMore = async () => {
  if (!cursor.value || loadingMore.value) return
  loadingMore.value = true
  const page = await fetchNotifications(cursor.value)
  loadingMore.value = false
  if (!page?.enabled) {
    useKunMessage(t('comment.unavailable'), 'error')
    return
  }
  // The cursor is a seq and a folded row that moves forward takes a new,
  // higher one, so a later page never repeats a row already shown.
  items.value = [...items.value, ...page.items]
  cursor.value = page.next_cursor ?? ''
  syncUnread(page.unread_count)
}

const applyRead = (id: number, count: number) => {
  items.value = items.value.map((item) => (item.id === id ? { ...item, read: true } : item))
  syncUnread(count)
}

const markAll = async () => {
  if (marking.value) return
  marking.value = true
  const result = await mutate(() => markNotificationsRead({ all: true }))
  marking.value = false
  if (!result) return
  items.value = items.value.map((item) => (item.read ? item : { ...item, read: true }))
  syncUnread(result.unread_count)
}

const open = (item: NotificationItem) => {
  if (item.read) return
  markNotificationsRead({ ids: [item.id] })
    .then((result) => applyRead(item.id, result.unread_count))
    // The reader is already on the way to the pack; a toast there about a
    // badge here would only confuse, and the row stays unread to try again.
    .catch(() => {})
}

const markRead = async (item: NotificationItem) => {
  const result = await mutate(() => markNotificationsRead({ ids: [item.id] }))
  if (result) applyRead(item.id, result.unread_count)
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <KunButton
      v-if="unread > 0 && !down"
      size="sm"
      variant="light"
      class-name="self-end"
      :disabled="marking"
      @click="markAll"
    >
      {{ t('notification.markAllRead') }}
    </KunButton>

    <p v-if="loading" class="text-default-500 text-sm">
      {{ t('comment.loading') }}
    </p>
    <p v-else-if="down" class="text-default-500 text-sm">
      {{ t('comment.unavailable') }}
    </p>
    <p v-else-if="!items.length" class="text-default-500 text-sm">
      {{ t('notification.empty') }}
    </p>

    <NotificationRow
      v-for="item in items"
      :key="item.id"
      :item="item"
      @open="open"
      @mark-read="markRead"
    />

    <KunButton
      v-if="cursor"
      variant="bordered"
      class-name="self-center"
      :disabled="loadingMore"
      @click="loadMore"
    >
      {{ loadingMore ? t('comment.loading') : t('comment.loadMore') }}
    </KunButton>
  </div>
</template>
