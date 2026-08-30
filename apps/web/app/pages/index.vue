<script setup lang="ts">
const { t, locale } = useI18n()
const localePath = useLocalePath()

const fallbackPacks: StickerPack[] = [
  { sid: 1, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 1, preview_url: '/stickers/KUNgal1/1.webp', count: 80 },
  { sid: 2, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 18, preview_url: '/stickers/KUNgal2/18.webp', count: 80 },
  { sid: 3, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 35, preview_url: '/stickers/KUNgal3/35.webp', count: 80 },
  { sid: 4, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 52, preview_url: '/stickers/KUNgal4/52.webp', count: 80 },
  { sid: 5, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 69, preview_url: '/stickers/KUNgal5/69.webp', count: 80 },
  { sid: 6, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 6, preview_url: '/stickers/KUNgal6/6.webp', count: 80 },
  { sid: 7, owner_uid: 2, status: 1, title: {}, description: {}, preview_pid: 12, preview_url: '/stickers/KUNgal7/12.webp', count: 18 }
]

const { data: packs } = await useAsyncData('sticker-packs', () => fetchPacks())
const list = computed(() => (packs.value?.length ? packs.value : fallbackPacks))

const packTitle = (pack: StickerPack) =>
  resolveMultilingual(pack.title, locale.value) || `${t('home.sticker')} [${pack.sid}]`
</script>

<template>
  <section class="flex flex-col gap-6">
    <article
      v-for="pack in list"
      :key="pack.sid"
      class="border-default-200 relative flex h-20 items-center overflow-hidden border px-6"
    >
      <img
        :src="pack.preview_url"
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
        {{ packTitle(pack) }}
      </KunLink>
      <KunButton
        v-if="pack.sid <= 7"
        is-icon-only
        variant="light"
        class="relative z-10 ml-auto"
        :href="STICKER_GITHUB_RELEASES"
        target="_blank"
        :aria-label="`Download sticker pack ${pack.sid}`"
      >
        <KunIcon name="lucide:download" class="text-xl" />
      </KunButton>
    </article>
  </section>
</template>
