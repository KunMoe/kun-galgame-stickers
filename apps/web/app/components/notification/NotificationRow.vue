<script setup lang="ts">
import type { NotificationItem } from '~/features/notification/api'
import { resolveMultilingual } from '~/features/pack/types'

const props = defineProps<{ item: NotificationItem }>()
const emit = defineEmits<{
  open: [item: NotificationItem]
  markRead: [item: NotificationItem]
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()
const NuxtLink = resolveComponent('NuxtLink')

const name = computed(() => props.item.actor?.name ?? t('notification.someone'))
const pack = computed(() =>
  props.item.pack
    ? resolveMultilingual(props.item.pack.title, locale.value) || t('pack.untitled')
    : t('notification.packUnavailable')
)
const posted = computed(() => props.item.kind === NOTIFICATION_KIND.posted)

// A folded row names one person and counts the rest; the count is of people,
// and the number of new comments rides on its own chip.
const sentence = computed(() => {
  const params = { name: name.value, pack: pack.value, count: props.item.actor_count }
  const many = props.item.actor_count > 1
  switch (props.item.kind) {
    case NOTIFICATION_KIND.replied:
      return t('notification.replied', params)
    case NOTIFICATION_KIND.mentioned:
      return t('notification.mentioned', params)
    case NOTIFICATION_KIND.posted:
    case NOTIFICATION_KIND.threadCreated:
      return t(many ? 'notification.postedMany' : 'notification.posted', params)
    case NOTIFICATION_KIND.liked:
      return t(many ? 'notification.likedMany' : 'notification.liked', params)
    default:
      return t('notification.other', params)
  }
})

// A row whose pack is gone has nowhere to go, so it is not a link; the only
// thing left to do with it is acknowledge it.
const link = computed(() =>
  props.item.pack ? { to: localePath(`/pack/${props.item.pack.id}`) } : {}
)

const formatTime = (value: string) => new Date(value).toLocaleString()
</script>

<template>
  <component
    :is="item.pack ? NuxtLink : 'article'"
    v-bind="link"
    :class="
      cn(
        'border-default-200 flex gap-3 border p-3',
        item.pack && 'hover:border-default-400 transition-colors'
      )
    "
    @click="item.pack && emit('open', item)"
  >
    <img
      v-if="item.pack?.cover_thumb_url"
      :src="item.pack.cover_thumb_url"
      alt=""
      width="48"
      height="48"
      loading="lazy"
      class="border-default-200 size-12 shrink-0 border object-cover"
    >
    <span
      v-else
      class="bg-default-100 text-default-400 flex size-12 shrink-0 items-center justify-center"
    >
      <KunIcon name="lucide:image-off" class="text-sm" />
    </span>

    <div class="min-w-0 flex-1">
      <p :class="cn('text-sm break-words', item.read ? 'text-default-500' : 'text-foreground')">
        {{ sentence }}
      </p>
      <div class="text-default-500 mt-1 flex flex-wrap items-center gap-2 text-xs">
        <KunChip v-if="posted" size="sm" color="primary" variant="flat">
          {{ t('notification.newComments', { count: item.item_count }) }}
        </KunChip>
        <time :datetime="item.updated_at">{{ formatTime(item.updated_at) }}</time>
      </div>
    </div>

    <KunButton
      v-if="!item.pack && !item.read"
      size="sm"
      variant="light"
      class-name="shrink-0 self-center"
      @click="emit('markRead', item)"
    >
      {{ t('notification.markRead') }}
    </KunButton>
    <span v-if="!item.read" class="bg-primary mt-2 size-2 shrink-0 rounded-full">
      <span class="sr-only">{{ t('notification.unread') }}</span>
    </span>
  </component>
</template>
