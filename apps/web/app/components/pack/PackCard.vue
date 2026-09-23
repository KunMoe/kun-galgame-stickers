<script setup lang="ts">
import type { Pack } from '~/features/pack/types'
import { PACK_DRAFT, PACK_PUBLISHED, RATING_NSFW, resolveMultilingual } from '~/features/pack/types'

const props = defineProps<{ pack: Pack; showStatus?: boolean }>()

const { t, locale } = useI18n()
const localePath = useLocalePath()

const title = computed(
  () => resolveMultilingual(props.pack.title, locale.value) || t('pack.untitled')
)
const to = computed(() => localePath(`/pack/${props.pack.id}`))
const author = computed(() => toKunUser(props.pack.author))
</script>

<template>
  <article class="border-default-200 bg-content1 group flex flex-col border transition-colors hover:border-primary">
    <KunLink :to="to" class="relative block aspect-square overflow-hidden">
      <img
        v-if="pack.cover_thumb_url"
        :src="pack.cover_thumb_url"
        :alt="title"
        width="320"
        height="320"
        loading="lazy"
        class="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
      >
      <span v-else class="text-default-400 flex h-full w-full items-center justify-center">
        <KunIcon name="lucide:image-off" class="text-3xl" />
      </span>

      <span class="absolute left-2 top-2 flex flex-wrap gap-1">
        <KunChip v-if="pack.is_official" size="sm" color="default" variant="solid">
          {{ t('pack.official') }}
        </KunChip>
        <KunChip
          v-if="pack.content_rating === RATING_NSFW || pack.catalog_work?.content_rating === 'r18'"
          size="sm"
          color="danger"
          variant="solid"
        >
          R18
        </KunChip>
        <KunChip
          v-if="showStatus && pack.status !== PACK_PUBLISHED"
          size="sm"
          :color="pack.status === PACK_DRAFT ? 'warning' : 'default'"
          variant="solid"
        >
          {{ pack.status === PACK_DRAFT ? t('pack.draft') : t('pack.hidden') }}
        </KunChip>
      </span>
    </KunLink>

    <div class="flex flex-1 flex-col gap-2 p-3">
      <KunLink
        :to="to"
        color="default"
        underline="none"
        class-name="hover:text-primary line-clamp-2 text-sm font-medium transition-colors"
      >
        {{ title }}
      </KunLink>

      <div class="text-default-500 mt-auto flex items-center justify-between gap-2 text-xs">
        <KunLink
          :to="localePath(`/u/${pack.author.id}`)"
          color="default"
          underline="none"
          class-name="hover:text-primary flex min-w-0 items-center gap-1.5 transition-colors"
        >
          <KunAvatar :user="author" :is-navigation="false" size="sm" />
          <span class="truncate">{{ pack.author.name }}</span>
        </KunLink>
        <span class="flex shrink-0 items-center gap-1">
          <KunIcon name="lucide:layers" class="text-sm" />
          {{ pack.sticker_count }}
        </span>
      </div>
    </div>
  </article>
</template>
