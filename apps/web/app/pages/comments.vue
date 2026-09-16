<script setup lang="ts">
import type { CommentFeedItem } from '~/features/comment/feed'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const user = useAuthUser()

const query = ref(String(route.query.q ?? ''))
const active = ref(route.query.tab === 'notifications' && user.value ? 'notifications' : 'latest')

const tabs = computed(() => {
  const items = [{ value: 'latest', textValue: t('comment.tabLatest'), icon: 'lucide:messages-square' }]
  if (user.value) {
    items.push({ value: 'notifications', textValue: t('notification.title'), icon: 'lucide:bell' })
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
  router.replace({ query: { ...route.query, tab: value === 'notifications' ? 'notifications' : undefined } })
})

onMounted(() => {
  if (query.value.trim().length >= 2) void runSearch(query.value)
})

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

      <NotificationList v-else />
    </template>
  </section>
</template>
