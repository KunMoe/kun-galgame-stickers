<script lang="ts">
  import Icon from '@iconify/svelte'
  import { goto } from '$app/navigation'
  import { t } from '~/lib/language'
  import { getLanguageContext } from '~/lib/contexts/language'
  import { getPreferredLanguageText } from '~/utils/getPreferredLanguageText'

  export let data
  const languageStore = getLanguageContext()

  const downloadImage = async (imagePath: string) => {
    const response = await fetch(imagePath)
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)

    const a = document.createElement('a')
    a.href = url
    a.download = imagePath.split('/').pop() || ''

    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }
</script>

<svelte:head>
  <title>{$t('sticker.title')}</title>
  <meta name="description" content={$t('sticker.description')} />
</svelte:head>

<div class="root">
  <div class="container">
    {#each data.stickersData as sticker}
      <div class="sticker">
        <div class="image-container">
          <img src={`/stickers/KUNgal${data.sid}/${sticker.pid}.webp`} alt="" />
        </div>
        <span class="sequence">{`${data.sid}-${sticker.pid}`}</span>

        <div class="info">
          <p>{$t('sticker.game')}: {getPreferredLanguageText(sticker.game, $languageStore)}</p>
          <p>{$t('sticker.lass')}: {getPreferredLanguageText(sticker.loli, $languageStore)}</p>
        </div>

        <div class="btn">
          <button class="original" on:click={() => goto(`/sticker/${data.sid}-${sticker.pid}`)}>
            {$t('sticker.original')}
          </button>

          <button
            aria-label={$t('sticker.download')}
            class="download"
            on:click={() => {
              downloadImage(`/kun-galgame-stickers/telegram/KUNgal${data.sid}/${sticker.pid}.png`)
            }}
          >
            <Icon icon="line-md:download-outline" />
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>

<style lang="scss">
  .container {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 20px;
  }

  .sticker {
    padding: 10px;
    box-shadow: var(--kungalgame-shadow-0);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    position: relative;
    transition: all 0.2s;

    .image-container {
      width: 100%;
      aspect-ratio: 1 / 1;
      display: flex;
      justify-content: center;
      align-items: center;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    &:hover {
      box-shadow: var(--kungalgame-shadow-1);
    }
  }

  .info {
    font-size: 13px;

    p {
      &:first-child {
        margin: 5px 0;
      }
    }
  }

  .sequence {
    user-select: none;
    position: absolute;
    bottom: 0;
    color: var(--kungalgame-trans-blue-2);
    text-shadow: 2px 2px 3px var(--kungalgame-trans-blue-1);
    font-size: 60px;
    font-style: italic;
    font-family: serif;
    cursor: pointer;
    z-index: -1;
  }

  .btn {
    display: flex;
    justify-content: space-between;
    margin-top: 10px;

    button {
      display: flex;
      justify-content: center;
      background-color: var(--kungalgame-trans-white-9);
      color: var(--kungalgame-blue-5);
      cursor: pointer;

      &:nth-child(1) {
        border: 1px solid var(--kungalgame-blue-5);
        padding: 3px 10px;

        &:hover {
          background-color: var(--kungalgame-blue-5);
          color: var(--kungalgame-white);
        }
      }

      &:nth-child(2) {
        border: 1px solid var(--kungalgame-trans-white-9);
        font-size: 23px;
      }
    }
  }
</style>
