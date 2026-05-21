<script lang="ts">
  import '../app.css'
  import Header from '$lib/components/Header.svelte'
  import ProgressBar from '$lib/components/ProgressBar.svelte'
  import BackToTop from '$lib/components/BackToTop.svelte'
  import { page } from '$app/state'
  import { m, detectLocaleFromPath, localizedPath } from '$lib/i18n'
  import { LOCALES } from '$lib/i18n/types'
  import { SITE_URL } from '$lib/config'
  import { locale } from '$lib/locale.svelte'
  import { theme } from '$lib/theme.svelte'
  import type { LayoutData } from './$types'

  interface Props {
    data: LayoutData
    children: import('svelte').Snippet
  }
  const { data, children }: Props = $props()

  $effect.pre(() => {
    locale.init(data.lang)
    theme.init(data.theme)
  })

  const basePath = $derived(detectLocaleFromPath(page.url.pathname).pathname)
</script>

<svelte:head>
  <title>{m().meta.title}</title>
  <meta name="description" content={m().meta.description} />
  <meta property="og:title" content={m().meta.og} />
  <meta property="og:type" content="website" />
  <meta property="og:description" content={m().meta.ogDescription} />
  <meta property="og:image" content="{SITE_URL}/title.webp" />
  <link rel="canonical" href="{SITE_URL}{page.url.pathname}" />
  {#each LOCALES as alt (alt)}
    <link rel="alternate" hreflang={alt} href="{SITE_URL}{localizedPath(alt, basePath)}" />
  {/each}
  <link rel="alternate" hreflang="x-default" href="{SITE_URL}{basePath}" />
</svelte:head>

<ProgressBar />

<div class="flex min-h-dvh flex-col bg-bg text-fg">
  <Header />

  <main class="mx-auto flex w-full max-w-4xl flex-1 flex-col px-4 pt-24 pb-4 sm:px-20">
    {@render children()}
    <BackToTop />
  </main>

  <footer class="flex flex-col items-center gap-1 px-4 py-3 text-sm text-muted">
    <p>{m().home.kun}</p>
    <p>
      {m().home.open}
      <a
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit"
        class="text-primary underline underline-offset-4"
      >
        GitHub
      </a>
    </p>
    <p>
      {m().footer.poweredBy}
      <a href="https://www.kungal.com" class="text-primary underline underline-offset-4">
        {m().footer.forumName}
      </a>
      {m().footer.forumSuffix}
    </p>
  </footer>
</div>
