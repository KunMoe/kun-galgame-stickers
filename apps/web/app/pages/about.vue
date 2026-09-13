<script setup lang="ts">
definePageMeta({ container: 'narrow' })

const { t, tm, rt, locale } = useI18n()
const localePath = useLocalePath()

const lines = (key: string): string[] =>
  (tm(key) as unknown[]).map((item) => rt(item as string))

const faq = computed(() =>
  (tm('about.faq.items') as unknown[]).map((item) => {
    const entry = item as { q: string; a: string }
    return { q: rt(entry.q), a: rt(entry.a) }
  })
)

useKunSeo(() => ({
  title: t('about.title'),
  description: t('meta.description'),
  image: kunOgImage('site', locale.value, {
    name: t('meta.title'),
    slogan: t('about.lead')
  })
}))
</script>

<template>
  <article class="flex flex-col gap-10">
    <header class="flex flex-col gap-3">
      <h1 class="text-3xl font-bold">{{ t('about.title') }}</h1>
      <p class="text-default-600">{{ t('about.lead') }}</p>
    </header>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.what.heading')" scale="h2" />
      <p v-for="(line, index) in lines('about.what.lines')" :key="index" class="text-default-600">
        {{ line }}
      </p>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.publish.heading')" scale="h2" />
      <p class="text-default-600">{{ t('about.publish.intro') }}</p>
      <ol class="text-default-600 list-decimal pl-6">
        <li v-for="(step, index) in lines('about.publish.steps')" :key="index">{{ step }}</li>
      </ol>
      <div>
        <KunButton color="primary" :href="localePath('/me/packs')">
          {{ t('about.publish.cta') }}
        </KunButton>
      </div>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.rules.heading')" scale="h2" />
      <ul class="text-default-600 list-disc pl-6">
        <li v-for="(rule, index) in lines('about.rules.items')" :key="index">{{ rule }}</li>
      </ul>
    </section>

    <section class="flex flex-col gap-4">
      <KunHeader :name="t('about.faq.heading')" scale="h2" />
      <div v-for="(item, index) in faq" :key="index" class="flex flex-col gap-1">
        <p class="text-primary font-medium">{{ item.q }}</p>
        <p class="text-default-600">{{ item.a }}</p>
      </div>
    </section>

    <section class="flex flex-col gap-3">
      <KunHeader :name="t('about.open.heading')" scale="h2" />
      <p class="text-default-600">{{ t('about.open.body') }}</p>
      <p>
        <KunLink :href="STICKER_GITHUB_REPO" target="_blank">GitHub</KunLink>
      </p>
    </section>
  </article>
</template>
