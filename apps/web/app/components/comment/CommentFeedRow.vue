<script setup lang="ts">
import type { CommentFeedItem } from '~/features/comment/feed'
import { resolveMultilingual } from '~/features/pack/types'

const props = defineProps<{ item: CommentFeedItem }>()

const { t, locale } = useI18n()
const localePath = useLocalePath()

const packTitle = computed(
  () => resolveMultilingual(props.item.pack.title, locale.value) || t('pack.untitled')
)
const packPath = computed(() => localePath(`/pack/${props.item.pack.id}`))
const formatTime = (value: string) => new Date(value).toLocaleString()
</script>

<template>
  <article class="border-default-200 flex gap-3 border p-3">
    <NuxtLink :to="packPath" class="shrink-0" :aria-label="packTitle">
      <img
        v-if="item.pack.cover_thumb_url"
        :src="item.pack.cover_thumb_url"
        alt=""
        width="48"
        height="48"
        loading="lazy"
        class="border-default-200 size-12 border object-cover"
      >
      <span
        v-else
        class="bg-default-100 text-default-400 flex size-12 items-center justify-center"
      >
        <KunIcon name="lucide:image-off" class="text-sm" />
      </span>
    </NuxtLink>

    <div class="min-w-0 flex-1">
      <div class="text-default-500 flex flex-wrap items-center gap-2 text-xs">
        <NuxtLink
          :to="localePath(`/u/${item.author.id}`)"
          class="text-foreground font-medium"
        >
          {{ item.author.name }}
        </NuxtLink>
        <time :datetime="item.created_at">{{ formatTime(item.created_at) }}</time>
      </div>

      <!-- The body is cooked and sanitized upstream; this site renders it and
           never re-processes it. -->
      <div class="prose-sm text-foreground mt-1 line-clamp-3 max-w-none text-sm break-words">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-html="item.content_html" />
      </div>

      <NuxtLink
        :to="packPath"
        class="text-default-500 hover:text-foreground mt-1.5 flex items-center gap-1 text-xs transition-colors"
      >
        <KunIcon name="lucide:package" class="text-sm" />
        <span class="truncate">{{ packTitle }}</span>
      </NuxtLink>
    </div>
  </article>
</template>
