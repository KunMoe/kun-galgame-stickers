<script lang="ts">
  import Icon from '@iconify/svelte'
  import { t } from '~/lib/language'
  import { randomNum } from '~/utils/random'

  const stickerPacks = [1, 2, 3, 4, 5, 6, 7]
  const randomNumber = randomNum(1, 7)
</script>

<section>
  {#each stickerPacks ?? [] as sticker}
    <div class="link">
      <a
        aria-label="KUN Visual Novel Sticker Package | 鲲 Galgame 表情包"
        href={`/sticker/${sticker}`}
      >
        {$t('home.sticker')} [{sticker}]
      </a>
      <img
        src={`stickers/KUNgal${sticker}/${randomNumber}.webp`}
        alt="KUN Visual Novel Stickers | 鲲 Galgame 表情包"
      />
      <a
        aria-label="KUN Visual Novel Sticker Package Download | 鲲 Galgame 贴纸包下载"
        class="icon"
        href="https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases"
        target="_blank"
        on:click|stopPropagation
      >
        <Icon icon="line-md:download-outline" />
      </a>
    </div>
  {/each}
</section>

<style lang="scss">
  .link {
    height: 63px;
    display: flex;
    align-items: center;
    position: relative;
    margin-bottom: 30px;
    overflow: hidden;
    border: 1px solid var(--kungalgame-trans-blue-1);
    box-shadow: var(--kungalgame-shadow-0);
    transition: all 0.2s;

    img {
      position: absolute;
      width: 63px;
      height: 63px;
    }

    a {
      &:first-child {
        padding: 20px 50px;
        margin-left: 30px;
        color: var(--kungalgame-blue-5);
      }

      &:last-child {
        padding: 10px;
        display: flex;
        color: var(--kungalgame-blue-5);
        font-size: 25px;
      }
    }

    &:hover {
      box-shadow: var(--kungalgame-shadow-1);
    }
  }

  @media (max-width: 700px) {
    .link {
      a {
        &:first-child {
          padding-right: 10px;
        }
      }
    }
  }
</style>
