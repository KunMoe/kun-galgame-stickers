<script setup lang="ts">
const { t, locale } = useI18n()
const route = useRoute()
const sid = Number(route.params.sid)
const pid = Number(route.params.pid)

const { data: sticker } = await useAsyncData(
  `sticker-${sid}-${pid}`,
  () => fetchSticker(sid, pid)
)

if (!sticker.value) {
  throw createError({ statusCode: 404, statusMessage: t('sticker.notFound') })
}

const game = computed(() => resolveMultilingual(sticker.value?.game, locale.value))
const lass = computed(() => resolveMultilingual(sticker.value?.loli, locale.value))

useSeoMeta({
  title: () => `${game.value} | ${lass.value}`,
  description: () => `${game.value} | ${lass.value}`
})
</script>

<template>
  <div v-if="sticker" class="grid place-items-center gap-4">
    <img
      :src="stickerOriginalSrc(sticker)"
      :alt="`${game} ${lass}`"
      class="bg-content1 max-w-full border p-2"
    >
    <section class="text-default-600 flex flex-col items-center gap-1 text-sm">
      <p>{{ t('sticker.game') }}: {{ game }}</p>
      <p>{{ t('sticker.lass') }}: {{ lass }}</p>
    </section>
  </div>
</template>
