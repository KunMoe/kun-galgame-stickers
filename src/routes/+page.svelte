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
      class="bg-content1 hover:border-primary relative flex h-20 items-center overflow-hidden border px-6 shadow-sm transition hover:shadow-md"
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
        class="text-primary hover:text-primary-600 focus-visible:outline-primary absolute inset-0 flex items-center pl-24 text-base focus-visible:outline-2"
      >
        <span>{m().home.sticker} [{pack.sid}]</span>
      </a>

      <a
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Download sticker pack {pack.sid}"
        class="text-primary hover:bg-content2 hover:text-primary-600 relative z-10 ml-auto flex h-10 w-10 items-center justify-center text-2xl transition"
      >
        <Icon icon="line-md:download-outline" />
      </a>
    </article>
  {/each}
</section>
