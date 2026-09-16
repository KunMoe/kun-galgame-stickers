<script setup lang="ts">
import type { CommentFeedItem } from '~/features/comment/feed'
import { resolveMultilingual } from '~/features/pack/types'

const { t, locale } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const router = useRouter()
const user = useAuthUser()

const query = ref(String(route.query.q ?? ''))
const active = ref(route.query.tab === 'unread' && user.value ? 'unread' : 'latest')

const { state: unread, refresh: refreshUnread } = useUnreadComments()

const tabs = computed(() => {
  const items = [{ value: 'latest', textValue: t('comment.tabLatest'), icon: 'lucide:messages-square' }]
  if (user.value) {
    items.push({ value: 'unread', textValue: t('comment.tabUnread'), icon: 'lucide:bell-dot' })
  }
  return items
})

// The latest feed is the page's SSR payload; "load more" appends to it by
// cursor rather than re-fetching what the reader already has.
const { data: latest, status } = await useAsyncData('comments-latest', () => fetchLatestComments())
const rows = ref<CommentFeedItem[]>(latest.value?.items ?? [])
const cursor = ref(latest.value?.next_cursor ?? '')
const loadingMore = ref(false)
// enabled is false for a deployment without community and for a request that
// did not come back. Both mean "cannot show comments", which is a different
// sentence from "nobody has commented".
const feedDown = ref(latest.value?.enabled === false)

const loadMore = async () => {
  if (!cursor.value || loadingMore.value) return
  loadingMore.value = true
  const page = await fetchLatestComments(cursor.value)
  rows.value = [...rows.value, ...page.items]
  cursor.value = page.next_cursor ?? ''
  feedDown.value = !page.enabled
  loadingMore.value = false
}

// null means "not searching" -- distinct from a search that found nothing,
// which has to say so rather than quietly showing the latest feed again.
const results = ref<CommentFeedItem[] | null>(null)
const searching = ref(false)
const searchCursor = ref('')
const searchDown = ref(false)
const loadingMoreResults = ref(false)

// A stale slower response must not overwrite a newer one.
let ticket = 0
const runSearch = async (raw: string) => {
  const trimmed = raw.trim()
  const mine = ++ticket
  if (trimmed.length < 2) {
    results.value = null
    searching.value = false
    return
  }
  searching.value = true
  const found = await searchComments(trimmed)
  if (mine !== ticket) return
  results.value = found.items
  searchCursor.value = found.next_cursor ?? ''
  searchDown.value = !found.enabled
  searching.value = false
}

// community's search lane cannot filter by anchor, so a page of it can be all
// other sites' catalog comments. The API walks a few pages before answering,
// and this is the reader's way past a stretch longer than that.
const loadMoreResults = async () => {
  if (!searchCursor.value || loadingMoreResults.value) return
  loadingMoreResults.value = true
  const mine = ticket
  const page = await searchComments(query.value, searchCursor.value)
  // A keystroke while this was in flight has already started a fresher search.
  if (mine === ticket) {
    results.value = [...(results.value ?? []), ...page.items]
    searchCursor.value = page.next_cursor ?? ''
    searchDown.value = !page.enabled
  }
  loadingMoreResults.value = false
}

let timer: ReturnType<typeof setTimeout> | undefined
watch(query, (value) => {
  clearTimeout(timer)
  // Longer than the pack palette's debounce: this one is a round trip to
  // community's trigram scan, not a query against a local table.
  timer = setTimeout(() => runSearch(value), 350)
  // The query lives in the URL so a search can be shared and reloaded.
  router.replace({ query: { ...route.query, q: value.trim() || undefined } })
})
onScopeDispose(() => clearTimeout(timer))

watch(active, (value) => {
  router.replace({ query: { ...route.query, tab: value === 'unread' ? 'unread' : undefined } })
})

onMounted(() => {
  if (query.value.trim().length >= 2) void runSearch(query.value)
  void refreshUnread()
})

const packTitle = (title: Record<string, string | undefined>) =>
  resolveMultilingual(title, locale.value) || t('pack.untitled')

useKunSeo(() => ({
  title: t('comment.pageTitle'),
  description: t('comment.pageSubtitle'),
  image: kunOgImage('site', locale.value, {
    name: t('comment.pageTitle'),
    slogan: t('comment.pageSubtitle')
  })
}))
</script>

<template>
  <section class="flex flex-col gap-6">
    <header class="flex flex-col gap-2">
      <h1 class="text-2xl font-bold">{{ t('comment.pageTitle') }}</h1>
      <p class="text-default-500 text-sm">{{ t('comment.pageSubtitle') }}</p>
    </header>

    <KunInput
      v-model="query"
      :placeholder="t('comment.searchPlaceholder')"
      :aria-label="t('comment.searchLabel')"
    />

    <!-- A search answers across every pack, so it replaces the tabs rather
         than filtering whichever one happens to be open. -->
    <div v-if="results !== null" class="flex flex-col gap-3">
      <p class="text-default-500 text-sm">
        {{ searching ? t('comment.searching') : t('comment.searchCount', results.length) }}
      </p>
      <CommentFeedRow v-for="item in results" :key="item.id" :item="item" />
      <p v-if="!searching && searchDown" class="text-default-500 text-sm">
        {{ t('comment.unavailable') }}
      </p>
      <p
        v-else-if="!searching && !results.length && !searchCursor"
        class="text-default-500 text-sm"
      >
        {{ t('comment.searchEmpty') }}
      </p>
      <KunButton
        v-if="searchCursor && !searchDown"
        variant="bordered"
        class-name="self-center"
        :disabled="loadingMoreResults"
        @click="loadMoreResults"
      >
        {{ loadingMoreResults ? t('comment.loading') : t('comment.loadMore') }}
      </KunButton>
    </div>

    <template v-else>
      <KunTab v-model="active" :items="tabs" variant="underlined" name="comment-tabs" />

      <div v-if="active === 'latest'" class="flex flex-col gap-3">
        <p v-if="feedDown" class="text-default-500 text-sm">
          {{ t('comment.unavailable') }}
        </p>
        <!-- A cursor with nothing on it is a stretch of other sites' comments,
             not the end of the feed, so it does not say "no comments". -->
        <p
          v-else-if="!rows.length && !cursor && status !== 'pending'"
          class="text-default-500 text-sm"
        >
          {{ t('comment.feedEmpty') }}
        </p>
        <CommentFeedRow v-for="item in rows" :key="item.id" :item="item" />
        <KunButton
          v-if="cursor && !feedDown"
          variant="bordered"
          class-name="self-center"
          :disabled="loadingMore"
          @click="loadMore"
        >
          {{ loadingMore ? t('comment.loading') : t('comment.loadMore') }}
        </KunButton>
      </div>

      <div v-else class="flex flex-col gap-3">
        <p v-if="!unread.enabled" class="text-default-500 text-sm">
          {{ t('comment.unavailable') }}
        </p>
        <p v-else-if="!unread.packs.length" class="text-default-500 text-sm">
          {{ t('comment.unreadEmpty') }}
        </p>
        <NuxtLink
          v-for="row in unread.packs"
          :key="row.thread_id"
          :to="localePath(`/pack/${row.pack.id}`)"
          class="border-default-200 hover:border-default-400 flex items-center gap-3 border p-3 transition-colors"
        >
          <img
            v-if="row.pack.cover_thumb_url"
            :src="row.pack.cover_thumb_url"
            alt=""
            width="48"
            height="48"
            loading="lazy"
            class="border-default-200 size-12 shrink-0 border object-cover"
          >
          <span
            v-else
            class="bg-default-100 text-default-400 flex size-12 shrink-0 items-center justify-center"
          >
            <KunIcon name="lucide:image-off" class="text-sm" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="text-foreground block truncate text-sm font-medium">
              {{ packTitle(row.pack.title) }}
            </span>
            <span class="text-default-500 block text-xs">
              {{ t('comment.commentCount', row.posts_count) }}
            </span>
          </span>
          <KunChip size="sm" color="primary" variant="flat">{{ row.unread_count }}</KunChip>
        </NuxtLink>
      </div>
    </template>
  </section>
</template>
