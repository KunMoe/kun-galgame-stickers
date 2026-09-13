<script setup lang="ts">
import { resolveMultilingual } from '~/features/pack/types'

const { t, locale } = useI18n()
const config = useRuntimeConfig()
const { scope, search, tag, linked, work, page, query, update } = useDiscoveryFilters()

const { data: tags } = await useAsyncData('discovery-tags', () => fetchTags())
const { data, status } = await useAsyncData(
  'discovery-packs',
  () => fetchPacks(query.value),
  { watch: [query] }
)

const packs = computed(() => data.value?.packs ?? [])
const total = computed(() => data.value?.total ?? 0)
const limit = 24
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / limit)))

// The game filter names itself from the packs that came back rather than
// calling catalog again -- every one of them carries the snapshot.
const workLabel = computed(() => {
  if (!work.value) return ''
  const match = packs.value.find((pack) => pack.catalog_work?.id === work.value)
  const name = resolveMultilingual(match?.catalog_work?.name, locale.value)
  return name || t('catalog.untitledWork')
})

const currentPage = computed({
  get: () => page.value,
  set: (value: number) => update({ page: value === 1 ? undefined : value })
})

useKunSeo(() => ({
  title: t('discovery.title'),
  description: t('meta.description'),
  image: kunOgImage('site', locale.value, {
    name: t('meta.title'),
    slogan: t('discovery.subtitle')
  }),
  jsonLd: {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: t('discovery.title'),
    description: t('meta.description'),
    isPartOf: { '@type': 'WebSite', name: t('meta.title'), url: config.public.siteUrl }
  }
}))
</script>

<template>
  <section class="flex flex-col gap-6">
    <header class="flex flex-col gap-2">
      <h1 class="text-2xl font-bold">{{ t('discovery.title') }}</h1>
      <p class="text-default-500 text-sm">{{ t('discovery.subtitle') }}</p>
    </header>

    <DiscoveryFilters
      :scope="scope"
      :tag="tag"
      :tags="tags ?? []"
      :linked="linked"
      @update="update"
    />

    <!-- Arriving from a game card filters to that game; say so and offer a way
         back, or the empty-looking grid reads as a bug. -->
    <div v-if="work" class="flex flex-wrap items-center gap-2">
      <KunChip size="sm" variant="flat">
        {{ workLabel }}
      </KunChip>
      <KunButton size="sm" variant="light" @click="update({ work: undefined })">
        {{ t('discovery.clearGame') }}
      </KunButton>
    </div>

    <p v-if="search || tag || linked || work" class="text-default-500 text-sm">
      {{ t('discovery.resultCount', total) }}
    </p>

    <PackGrid :packs="packs" :pending="status === 'pending'" />

    <KunPagination
      v-if="totalPages > 1"
      v-model:current-page="currentPage"
      :total-page="totalPages"
      :is-loading="status === 'pending'"
      class="self-center"
    />
  </section>
</template>
