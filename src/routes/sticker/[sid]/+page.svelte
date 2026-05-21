<script lang="ts">
  import Icon from '@iconify/svelte'
  import { m, resolveMultilingual, localizedPath } from '$lib/i18n'
  import { locale } from '$lib/locale.svelte'
  import type { PageData } from './$types'

  interface Props {
    data: PageData
  }
  const { data }: Props = $props()

  const downloadImage = async (imagePath: string) => {
    const response = await fetch(imagePath)
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = imagePath.split('/').pop() ?? 'sticker.png'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }
</script>

<svelte:head>
  <title>{m().sticker.title} [{data.sid}]</title>
  <meta name="description" content={m().sticker.description} />
</svelte:head>

<div class="grid grid-cols-2 gap-5 sm:grid-cols-3 md:grid-cols-4">
  {#each data.stickers as sticker (sticker.pid)}
    <article
      class="relative flex flex-col gap-2 overflow-hidden border bg-content1 p-3 shadow-sm transition hover:border-primary hover:shadow-md"
    >
      <div class="aspect-square w-full overflow-hidden">
        <img
          src="/stickers/KUNgal{data.sid}/{sticker.pid}.webp"
          alt=""
          width="320"
          height="320"
          loading="lazy"
          class="h-full w-full object-cover"
        />
      </div>

      <span
        aria-hidden="true"
        class="pointer-events-none absolute right-3 bottom-12 -z-10 font-serif text-6xl font-bold italic text-primary-100"
      >
        {data.sid}-{sticker.pid}
      </span>

      <div class="text-sm text-default-600">
        <p class="truncate">{m().sticker.game}: {resolveMultilingual(sticker.game, locale.current)}</p>
        <p class="truncate">{m().sticker.lass}: {resolveMultilingual(sticker.loli, locale.current)}</p>
      </div>

      <div class="mt-1 flex items-center justify-between">
        <a
          href={localizedPath(locale.current, `/sticker/${data.sid}-${sticker.pid}`)}
          class="border border-primary px-3 py-1 text-sm text-primary transition hover:bg-primary hover:text-white"
        >
          {m().sticker.original}
        </a>
        <button
          type="button"
          aria-label={m().sticker.download}
          onclick={() =>
            downloadImage(`/kun-galgame-stickers/telegram/KUNgal${data.sid}/${sticker.pid}.png`)}
          class="flex h-9 w-9 items-center justify-center text-2xl text-primary transition hover:bg-content2 hover:text-primary-600"
        >
          <Icon icon="line-md:download-outline" />
        </button>
      </div>
    </article>
  {/each}
</div>
