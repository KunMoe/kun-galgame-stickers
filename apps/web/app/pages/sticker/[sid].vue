<script setup lang="ts">
const { t, locale } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const sid = computed(() => Number(route.params.sid))

const { data: pack } = await useAsyncData(
  () => `sticker-pack-${sid.value}`,
  () => fetchPack(sid.value)
)

if (!pack.value) {
  throw createError({ statusCode: 404, statusMessage: t('sticker.notFound') })
}

useSeoMeta({
  title: `${t('sticker.title')} [${sid.value}]`,
  description: t('sticker.description')
})

const downloadImage = async (path: string) => {
  const response = await fetch(path)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = path.split('/').pop() ?? 'sticker.png'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="grid grid-cols-2 gap-5 sm:grid-cols-3 md:grid-cols-4">
    <KunCard
      v-for="sticker in pack?.stickers ?? []"
      :key="sticker.pid"
      color="default"
      class-name="relative flex flex-col gap-2 p-3"
    >
      <div class="aspect-square w-full overflow-hidden">
        <img
          :src="sticker.thumb_url"
          alt=""
          width="320"
          height="320"
          loading="lazy"
          class="h-full w-full object-cover"
        >
      </div>
      <div class="text-default-600 text-sm">
        <p class="truncate">
          {{ t('sticker.game') }}: {{ resolveMultilingual(sticker.game, locale) }}
        </p>
        <p class="truncate">
          {{ t('sticker.lass') }}: {{ resolveMultilingual(sticker.loli, locale) }}
        </p>
      </div>
      <div class="mt-1 flex items-center justify-between">
        <KunButton
          size="sm"
          variant="bordered"
          :href="localePath(`/sticker/${sid}-${sticker.pid}`)"
        >
          {{ t('sticker.original') }}
        </KunButton>
        <KunButton
          is-icon-only
          variant="light"
          :aria-label="t('sticker.download')"
          @click="downloadImage(stickerOriginalSrc(sticker))"
        >
          <KunIcon name="lucide:download" class="text-xl" />
        </KunButton>
      </div>
    </KunCard>
  </div>
</template>
