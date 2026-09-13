<script setup lang="ts">
const { t, locale } = useI18n()
const route = useRoute()

const slug = computed(() => String(route.params.slug))
const page = computed(() => {
  const parsed = Number.parseInt(String(route.query.page ?? '1'), 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1
})

const { data, status } = await useAsyncData(
  () => `tag-${slug.value}-${page.value}`,
  () => fetchPacks({ tag: slug.value, page: page.value }),
  { watch: [slug, page] }
)

const total = computed(() => data.value?.total ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / 24)))

const router = useRouter()
const currentPage = computed({
  get: () => page.value,
  set: (value: number) =>
    router.push({ query: value === 1 ? {} : { page: String(value) } })
})

useKunSeo(() => ({
  title: `#${slug.value}`,
  description: t('tag.seo', { tag: slug.value }),
  image: kunOgImage('site', locale.value, {
    name: `#${slug.value}`,
    slogan: t('tag.seo', { tag: slug.value })
  })
}))
</script>

<template>
  <section class="flex flex-col gap-6">
    <header class="flex flex-col gap-1">
      <h1 class="text-2xl font-bold">#{{ slug }}</h1>
      <p class="text-default-500 text-sm">{{ t('discovery.resultCount', total) }}</p>
    </header>

    <PackGrid :packs="data?.packs ?? []" :pending="status === 'pending'" />

    <KunPagination
      v-if="totalPages > 1"
      v-model:current-page="currentPage"
      :total-page="totalPages"
      class="self-center"
    />
  </section>
</template>
