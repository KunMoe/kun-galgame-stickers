<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const { t, locale } = useI18n()
const localePath = useLocalePath()

const { data: packs, refresh } = await useAsyncData('my-sticker-packs', () => fetchMyPacks())

const statusLabel = (status: number) => {
  if (status === PACK_PUBLISHED) return t('publish.published')
  if (status === PACK_HIDDEN) return t('publish.hidden')
  return t('publish.draft')
}

const toggle = async (pack: StickerPack) => {
  if (pack.status === PACK_PUBLISHED) await unpublishPack(pack.sid)
  else await publishPack(pack.sid)
  await refresh()
}
</script>

<template>
  <section class="flex flex-col gap-6">
    <div class="flex items-center justify-between gap-4">
      <KunHeader :name="t('publish.title')" scale="h2" />
      <KunButton color="primary" :href="localePath('/me/packs/new')">
        {{ t('publish.new') }}
      </KunButton>
    </div>

    <p v-if="!packs?.length" class="text-default-500">
      {{ t('publish.empty') }}
    </p>

    <article
      v-for="pack in packs ?? []"
      :key="pack.sid"
      class="border-default-200 flex items-center gap-4 border p-3"
    >
      <img
        :src="pack.preview_url"
        alt=""
        width="64"
        height="64"
        class="h-16 w-16 object-cover"
      >
      <div class="min-w-0 flex-1">
        <KunLink :to="localePath(`/me/packs/${pack.sid}`)" class="truncate text-base">
          {{ resolveMultilingual(pack.title, locale) || t('publish.new') }}
        </KunLink>
        <p class="text-default-500 text-sm">
          {{ statusLabel(pack.status) }} · {{ pack.count }}
        </p>
      </div>
      <KunButton
        size="sm"
        variant="bordered"
        :disabled="pack.status !== PACK_PUBLISHED && pack.count < 1"
        @click="toggle(pack)"
      >
        {{ pack.status === PACK_PUBLISHED ? t('publish.unpublish') : t('publish.publish') }}
      </KunButton>
    </article>
  </section>
</template>
