<script setup lang="ts">
const { t, locales } = useI18n()
const localePath = useLocalePath()
const user = useAuthUser()
const colorMode = useColorMode()
const switchLocalePath = useSwitchLocalePath()

const themeItems = computed(() => [
  { key: 'light', label: t('header.light') },
  { key: 'dark', label: t('header.dark') },
  { key: 'system', label: t('header.system') }
])

const languageItems = computed(() =>
  (locales.value as { code: string; name: string }[]).map((item) => ({
    key: item.code,
    label: item.name
  }))
)

const onTheme = (item: { key: string }) => {
  colorMode.preference = item.key
}

const onLanguage = async (item: { key: string }) => {
  const code = item.key as 'zh-cn' | 'en' | 'ja'
  await navigateTo(switchLocalePath(code))
}
</script>

<template>
  <header
    class="bg-content1/80 fixed inset-x-0 top-0 z-[1007] flex h-14 items-center gap-4 border-b px-4 backdrop-blur-md sm:px-12"
  >
    <KunLink :to="localePath('/')" class="flex items-center gap-3">
      <img src="/favicon.webp" alt="" class="h-10 w-10" >
      <span class="hidden text-lg sm:block">{{ t('header.title') }}</span>
    </KunLink>

    <nav class="flex flex-1 items-center justify-center gap-6 text-base">
      <KunLink :to="localePath('/')" class="text-primary">{{ t('header.home') }}</KunLink>
      <KunLink :to="localePath('/about')" class="text-primary">{{ t('header.about') }}</KunLink>
      <KunLink
        v-if="user"
        :to="localePath('/me/packs')"
        class="text-primary"
      >
        {{ t('header.myPacks') }}
      </KunLink>
    </nav>

    <div class="flex items-center gap-1">
      <KunDropdown :items="themeItems" @select="onTheme">
        <template #trigger>
          <span class="inline-flex h-9 w-9 items-center justify-center" aria-label="Theme">
            <KunIcon name="lucide:sun-moon" class="text-xl" />
          </span>
        </template>
      </KunDropdown>

      <KunDropdown :items="languageItems" @select="onLanguage">
        <template #trigger>
          <span class="inline-flex h-9 w-9 items-center justify-center" aria-label="Language">
            <KunIcon name="lucide:languages" class="text-xl" />
          </span>
        </template>
      </KunDropdown>

      <KunButton
        is-icon-only
        variant="light"
        :href="STICKER_GITHUB_REPO"
        target="_blank"
        aria-label="GitHub"
      >
        <KunIcon name="lucide:github" class="text-xl" />
      </KunButton>

      <AppUserMenu />
    </div>
  </header>
</template>
