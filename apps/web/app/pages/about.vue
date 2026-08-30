<script setup lang="ts">
const { t, tm } = useI18n()
const packIds = [1, 2, 3, 4, 5, 6, 7]

useSeoMeta({
  title: () => t('about.title'),
  description: () => t('meta.description')
})

const lines = computed(() => tm('about.introduction.lines') as string[])
const purpose = computed(() => tm('about.introduction.purpose') as string[])
const rules = computed(() => tm('about.rules.items') as string[])
const gameLines = computed(() => tm('about.games.lines') as string[])
const faq = computed(() => tm('about.faq.items') as { q: string; a: string }[])
const tips = computed(() => tm('about.tips.lines') as string[])
</script>

<template>
  <article class="flex flex-col gap-12">
    <h1 class="text-center text-3xl font-bold">
      <KunLink :href="STICKER_GITHUB_REPO" target="_blank">
        {{ t('about.repo') }}
      </KunLink>
    </h1>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.introduction.heading')" scale="h2" />
      <p v-for="line in lines" :key="line">{{ line }}</p>
      <video poster="/title.webp" controls loop playsinline class="w-full border">
        <source src="/introduction.mp4" type="video/mp4" >
      </video>
      <p class="mt-4">{{ t('about.introduction.bilibiliPrefix') }}</p>
      <p>
        Bilibili:
        <KunLink href="https://www.bilibili.com/video/BV19u4m1c7ZN" target="_blank">
          {{ t('about.introduction.bilibiliLink') }}
        </KunLink>
      </p>
      <p class="mt-4">{{ t('about.introduction.purposeIntro') }}</p>
      <ul class="list-disc pl-6">
        <li v-for="item in purpose" :key="item">{{ item }}</li>
      </ul>
      <p class="mt-4">{{ t('about.introduction.future') }}</p>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.telegram.heading')" scale="h2" />
      <p>{{ t('about.telegram.intro') }}</p>
      <ul class="list-disc pl-6">
        <li v-for="id in packIds" :key="id">
          <KunLink :href="`https://t.me/addstickers/KUNgal${id}`" target="_blank">
            {{ t('about.telegram.packLabel') }} [{{ id }}]
          </KunLink>
        </li>
      </ul>
      <p>{{ t('about.telegram.note') }}</p>
      <p>{{ t('about.telegram.downloadIntro') }}</p>
      <KunLink
        :href="STICKER_GITHUB_RELEASES"
        target="_blank"
      >
        {{ t('about.telegram.downloadLink') }}
      </KunLink>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.rules.heading')" scale="h2" />
      <p>{{ t('about.rules.intro') }}</p>
      <ul class="list-disc pl-6">
        <li v-for="item in rules" :key="item">{{ item }}</li>
      </ul>
      <strong>{{ t('about.rules.note') }}</strong>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.games.heading')" scale="h2" />
      <p v-for="line in gameLines" :key="line">{{ line }}</p>
      <KunLink
        :href="STICKER_GITHUB_GAME_LIST"
        target="_blank"
      >
        {{ t('about.games.linkLabel') }}
      </KunLink>
    </section>

    <section class="flex flex-col gap-4">
      <KunHeader :name="t('about.faq.heading')" scale="h2" />
      <div v-for="item in faq" :key="item.q" class="flex flex-col gap-1">
        <p class="text-primary font-bold">Q: {{ item.q }}</p>
        <p>A: {{ item.a }}</p>
      </div>
    </section>

    <section class="flex flex-col gap-2">
      <KunHeader :name="t('about.tips.heading')" scale="h2" />
      <p v-for="line in tips" :key="line">{{ line }}</p>
      <p class="text-default-500">{{ t('about.tips.note') }}</p>
    </section>
  </article>
</template>
