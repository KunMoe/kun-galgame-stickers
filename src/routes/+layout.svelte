<script lang="ts">
  import '../tailwindcss.css'
  import Header from '$lib/components/Header.svelte'
  import ProgressBar from '$lib/components/ProgressBar.svelte'
  import BackToTop from '$lib/components/BackToTop.svelte'
  import { browser } from '$app/environment'
  import { page } from '$app/state'
  import { PUBLIC_KUN_STICKER_THEME } from '$env/static/public'
  import { m, detectLocaleFromPath, localizedPath } from '$lib/i18n'
  import { LOCALES } from '$lib/i18n/types'
  import { SITE_URL } from '$lib/config'
  import { locale } from '$lib/locale.svelte'
  import { theme, isDarkResolved } from '$lib/theme.svelte'
  import type { LayoutData } from './$types'

  interface Props {
    data: LayoutData
    children: import('svelte').Snippet
  }
  const { data, children }: Props = $props()

  // Sync server-resolved values into client state. Runs before the first DOM
  // update so apply effect sees the correct theme. Re-runs only when data
  // actually changes (e.g. navigation that returns a different cookie).
  $effect.pre(() => {
    locale.current = data.lang
    theme.current = data.theme
  })

  // Reactively reflect theme into <html> class + cookie, and follow system
  // preference changes while the user is in 'system' mode. Re-runs whenever
  // `theme.current` changes — including direct user toggles.
  $effect(() => {
    if (!browser) return
    const t = theme.current
    document.cookie = `${PUBLIC_KUN_STICKER_THEME}=${t}; path=/; max-age=${60 * 60 * 24 * 365}; samesite=lax`

    const apply = () =>
      document.documentElement.classList.toggle('kun-dark-mode', isDarkResolved(t))
    apply()

    if (t === 'system') {
      const media = window.matchMedia('(prefers-color-scheme: dark)')
      media.addEventListener('change', apply)
      return () => media.removeEventListener('change', apply)
    }
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

<div class="flex min-h-dvh flex-col bg-background text-foreground">
  <Header />

  <main class="mx-auto flex w-full max-w-4xl flex-1 flex-col px-4 pt-24 pb-4 sm:px-20">
    {@render children()}
    <BackToTop />
  </main>

  <footer class="flex flex-col items-center gap-1 px-4 py-4 text-sm text-default-500">
    <p>{m().home.kun}</p>
    <p>
      {m().home.open}
      <a
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit"
        class="text-primary underline underline-offset-4 hover:text-primary-600"
      >
        GitHub
      </a>
    </p>
    <p>
      {m().footer.poweredBy}
      <a
        href="https://www.kungal.com"
        class="text-primary underline underline-offset-4 hover:text-primary-600"
      >
        {m().footer.forumName}
      </a>
      {m().footer.forumSuffix}
    </p>
  </footer>
</div>
