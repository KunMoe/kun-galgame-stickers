<script setup lang="ts">
const { t, locale } = useI18n()
const colorMode = useColorMode()
const config = useRuntimeConfig()
const route = useRoute()

useHead({
  htmlAttrs: {
    lang: () => locale.value,
    class: () => (colorMode.value === 'dark' ? 'kun-dark-mode' : '')
  },
  title: () => t('meta.title'),
  meta: [
    { name: 'description', content: () => t('meta.description') },
    { property: 'og:title', content: () => t('meta.og') },
    { property: 'og:description', content: () => t('meta.ogDescription') },
    { property: 'og:image', content: `${config.public.siteUrl}/title.webp` }
  ],
  link: [{ rel: 'canonical', href: () => `${config.public.siteUrl}${route.fullPath}` }]
})
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <KunMessageProvider />
  <KunAlertProvider />
</template>
