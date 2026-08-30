<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const { t, locale } = useI18n()
const localePath = useLocalePath()
const name = ref('')
const pending = ref(false)

const create = async () => {
  const trimmed = name.value.trim()
  if (!trimmed || pending.value) return
  pending.value = true
  const pack = await createPack({ [localeField(locale.value)]: trimmed })
  pending.value = false
  if (!pack) return
  await navigateTo(localePath(`/me/packs/${pack.sid}`))
}
</script>

<template>
  <section class="mx-auto flex w-full max-w-md flex-col gap-4">
    <KunLink :to="localePath('/me/packs')" class="text-sm">
      ← {{ t('publish.back') }}
    </KunLink>
    <KunHeader :name="t('publish.new')" scale="h2" />
    <label class="flex flex-col gap-1 text-sm">
      {{ t('publish.name') }}
      <input
        v-model="name"
        class="border-default-200 bg-content1 h-10 border px-3"
        :placeholder="t('publish.namePlaceholder')"
        @keydown.enter="create"
      >
    </label>
    <KunButton color="primary" :disabled="pending || !name.trim()" @click="create">
      {{ pending ? t('publish.creating') : t('publish.create') }}
    </KunButton>
  </section>
</template>
