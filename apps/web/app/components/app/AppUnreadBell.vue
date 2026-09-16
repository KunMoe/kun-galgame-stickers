<script setup lang="ts">
const { t } = useI18n()
const localePath = useLocalePath()
const user = useAuthUser()
const { state, refresh, ensure } = useUnreadComments()

onMounted(() => void ensure())
// Signing in or out changes whose unread this is, and signing out has to empty
// the badge rather than leave the last reader's number on screen.
watch(user, () => void refresh())
</script>

<template>
  <NuxtLink
    v-if="user"
    :to="{ path: localePath('/comments'), query: { tab: 'unread' } }"
    :aria-label="t('comment.tabUnread')"
    class="text-default-500 hover:text-foreground hidden h-9 w-9 items-center justify-center transition-colors sm:inline-flex"
  >
    <KunBadge :count="state.total" :max="99" :show="state.total > 0" color="danger">
      <KunIcon name="lucide:bell" class="text-xl" />
    </KunBadge>
  </NuxtLink>
</template>
