<script setup lang="ts">
const { t } = useI18n()
const localePath = useLocalePath()

const fallbackPacks = [
  { sid: 1, preview_pid: 1, count: 80 },
  { sid: 2, preview_pid: 18, count: 80 },
  { sid: 3, preview_pid: 35, count: 80 },
  { sid: 4, preview_pid: 52, count: 80 },
  { sid: 5, preview_pid: 69, count: 80 },
  { sid: 6, preview_pid: 6, count: 80 },
  { sid: 7, preview_pid: 12, count: 18 }
]

const { data: packs } = await useAsyncData('sticker-packs', () => fetchPacks())
const list = computed(() => (packs.value?.length ? packs.value : fallbackPacks))
</script>

<template>
  <section class="flex flex-col gap-6">
    <article
      v-for="pack in list"
      :key="pack.sid"
      class="border-default-200 relative flex h-20 items-center overflow-hidden border px-6"
    >
      <img
        :src="`/stickers/KUNgal${pack.sid}/${pack.preview_pid}.webp`"
        alt=""
        width="64"
        height="64"
        loading="lazy"
        class="absolute inset-y-0 left-0 h-20 w-20 object-cover"
      >
      <KunLink
        :to="localePath(`/sticker/${pack.sid}`)"
        class="absolute inset-0 flex items-center pl-24 text-base"
      >
        {{ t('home.sticker') }} [{{ pack.sid }}]
      </KunLink>
      <KunButton
        is-icon-only
        variant="light"
        class="relative z-10 ml-auto"
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases"
        target="_blank"
        :aria-label="`Download sticker pack ${pack.sid}`"
      >
        <KunIcon name="lucide:download" class="text-xl" />
      </KunButton>
    </article>
  </section>
</template>
