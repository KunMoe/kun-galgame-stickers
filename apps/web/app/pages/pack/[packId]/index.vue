<script setup lang="ts">
import { PACK_PUBLISHED, RATING_NSFW, resolveMultilingual } from '~/features/pack/types'

const { t, locale } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const config = useRuntimeConfig()

const packId = computed(() => String(route.params.packId))
const { data: pack } = await useAsyncData(`pack-${packId.value}`, () => fetchPack(packId.value))

if (!pack.value) {
  throw createError({ statusCode: 404, statusMessage: 'pack not found', fatal: true })
}

const title = computed(() => resolveMultilingual(pack.value?.title, locale.value) || t('pack.untitled'))
const description = computed(() => resolveMultilingual(pack.value?.description, locale.value))
const author = computed(() => (pack.value ? toKunUser(pack.value.author) : null))
const tagLabel = (name: Record<string, string | undefined>, slug: string) =>
  resolveMultilingual(name, locale.value) || slug

const gameLimit = 6
const showAllGames = ref(false)

// The pack's declared game leads; the games its stickers point at follow,
// deduplicated against it. A mixed pack has no declared game and just lists
// what is inside.
const games = computed(() => {
  const declared = pack.value?.catalog_work
  const fromStickers = (pack.value?.works ?? []).filter((work) => work.id !== declared?.id)
  return declared ? [declared, ...fromStickers] : fromStickers
})

const shownGames = computed(() =>
  showAllGames.value ? games.value : games.value.slice(0, gameLimit)
)

useKunSeo(() => ({
  title: title.value,
  description: description.value || t('pack.seoFallback', { name: title.value }),
  image: kunOgImage(`pack/${packId.value}`, locale.value),
  type: 'article',
  // ImageGallery rather than CreativeWork: a pack is a set of images, and the
  // count and the author are the two facts a result card can use.
  jsonLd: {
    '@context': 'https://schema.org',
    '@type': 'ImageGallery',
    name: title.value,
    description: description.value || undefined,
    url: `${config.public.siteUrl}/pack/${packId.value}`,
    image: pack.value?.cover_url || undefined,
    datePublished: pack.value?.published_at,
    dateModified: pack.value?.updated_at,
    author: pack.value ? { '@type': 'Person', name: pack.value.author.name } : undefined,
    numberOfItems: pack.value?.sticker_count,
    isFamilyFriendly: pack.value?.content_rating !== RATING_NSFW
  }
}))
</script>

<template>
  <article v-if="pack" class="flex flex-col gap-8">
    <header class="flex flex-col gap-4 sm:flex-row sm:gap-6">
      <img
        v-if="pack.cover_thumb_url"
        :src="pack.cover_thumb_url"
        :alt="title"
        width="320"
        height="320"
        class="border-default-200 size-32 shrink-0 border object-cover sm:size-40"
      >

      <div class="flex min-w-0 flex-1 flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunChip v-if="pack.is_official" size="sm" color="default" variant="solid">
            {{ t('pack.official') }}
          </KunChip>
          <KunChip v-if="pack.content_rating === RATING_NSFW" size="sm" color="danger" variant="solid">
            R18
          </KunChip>
          <KunChip v-if="pack.status !== PACK_PUBLISHED" size="sm" color="warning" variant="solid">
            {{ t('pack.draft') }}
          </KunChip>
        </div>

        <h1 class="text-2xl font-bold">{{ title }}</h1>
        <p v-if="description" class="text-default-600 text-sm">{{ description }}</p>

        <div class="text-default-500 flex flex-wrap items-center gap-4 text-sm">
          <KunLink
            :to="localePath(`/u/${pack.author.id}`)"
            color="default"
            underline="none"
            class-name="hover:text-primary flex items-center gap-2 transition-colors"
          >
            <KunAvatar :user="author" :is-navigation="false" size="sm" />
            <span>{{ pack.author.name }}</span>
          </KunLink>
          <span>{{ t('pack.stickerCount', pack.sticker_count) }}</span>
          <span>{{ t('pack.downloadCount', pack.download_count) }}</span>
        </div>

        <div v-if="pack.tags.length" class="flex flex-wrap gap-2">
          <KunLink
            v-for="item in pack.tags"
            :key="item.id"
            :to="localePath(`/tag/${item.slug}`)"
            color="default"
            underline="none"
          >
            <KunChip size="sm" variant="flat">#{{ tagLabel(item.name, item.slug) }}</KunChip>
          </KunLink>
        </div>

        <div>
          <KunButton color="primary" :href="packDownloadUrl(pack.id)" class="gap-2">
            <KunIcon name="lucide:download" class="text-lg" />
            {{ t('pack.downloadAll') }}
          </KunButton>
        </div>
      </div>
    </header>

    <!-- The game a pack declares, plus whatever else its stickers point at.
         A single-game pack shows one card; the seeded official packs span
         dozens, so those collapse behind a disclosure instead of pushing the
         stickers off the screen. -->
    <section v-if="games.length || pack.characters.length" class="flex flex-col gap-4">
      <div v-if="games.length" class="flex flex-col gap-2">
        <h2 class="text-sm font-medium">{{ t('catalog.fromGame') }}</h2>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <CatalogWorkCard
            v-for="work in shownGames"
            :key="work.id"
            :work="work"
            :compact="games.length > 1"
          />
        </div>
        <KunButton
          v-if="games.length > gameLimit"
          variant="light"
          size="sm"
          class-name="self-start"
          @click="showAllGames = !showAllGames"
        >
          {{ showAllGames ? t('catalog.showLess') : t('catalog.showAllGames', { count: games.length }) }}
        </KunButton>
      </div>

      <div v-if="pack.characters.length" class="flex flex-col gap-2">
        <h2 class="text-sm font-medium">{{ t('catalog.characters') }}</h2>
        <div class="flex flex-wrap gap-2">
          <CatalogCharacterChip
            v-for="character in pack.characters"
            :key="character.id"
            :character="character"
          />
        </div>
      </div>
    </section>

    <StickerGrid :stickers="pack.stickers" :pack-id="pack.id" />

    <CommentSection :pack-id="pack.id" />
  </article>
</template>
