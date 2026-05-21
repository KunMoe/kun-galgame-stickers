<script lang="ts">
  import { locale } from '$lib/locale.svelte'
  import { LOCALES, LOCALE_NATIVE_NAMES } from '$lib/i18n/types'

  interface Props {
    onclose: () => void
  }
  const { onclose }: Props = $props()

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
  class="absolute top-12 right-0 z-50 flex w-32 flex-col gap-1 rounded-md border border-border bg-surface p-2 shadow-glow-sm focus:outline-none"
>
  {#each LOCALES as code (code)}
    {@const selected = locale.current === code}
    <button
      type="button"
      role="menuitemradio"
      aria-checked={selected}
      onclick={async () => {
        await locale.switchTo(code)
        onclose()
      }}
      class="rounded px-3 py-1.5 text-left text-sm transition {selected
        ? 'bg-primary text-surface'
        : 'text-fg hover:bg-accent'}"
    >
      {LOCALE_NATIVE_NAMES[code]}
    </button>
  {/each}
</div>
