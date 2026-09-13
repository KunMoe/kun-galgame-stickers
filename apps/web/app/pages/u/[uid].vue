<script setup lang="ts">
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const uid = computed(() => Number.parseInt(String(route.params.uid), 10))
if (!Number.isFinite(uid.value) || uid.value <= 0) {
  throw createError({ statusCode: 404, statusMessage: 'user not found', fatal: true })
}

const page = computed(() => {
  const parsed = Number.parseInt(String(route.query.page ?? '1'), 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1
})

const { data, status } = await useAsyncData(
  () => `user-${uid.value}-${page.value}`,
  () => fetchUserPacks(uid.value, { page: page.value }),
  { watch: [uid, page] }
)

const author = computed(() => data.value?.packs[0]?.author ?? null)
const total = computed(() => data.value?.total ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / 24)))

const currentPage = computed({
  get: () => page.value,
  set: (value: number) => router.push({ query: value === 1 ? {} : { page: String(value) } })
})

useKunSeo(() => ({
  title: author.value ? t('user.title', { name: author.value.name }) : t('user.fallback'),
  description: author.value
    ? t('user.seo', { name: author.value.name })
    : t('meta.description'),
  image: kunOgImage('site', locale.value, {
    name: author.value ? t('user.title', { name: author.value.name }) : t('user.fallback'),
    slogan: t('user.packCount', { count: total.value })
  }),
  type: 'profile'
}))
</script>

<template>
  <section class="flex flex-col gap-6">
    <header class="flex items-center gap-3">
      <img
        v-if="author?.avatar"
        :src="author.avatar"
        alt=""
        width="48"
        height="48"
        class="size-12 object-cover"
      >
      <div class="flex flex-col">
        <h1 class="text-xl font-bold">{{ author?.name || t('user.fallback') }}</h1>
        <p class="text-default-500 text-sm">{{ t('user.packCount', { count: total }) }}</p>
      </div>
    </header>

    <PackGrid
      :packs="data?.packs ?? []"
      :pending="status === 'pending'"
      :empty-text="t('user.empty')"
    />

    <KunPagination
      v-if="totalPages > 1"
      v-model:current-page="currentPage"
      :total-page="totalPages"
      class="self-center"
    />
  </section>
</template>
