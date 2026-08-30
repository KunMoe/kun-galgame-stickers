<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <div class="bg-background text-foreground flex min-h-dvh flex-col">
    <AppHeader />
    <main class="mx-auto flex w-full max-w-4xl flex-1 flex-col px-4 pt-24 pb-4 sm:px-20">
      <slot />
    </main>
    <footer class="text-default-500 flex flex-col items-center gap-1 px-4 py-4 text-sm">
      <p>{{ t('home.kun') }}</p>
      <p>
        {{ t('home.open') }}
        <KunLink :href="STICKER_GITHUB_REPO" target="_blank">
          GitHub
        </KunLink>
      </p>
      <p>
        {{ t('footer.poweredBy') }}
        <KunLink href="https://www.kungal.com" target="_blank">
          {{ t('footer.forumName') }}
        </KunLink>
        {{ t('footer.forumSuffix') }}
      </p>
    </footer>
  </div>
</template>
