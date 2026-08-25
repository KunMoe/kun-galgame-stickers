<script lang="ts">
  import { locale } from '$lib/locale.svelte'
  import { LOCALES, LOCALE_NATIVE_NAMES } from '$lib/i18n/types'

  interface Props {
    onclose: () => void
  }
  const { onclose }: Props = $props()

  let container: HTMLDivElement | undefined = $state()
  $effect(() => container?.focus())

  const handleBlur = () => {
    requestAnimationFrame(() => {
      if (container && !container.contains(document.activeElement)) onclose()
    })
  }
</script>

<div
  bind:this={container}
  tabindex="-1"
  role="menu"
  onfocusout={handleBlur}
  class="bg-content1 absolute top-12 right-0 z-50 flex w-32 flex-col gap-1 border p-2 shadow-lg"
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
      class="px-3 py-1.5 text-left text-sm transition {selected
        ? 'bg-primary text-white'
        : 'hover:bg-content2'}"
    >
      {LOCALE_NATIVE_NAMES[code]}
    </button>
  {/each}
</div>
