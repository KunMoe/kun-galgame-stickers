<script lang="ts">
  import Icon from '@iconify/svelte'
  import { m, localizedPath } from '$lib/i18n'
  import { locale } from '$lib/locale.svelte'
  import type { PageData } from './$types'

  interface Props {
    data: PageData
  }
  const { data }: Props = $props()
</script>

<section class="flex flex-col gap-6">
  {#each data.packs as pack (pack.sid)}
    <article
      class="group relative flex h-20 items-center overflow-hidden rounded-md border border-border bg-surface px-6 shadow-glow-sm transition hover:shadow-glow"
    >
      <img
        src="/stickers/KUNgal{pack.sid}/{pack.previewPid}.webp"
        alt=""
        width="64"
        height="64"
        loading="lazy"
        class="absolute inset-y-0 left-0 h-20 w-20 object-cover"
      />

      <a
        href={localizedPath(locale.current, `/sticker/${pack.sid}`)}
        aria-label="{m().home.sticker} [{pack.sid}]"
        class="absolute inset-0 flex items-center pl-24 text-base text-primary focus:outline-2 focus:outline-primary"
      >
        <span>{m().home.sticker} [{pack.sid}]</span>
      </a>

      <a
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Download sticker pack {pack.sid}"
        class="relative z-10 ml-auto flex h-10 w-10 items-center justify-center rounded text-2xl text-primary transition hover:bg-accent"
      >
        <Icon icon="line-md:download-outline" />
      </a>
    </article>
  {/each}
</section>
