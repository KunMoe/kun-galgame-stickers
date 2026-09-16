<script setup lang="ts">
const props = defineProps<{ packId: string; threadId: number }>()
// null is "no row": a reader who never touched the thread has no level at all.
const level = defineModel<NotifyLevel | null>('level', { required: true })

const { t } = useI18n()
const mutate = useMutation()

const subscribed = computed(() => level.value !== null && level.value >= NOTIFY.tracking)

// Commenting already subscribes you upstream, so on a thread this exists for
// the two deliberate choices: muting a wall you are in, and following one you
// have not spoken in. Before the first comment there is no thread to address,
// so the follow goes on the pack instead, and muting it would mean nothing.
const toggle = async () => {
  const muting = subscribed.value
  const state = await mutate(() =>
    props.threadId
      ? setCommentNotification(props.threadId, muting ? NOTIFY.muted : NOTIFY.watching)
      : setPackCommentNotification(props.packId, muting ? NOTIFY.normal : NOTIFY.watching)
  )
  if (!state) return
  level.value = state.notification_level
  useKunMessage(t(muting ? 'comment.muted' : 'comment.subscribed'), 'success')
}
</script>

<template>
  <KunButton
    size="sm"
    variant="light"
    class-name="gap-1.5"
    :aria-label="t(subscribed ? 'comment.mute' : 'comment.subscribe')"
    @click="toggle"
  >
    <KunIcon :name="subscribed ? 'lucide:bell-off' : 'lucide:bell'" class="text-base" />
    <span class="hidden sm:inline">{{ t(subscribed ? 'comment.mute' : 'comment.subscribe') }}</span>
  </KunButton>
</template>
