<script setup lang="ts">
import type { Comment, FlagReason, LikeResult } from '~/features/comment/api'

const props = defineProps<{ packId: string }>()

const { t } = useI18n()
const route = useRoute()
const user = useAuthUser()
const mutate = useMutation()

const { data, refresh } = await useAsyncData(`comments-${props.packId}`, () =>
  fetchComments(props.packId)
)

// The first page is the SSR payload. Later pages and the reader's own writes
// are laid over it here rather than re-fetched, so saving an edit on page three
// does not send the reader back to page one.
const list = ref<Comment[]>([])
const cursor = ref('')
const total = ref(0)
const loadingMore = ref(false)
watch(
  data,
  (page) => {
    list.value = page?.comments ?? []
    cursor.value = page?.next_cursor ?? ''
    total.value = page?.total ?? 0
  },
  { immediate: true }
)

const loadMore = async () => {
  if (!cursor.value || loadingMore.value) return
  loadingMore.value = true
  const page = await fetchComments(props.packId, cursor.value)
  loadingMore.value = false
  if (!page) {
    useKunMessage(t('comment.unavailable'), 'error')
    return
  }
  list.value = mergeComments(list.value, page.comments)
  cursor.value = page.next_cursor ?? ''
  total.value = page.total
}

const draft = ref('')
const sending = ref(false)
const editing = ref<Comment | null>(null)
const editDraft = ref('')
const removing = ref<Comment | null>(null)
const replyingTo = ref<Comment | null>(null)
const reporting = ref<Comment | null>(null)
const reportReason = ref<FlagReason>(0)
const reportNote = ref('')

const nodes = computed(() => nestComments(list.value))
// A null payload means the request failed, not that there are no comments --
// showing an input that will fail on submit is worse than showing nothing.
const enabled = computed(() => data.value?.enabled === true)

const submit = async () => {
  const body = draft.value.trim()
  if (!body || sending.value) return
  sending.value = true
  const created = await mutate(() => addComment(props.packId, body, replyingTo.value?.id))
  sending.value = false
  if (!created) return
  draft.value = ''
  replyingTo.value = null
  // The first comment is what creates the thread, and the section needs the
  // thread's id before it can offer to subscribe; every later one is shown in
  // place.
  if (!threadId.value) {
    await refresh()
    return
  }
  list.value = mergeComments(list.value, [created])
  total.value += 1
}

const startReply = (comment: Comment) => {
  replyingTo.value = comment
  editing.value = null
  // The composer is one box at the top; jumping to it is how the reader knows
  // where their reply is going.
  if (import.meta.client) composer.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

const openReport = (comment: Comment) => {
  reporting.value = comment
  reportReason.value = 0
  reportNote.value = ''
}

const submitReport = async () => {
  const target = reporting.value
  if (!target) return
  const done = await mutate(() => reportComment(target.id, reportReason.value, reportNote.value))
  reporting.value = null
  if (done) useKunMessage(t('comment.reported'), 'success')
}

const composer = ref<HTMLElement | null>(null)

const reasonOptions = computed(() =>
  FLAG_REASONS.map((value) => ({ value, label: t(`comment.reason${value}`) }))
)

const startEdit = (comment: Comment) => {
  editing.value = comment
  editDraft.value = comment.content_raw
}

const saveEdit = async () => {
  const target = editing.value
  const body = editDraft.value.trim()
  if (!target || !body) return
  const saved = await mutate(() => editComment(target.id, body))
  if (!saved) return
  editing.value = null
  list.value = mergeComments(list.value, [saved])
}

const confirmRemove = async () => {
  const target = removing.value
  removing.value = null
  if (!target) return
  const done = await mutate(() => deleteComment(target.id))
  if (!done) return
  // Replies stay: a tombstoned root still leaves its answers standing, and
  // nestComments shows them on their own.
  list.value = list.value.filter((comment) => comment.id !== target.id)
  total.value = Math.max(0, total.value - 1)
}

const applyLike = (id: number, result: LikeResult) => {
  list.value = list.value.map((comment) =>
    comment.id === id ? { ...comment, is_liked: result.liked, like_count: result.like_count } : comment
  )
}

const signIn = () => startOAuthLogin(route.fullPath)

// Everything below is the thread's own state rather than its posts: there is
// no thread at all until the first comment creates one, so every one of these
// is inert on a pack nobody has spoken about yet.
const threadId = computed(() => data.value?.thread_id ?? 0)
const notifyLevel = ref<NotifyLevel | null>(null)
watch(data, (page) => { notifyLevel.value = page?.viewer?.notification_level ?? null }, { immediate: true })

const subscribed = computed(() => notifyLevel.value !== null && notifyLevel.value >= NOTIFY.tracking)

const toggleSubscribe = async () => {
  const muting = subscribed.value
  const state = await mutate(() =>
    setCommentNotification(threadId.value, muting ? NOTIFY.muted : NOTIFY.watching)
  )
  if (!state) return
  notifyLevel.value = state.notification_level
  useKunMessage(t(muting ? 'comment.muted' : 'comment.subscribed'), 'success')
}

const { clear: clearUnread } = useUnreadComments()

// The receipt is a write, so it waits for the client and only fires when there
// is something new to report. A failure costs the reader a badge that clears
// on their next visit, which is not worth a toast over their comments.
onMounted(() => {
  const page = data.value
  if (!user.value || !page?.thread_id || !page.highest_post_number) return
  if ((page.viewer?.last_read_post_number ?? 0) >= page.highest_post_number) return
  markCommentsRead(page.thread_id, page.highest_post_number)
    .then(() => clearUnread(page.thread_id))
    .catch(() => {})
})
</script>

<template>
  <section v-if="enabled" class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <h2 class="text-lg font-medium">
        {{ t('comment.title') }}
        <span v-if="total" class="text-default-500 text-sm font-normal">{{ total }}</span>
      </h2>
      <!-- Commenting already subscribes you upstream, so this button exists for
           the two deliberate choices: muting a wall you are in, and following
           one you have not spoken in. -->
      <KunButton
        v-if="user && threadId"
        size="sm"
        variant="light"
        class-name="gap-1.5"
        :aria-label="t(subscribed ? 'comment.mute' : 'comment.subscribe')"
        @click="toggleSubscribe"
      >
        <KunIcon :name="subscribed ? 'lucide:bell-off' : 'lucide:bell'" class="text-base" />
        <span class="hidden sm:inline">{{ t(subscribed ? 'comment.mute' : 'comment.subscribe') }}</span>
      </KunButton>
    </div>

    <div v-if="user" ref="composer" class="flex flex-col gap-2">
      <div
        v-if="replyingTo"
        class="border-default-200 text-default-500 flex items-center justify-between gap-2 border px-3 py-1.5 text-xs"
      >
        <span>{{ t('comment.replyingTo', { name: replyingTo.author.name }) }}</span>
        <KunButton size="sm" variant="light" @click="replyingTo = null">
          {{ t('auth.cancel') }}
        </KunButton>
      </div>
      <KunTextarea
        v-model="draft"
        :rows="3"
        :placeholder="t('comment.placeholder')"
        :maxlength="MAX_COMMENT_LENGTH"
      />
      <div class="flex items-center justify-between gap-3">
        <span class="text-default-400 text-xs">{{ t('comment.markdownHint') }}</span>
        <KunButton color="primary" size="sm" :disabled="!draft.trim() || sending" @click="submit">
          {{ sending ? t('comment.sending') : t('comment.send') }}
        </KunButton>
      </div>
    </div>
    <div v-else class="border-default-200 flex items-center justify-between gap-3 border p-3">
      <span class="text-default-500 text-sm">{{ t('comment.signInPrompt') }}</span>
      <KunButton size="sm" variant="flat" @click="signIn">{{ t('auth.login') }}</KunButton>
    </div>

    <p v-if="!list.length && !cursor" class="text-default-500 text-sm">
      {{ t('comment.empty') }}
    </p>

    <ul v-else class="flex flex-col gap-5">
      <li v-for="node in nodes" :key="node.id" class="flex flex-col gap-3">
        <div v-if="editing?.id === node.id" class="flex flex-col gap-2">
          <KunTextarea v-model="editDraft" :rows="3" :maxlength="MAX_COMMENT_LENGTH" />
          <div class="flex gap-2">
            <KunButton size="sm" color="primary" @click="saveEdit">{{ t('editor.save') }}</KunButton>
            <KunButton size="sm" variant="light" @click="editing = null">
              {{ t('auth.cancel') }}
            </KunButton>
          </div>
        </div>
        <CommentItem
          v-else
          :comment="node"
          @reply="startReply"
          @edit="startEdit"
          @remove="removing = $event"
          @report="openReport"
          @liked="applyLike"
        />

        <div
          v-if="node.replies.length"
          class="border-default-200 ml-4 flex flex-col gap-3 border-l pl-3"
        >
          <template v-for="reply in node.replies" :key="reply.id">
            <div v-if="editing?.id === reply.id" class="flex flex-col gap-2">
              <KunTextarea v-model="editDraft" :rows="3" :maxlength="MAX_COMMENT_LENGTH" />
              <div class="flex gap-2">
                <KunButton size="sm" color="primary" @click="saveEdit">
                  {{ t('editor.save') }}
                </KunButton>
                <KunButton size="sm" variant="light" @click="editing = null">
                  {{ t('auth.cancel') }}
                </KunButton>
              </div>
            </div>
            <CommentItem
              v-else
              :comment="reply"
              nested
              @reply="startReply"
              @edit="startEdit"
              @remove="removing = $event"
              @report="openReport"
              @liked="applyLike"
            />
          </template>
        </div>
      </li>
    </ul>

    <!-- The wall is oldest first, so what is left to load is the newer end. -->
    <KunButton
      v-if="cursor"
      variant="bordered"
      class-name="self-center"
      :disabled="loadingMore"
      @click="loadMore"
    >
      {{ loadingMore ? t('comment.loading') : t('comment.loadMore') }}
    </KunButton>

    <KunModal
      :model-value="!!reporting"
      :title="t('comment.reportTitle')"
      @update:model-value="reporting = null"
    >
      <div class="flex flex-col gap-3">
        <p class="text-default-600 text-sm">{{ t('comment.reportPrompt') }}</p>
        <KunRadioGroup
          v-model="reportReason"
          :options="reasonOptions"
          orientation="vertical"
          :aria-label="t('comment.reportTitle')"
        />
        <KunTextarea
          v-model="reportNote"
          :rows="2"
          :maxlength="200"
          :placeholder="t('comment.reportNote')"
        />
        <div class="flex justify-end gap-2">
          <KunButton variant="light" @click="reporting = null">{{ t('auth.cancel') }}</KunButton>
          <KunButton color="danger" @click="submitReport">{{ t('comment.report') }}</KunButton>
        </div>
      </div>
    </KunModal>

    <KunModal :model-value="!!removing" :title="t('comment.deleteTitle')" @update:model-value="removing = null">
      <p class="text-default-600 mb-4 text-sm">{{ t('comment.deletePrompt') }}</p>
      <div class="flex justify-end gap-2">
        <KunButton variant="light" @click="removing = null">{{ t('auth.cancel') }}</KunButton>
        <KunButton color="danger" @click="confirmRemove">{{ t('comment.delete') }}</KunButton>
      </div>
    </KunModal>
  </section>
</template>
