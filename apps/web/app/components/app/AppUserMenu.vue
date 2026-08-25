<script setup lang="ts">
const { t } = useI18n()
const route = useRoute()
const user = useAuthUser()
const showLogout = ref(false)
const pending = ref<'local' | 'everywhere' | null>(null)

const returnTo = computed(() => route.fullPath)

const displayName = computed(() => user.value?.name || user.value?.email || '')

const kunUser = computed(() => {
  if (!user.value) return null
  return {
    id: user.value.id || 0,
    name: displayName.value || 'KUN',
    avatar: user.value.picture || ''
  }
})

const login = () => startOAuthLogin(returnTo.value)
const register = () => startOAuthRegister(returnTo.value)

const logoutLocalOnly = async () => {
  if (pending.value) return
  pending.value = 'local'
  await logoutLocal()
  user.value = null
  showLogout.value = false
  pending.value = null
}

const logoutEverywhere = async () => {
  if (pending.value) return
  pending.value = 'everywhere'
  await logoutLocal()
  user.value = null
  startOAuthLogout()
}
</script>

<template>
  <div v-if="!user" class="flex items-center gap-1">
    <KunButton variant="light" class="hidden sm:inline-flex" @click="register">
      {{ t('auth.register') }}
    </KunButton>
    <KunButton color="primary" variant="light" @click="login">
      {{ t('auth.login') }}
    </KunButton>
  </div>
  <KunPopover v-else>
    <template #trigger>
      <KunButton variant="light" class="gap-2">
        <KunAvatar v-if="kunUser" :user="kunUser" :is-navigation="false" size="sm" />
        <span class="hidden max-w-32 truncate sm:inline">{{ displayName }}</span>
      </KunButton>
    </template>
    <div class="flex w-48 flex-col gap-1 p-2">
      <KunButton
        variant="light"
        href="https://oauth.kungal.com/profile"
        target="_blank"
        class="justify-start"
      >
        {{ t('auth.profile') }}
      </KunButton>
      <KunButton variant="light" class="justify-start" @click="showLogout = true">
        {{ t('auth.logout') }}
      </KunButton>
    </div>
  </KunPopover>

  <KunModal v-model="showLogout" :title="t('auth.logoutTitle')">
    <p class="text-default-500 mb-4 text-sm">{{ t('auth.logoutPrompt') }}</p>
    <div class="flex flex-col gap-3">
      <KunButton
        color="primary"
        variant="bordered"
        class="h-auto flex-col items-start whitespace-normal p-3"
        :disabled="!!pending"
        @click="logoutEverywhere"
      >
        <span>{{ t('auth.logoutEverywhere') }}</span>
        <span class="text-default-500 text-xs font-normal">{{ t('auth.logoutEverywhereHint') }}</span>
      </KunButton>
      <KunButton
        variant="bordered"
        class="h-auto flex-col items-start whitespace-normal p-3"
        :disabled="!!pending"
        @click="logoutLocalOnly"
      >
        <span>{{ t('auth.logoutLocal') }}</span>
        <span class="text-default-500 text-xs font-normal">{{ t('auth.logoutLocalHint') }}</span>
      </KunButton>
      <div class="flex justify-end">
        <KunButton variant="light" @click="showLogout = false">{{ t('auth.cancel') }}</KunButton>
      </div>
    </div>
  </KunModal>
</template>
