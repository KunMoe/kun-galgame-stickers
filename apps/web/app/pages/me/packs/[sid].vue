<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const { t, locale } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const sid = computed(() => Number(route.params.sid))
const key = computed(() => localeField(locale.value))
const uploading = ref(false)
const input = ref<HTMLInputElement | null>(null)

const { data: pack, refresh } = await useAsyncData(
  () => `edit-pack-${sid.value}`,
  () => fetchPack(sid.value)
)

if (!pack.value) {
  throw createError({ statusCode: 404, statusMessage: t('publish.notFound') })
}

const title = ref(resolveMultilingual(pack.value.title, locale.value))

const saveTitle = async () => {
  const trimmed = title.value.trim()
  if (!trimmed) return
  await patchPack(sid.value, { title: { ...pack.value?.title, [key.value]: trimmed } })
  await refresh()
}

const onFiles = async (event: Event) => {
  const files = Array.from((event.target as HTMLInputElement).files ?? [])
  if (!files.length || uploading.value) return
  uploading.value = true
  for (const file of files) {
    const uploaded = await uploadPackImage(sid.value, file)
    if (!uploaded?.hash) continue
    await addSticker(sid.value, { image_hash: uploaded.hash })
  }
  uploading.value = false
  if (input.value) input.value.value = ''
  await refresh()
}

const saveSticker = async (sticker: Sticker, field: 'game' | 'loli', value: string) => {
  await patchSticker(sid.value, sticker.pid, {
    [field]: { ...sticker[field], [key.value]: value }
  })
}

const setPreview = async (pid: number) => {
  await patchPack(sid.value, { preview_pid: pid })
  await refresh()
}

const remove = async (pid: number) => {
  await deleteSticker(sid.value, pid)
  await refresh()
}

const toggle = async () => {
  if (pack.value?.status === PACK_PUBLISHED) await unpublishPack(sid.value)
  else await publishPack(sid.value)
  await refresh()
}
</script>

<template>
  <section v-if="pack" class="flex flex-col gap-6">
    <KunLink :to="localePath('/me/packs')" class="text-sm">
      ← {{ t('publish.back') }}
    </KunLink>

    <div class="flex flex-wrap items-end gap-3">
      <label class="flex min-w-56 flex-1 flex-col gap-1 text-sm">
        {{ t('publish.name') }}
        <input
          v-model="title"
          class="border-default-200 bg-content1 h-10 border px-3"
          @change="saveTitle"
        >
      </label>
      <KunButton
        color="primary"
        :disabled="pack.status !== PACK_PUBLISHED && pack.count < 1"
        @click="toggle"
      >
        {{ pack.status === PACK_PUBLISHED ? t('publish.unpublish') : t('publish.publish') }}
      </KunButton>
    </div>

    <p class="text-default-500 text-sm">
      {{ t('publish.limit') }}
    </p>

    <div>
      <input
        ref="input"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        multiple
        class="hidden"
        @change="onFiles"
      >
      <KunButton variant="bordered" :disabled="uploading || pack.count >= 80" @click="input?.click()">
        {{ uploading ? t('publish.uploading') : t('publish.upload') }}
      </KunButton>
    </div>

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <KunCard
        v-for="sticker in pack.stickers"
        :key="sticker.pid"
        color="default"
        class-name="flex flex-col gap-2 p-3"
      >
        <img :src="sticker.thumb_url" alt="" class="aspect-square w-full object-cover">
        <label class="text-sm">
          {{ t('sticker.game') }}
          <input
            class="border-default-200 bg-content1 mt-1 h-9 w-full border px-2"
            :value="resolveMultilingual(sticker.game, locale)"
            @change="saveSticker(sticker, 'game', ($event.target as HTMLInputElement).value)"
          >
        </label>
        <label class="text-sm">
          {{ t('sticker.lass') }}
          <input
            class="border-default-200 bg-content1 mt-1 h-9 w-full border px-2"
            :value="resolveMultilingual(sticker.loli, locale)"
            @change="saveSticker(sticker, 'loli', ($event.target as HTMLInputElement).value)"
          >
        </label>
        <div class="flex justify-between">
          <KunButton size="sm" variant="light" @click="setPreview(sticker.pid)">
            {{ t('publish.preview') }}
          </KunButton>
          <KunButton size="sm" variant="light" @click="remove(sticker.pid)">
            {{ t('publish.remove') }}
          </KunButton>
        </div>
      </KunCard>
    </div>
  </section>
</template>
