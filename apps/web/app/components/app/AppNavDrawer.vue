<script setup lang="ts">
const open = defineModel<boolean>({ default: false })

const { t, locales, locale } = useI18n()
const localePath = useLocalePath()
const switchLocalePath = useSwitchLocalePath()
const user = useAuthUser()
const colorMode = useColorMode()
const route = useRoute()

const close = () => {
  open.value = false
}

watch(() => route.fullPath, close)

const navItems = computed(() => {
  const items = [
    { to: localePath('/'), label: t('header.discover'), icon: 'lucide:compass' },
    { to: localePath('/comments'), label: t('header.comments'), icon: 'lucide:messages-square' },
    { to: localePath('/about'), label: t('header.about'), icon: 'lucide:info' }
  ]
  if (user.value) {
    items.push({
      to: localePath('/me/packs'),
      label: t('header.myPacks'),
      icon: 'lucide:folder-heart'
    })
  }
  items.push({
    to: localePath('/settings'),
    label: t('header.settings'),
    icon: 'lucide:settings'
  })
  return items
})

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

const onLanguage = async (code: string) => {
  await navigateTo(switchLocalePath(code as 'zh-cn' | 'en' | 'ja'))
}

const register = () => {
  close()
  startOAuthRegister(route.fullPath)
}
</script>

<template>
  <KunDrawer
    v-model="open"
    placement="left"
    size="sm"
    :responsive="false"
    :title="t('header.title')"
  >
    <nav class="flex flex-col gap-1">
      <KunButton
        v-for="item in navItems"
        :key="item.to"
        :href="item.to"
        variant="light"
        class-name="w-full justify-start gap-2"
        @click="close"
      >
        <KunIcon :name="item.icon" class="text-lg" />
        {{ item.label }}
      </KunButton>
      <KunButton
        v-if="!user"
        variant="light"
        class-name="w-full justify-start gap-2"
        @click="register"
      >
        <KunIcon name="lucide:user-plus" class="text-lg" />
        {{ t('auth.register') }}
      </KunButton>
    </nav>

    <div class="mt-6 flex flex-col gap-2">
      <p class="text-default-500 text-xs">{{ t('header.theme') }}</p>
      <div class="flex flex-wrap gap-1">
        <KunButton
          v-for="item in themeItems"
          :key="item.key"
          size="sm"
          :variant="colorMode.preference === item.key ? 'solid' : 'light'"
          @click="colorMode.preference = item.key"
        >
          {{ item.label }}
        </KunButton>
      </div>
    </div>

    <div class="mt-4 flex flex-col gap-2">
      <p class="text-default-500 text-xs">{{ t('header.language') }}</p>
      <div class="flex flex-wrap gap-1">
        <KunButton
          v-for="item in languageItems"
          :key="item.key"
          size="sm"
          :variant="item.key === locale ? 'solid' : 'light'"
          @click="onLanguage(item.key)"
        >
          {{ item.label }}
        </KunButton>
      </div>
    </div>
  </KunDrawer>
</template>
