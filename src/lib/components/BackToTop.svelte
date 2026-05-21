<script lang="ts">
  import Icon from '@iconify/svelte'
  import { m } from '$lib/i18n'

  let visible = $state(false)

  const handleScroll = () => {
    visible = window.scrollY > window.innerHeight * 0.4
  }

  $effect(() => {
    handleScroll()
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  })

  const scrollToTop = () => window.scrollTo({ top: 0, behavior: 'smooth' })
</script>

{#if visible}
  <button
    type="button"
    aria-label={m().sticker.backToTop}
    onclick={scrollToTop}
    class="fixed right-4 bottom-8 z-50 flex h-12 w-12 items-center justify-center border border-primary bg-content1/80 text-2xl text-primary shadow-sm backdrop-blur transition hover:bg-primary hover:text-white"
  >
    <Icon icon="line-md:arrow-close-up" />
  </button>
{/if}
