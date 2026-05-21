<script lang="ts">
  import Icon from '@iconify/svelte'
  import { m } from '$lib/i18n'
  import { theme } from '$lib/theme.svelte'
  import type { Theme } from '$lib/i18n/types'

  interface Props {
    onclose: () => void
  }
  const { onclose }: Props = $props()

  const items: { icon: string; value: Theme }[] = [
    { icon: 'line-md:moon-filled-alt-to-sunny-filled-loop-transition', value: 'light' },
    { icon: 'line-md:sunny-outline-to-moon-loop-transition', value: 'dark' },
    { icon: 'line-md:computer', value: 'system' }
  ]

  let container: HTMLDivElement
  $effect(() => container.focus())

  const handleBlur = () => {
    requestAnimationFrame(() => {
      if (!container.contains(document.activeElement)) onclose()
    })
  }
</script>

<div
  bind:this={container}
  tabindex="-1"
  role="menu"
  onfocusout={handleBlur}
  class="absolute top-12 right-0 z-50 flex w-36 flex-col gap-1 rounded-md border border-border bg-surface p-2 shadow-glow-sm focus:outline-none"
>
  {#each items as item (item.value)}
    {@const selected = theme.current === item.value}
    <button
      type="button"
      role="menuitemradio"
      aria-checked={selected}
      onclick={() => {
        theme.set(item.value)
        onclose()
      }}
      class="flex items-center gap-2 rounded px-3 py-1.5 text-left text-sm transition {selected
        ? 'bg-primary text-surface'
        : 'text-fg hover:bg-accent'}"
    >
      <Icon icon={item.icon} class="text-lg" />
      <span>{m().header[item.value]}</span>
    </button>
  {/each}
</div>
