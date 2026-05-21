<script lang="ts">
  import { m, resolveMultilingual } from '$lib/i18n'
  import { locale } from '$lib/locale.svelte'
  import type { PageData } from './$types'

  interface Props {
    data: PageData
  }
  const { data }: Props = $props()

  const game = $derived(resolveMultilingual(data.sticker.game, locale.current))
  const lass = $derived(resolveMultilingual(data.sticker.loli, locale.current))
</script>

<svelte:head>
  <title>{game} | {lass}</title>
  <meta name="description" content="{game} | {lass}" />
</svelte:head>

<div class="grid place-items-center gap-4">
  <img
    src="/kun-galgame-stickers/telegram/KUNgal{data.sticker.sid}/{data.sticker.pid}.png"
    alt="{game} {lass}"
    class="max-w-full rounded"
  />

  <section class="flex flex-col items-center gap-1 text-sm">
    <p>{m().sticker.game}: {game}</p>
    <p>{m().sticker.lass}: {lass}</p>
  </section>
</div>
