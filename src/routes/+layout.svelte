<script lang="ts">
  import Header from '~/lib/components/Header.svelte'
  import ProgressBar from '~/lib/components/ProgressBar.svelte'
  import Icon from '@iconify/svelte'
  import '~/styles/index.scss'
  import { onMount, beforeUpdate } from 'svelte'
  import { setColorSchemeContext } from '~/lib/contexts/theme'
  import { setLanguageContext } from '~/lib/contexts/language'
  import { t, locale } from '~/lib/language'

  export let data
  let showButton = false

  setColorSchemeContext(data.theme)
  setLanguageContext(data.language)

  const scrollToTop = () => {
    window.scrollTo({
      top: 0,
      behavior: 'smooth'
    })
  }

  const checkRoute = () => {
    if (window.location.pathname === '/') {
      showButton = false
      return
    }
    checkHeight()
  }

  const checkHeight = () => {
    if (document.body.scrollHeight > window.innerHeight) {
      showButton = true
    }
  }

  onMount(() => checkHeight())

  beforeUpdate(() => {
    checkRoute()
  })
</script>

<svelte:head>
  <title>{$t('seo.title')}</title>
  <link rel="icon" href="/favicon.webp" type="image/webp" />
  <meta name="description" content={$t('seo.description')} />

  <meta property="og:title" content={$t('seo.og')} />
  <meta property="og:type" content="website" />

  <meta property="og:description" content={$t('seo.og-desc')} />
  <meta property="og:image" content="/title.webp" />
</svelte:head>

<ProgressBar />

<div class="app">
  <Header />

  <main>
    <slot />

    {#if showButton}
      <button aria-label="Back to Top | 返回顶部" class="top" on:click={scrollToTop}>
        <Icon icon="line-md:arrow-close-up" />
      </button>
    {/if}
  </main>

  <footer>
    <p>{$t('home.kun')}</p>
    <p>
      {$t('home.open')}
      <a
        aria-label="KUN Visual Novel Open Source GitHub Repository | 鲲 Galgame 开源 GitHub 仓库"
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit"
      >
        GitHub
      </a>
    </p>
    {#if $locale === 'zh-cn'}
      <p>
        由
        <a aria-label="KUN Visual Novel Forum | 鲲 Galgame 论坛" href="https://www.kungal.com">
          鲲 Galgame 论坛
        </a>
        提供支持
      </p>
    {/if}

    {#if $locale === 'en-us'}
      <p>
        Powered by
        <a aria-label="KUN Visual Novel Forum | 鲲 Galgame 论坛" href="https://www.kungal.com">
          KUN Visual Novel Forum
        </a>
      </p>
    {/if}
  </footer>
</div>

<style lang="scss">
  .app {
    display: flex;
    flex-direction: column;
    min-height: 100dvh;
    color: var(--kungalgame-font-color-3);
    background-color: var(--kungalgame-trans-blue-0);
  }

  main {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 5rem;
    padding-bottom: 1rem;
    width: 100%;
    max-width: 64rem;
    margin: 0 auto;
    box-sizing: border-box;
  }

  .top {
    position: fixed;
    bottom: 2rem;
    right: 1rem;
    background: none;
    border: 1px solid var(--kungalgame-blue-4);
    height: 50px;
    width: 50px;
    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 22px;
    color: var(--kungalgame-blue-4);
    background-color: var(--kungalgame-trans-white-5);
    backdrop-filter: blur(5px);
  }

  footer {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    padding: 12px;

    a {
      color: var(--kungalgame-blue-5);
      text-decoration: underline;
      text-underline-offset: 3px;
    }
  }

  @media (max-width: 700px) {
    main {
      padding: 5rem 1rem;
      padding-bottom: 1rem;
    }
  }
</style>
