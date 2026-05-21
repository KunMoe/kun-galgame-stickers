<script lang="ts">
  import Icon from '@iconify/svelte'
  import { page } from '$app/state'
  import { m, localizedPath } from '$lib/i18n'
  import { locale } from '$lib/locale.svelte'
  import { detectLocaleFromPath } from '$lib/i18n'
  import ThemeMenu from './ThemeMenu.svelte'
  import LanguageMenu from './LanguageMenu.svelte'

  let themeOpen = $state(false)
  let langOpen = $state(false)

  const basePath = $derived(detectLocaleFromPath(page.url.pathname).pathname)
  const home = $derived(localizedPath(locale.current, '/'))
  const about = $derived(localizedPath(locale.current, '/about'))
</script>

<header
  class="fixed inset-x-0 top-0 z-[1007] flex h-14 items-center gap-4 border-b border-border bg-surface-translucent px-4 backdrop-blur-md sm:px-12"
>
  <a href={home} aria-label={m().header.title} class="flex items-center gap-3">
    <img src="/favicon.webp" alt="" class="h-10 w-10" />
    <span class="hidden text-lg text-fg sm:block">{m().header.title}</span>
  </a>

  <nav class="flex flex-1 items-center justify-center gap-6 text-base">
    <a
      href={home}
      aria-current={basePath === '/' ? 'page' : undefined}
      class="text-primary underline decoration-1 underline-offset-4 hover:text-primary-hover"
    >
      {m().header.home}
    </a>
    <a
      href={about}
      aria-current={basePath === '/about' ? 'page' : undefined}
      class="text-primary underline decoration-1 underline-offset-4 hover:text-primary-hover"
    >
      {m().header.about}
    </a>
  </nav>

  <div class="flex items-center gap-4 text-xl text-primary">
    <div class="relative">
      <button
        type="button"
        aria-label="Theme switch"
        aria-haspopup="menu"
        aria-expanded={themeOpen}
        onclick={() => (themeOpen = !themeOpen)}
        class="flex h-9 w-9 items-center justify-center rounded transition hover:bg-accent"
      >
        <Icon icon="line-md:light-dark-loop" />
      </button>
      {#if themeOpen}
        <ThemeMenu onclose={() => (themeOpen = false)} />
      {/if}
    </div>

    <div class="relative">
      <button
        type="button"
        aria-label="Language switch"
        aria-haspopup="menu"
        aria-expanded={langOpen}
        onclick={() => (langOpen = !langOpen)}
        class="flex h-9 w-9 items-center justify-center rounded transition hover:bg-accent"
      >
        <Icon icon="material-symbols:language" />
      </button>
      {#if langOpen}
        <LanguageMenu onclose={() => (langOpen = false)} />
      {/if}
    </div>

    <a
      href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit"
      target="_blank"
      rel="noopener noreferrer"
      aria-label="GitHub repository"
      class="flex h-9 w-9 items-center justify-center rounded transition hover:bg-accent"
    >
      <Icon icon="line-md:github-loop" />
    </a>
  </div>
</header>
