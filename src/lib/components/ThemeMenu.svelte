<script lang="ts">
  import Icon from '@iconify/svelte'
  import { onMount } from 'svelte'
  import { getColorSchemeContext } from '$lib/contexts/theme'
  import { isShowThemeMenu } from '../store/menuStore'
  import { t } from '../language'
  import type { ThemeItem } from '~/types/menu'

  let menuContainer: HTMLElement
  const colorSchemeStore = getColorSchemeContext()

  let kunThemeName: ThemeItem[] = [
    {
      icon: 'line-md:moon-filled-alt-to-sunny-filled-loop-transition',
      name: 'light',
      selected: false
    },
    {
      icon: 'line-md:sunny-outline-to-moon-loop-transition',
      name: 'dark',
      selected: false
    },
    {
      icon: 'line-md:computer',
      name: 'system',
      selected: false
    }
  ]

  kunThemeName = kunThemeName.map((item) => ({
    ...item,
    selected: item.name === $colorSchemeStore
  }))

  const handleChangeTheme = (theme: App.KunTheme) => {
    menuContainer.focus()
    colorSchemeStore.change(theme)
    kunThemeName = kunThemeName.map((item) => ({
      ...item,
      selected: item.name === $colorSchemeStore
    }))
  }

  onMount(() => menuContainer.focus())

  const handleMenuBlur = () => {
    requestAnimationFrame(() => {
      if (!menuContainer.contains(document.activeElement)) {
        isShowThemeMenu.set(false)
      }
    })
  }
</script>

<button bind:this={menuContainer} class="container" on:blur={handleMenuBlur}>
  <span class="triangle1"></span>
  <span class="triangle2"></span>
  <div class="kungalgamer">
    {#each kunThemeName ?? [] as item}
      <button
        class={`item ${item.selected ? 'selected' : ''}`}
        on:click={() => handleChangeTheme(item.name)}
      >
        {#if item.icon}
          <Icon icon={item.icon} />
        {/if}
        <span>{$t(`header.${item.name}`)}</span>
      </button>
    {/each}
  </div>
</button>

<style lang="scss">
  .container {
    position: absolute;
    top: 30px;
    right: 20px;
    opacity: 0.9;
    border: none;
    background-color: var(--kungalgame-blue-4);
  }

  .triangle1 {
    position: absolute;
    top: 1px;
    width: 0;
    height: 0;
    border-left: 10px solid transparent;
    border-right: 10px solid transparent;
    border-bottom: 17px solid var(--kungalgame-white);
    z-index: 1;
  }

  .triangle2 {
    position: absolute;
    width: 0;
    height: 0;
    border-left: 10px solid transparent;
    border-right: 10px solid transparent;
    border-bottom: 17px solid var(--kungalgame-blue-1);
  }

  .kungalgamer {
    padding: 10px;
    top: 16px;
    transform: translateX(-43%);
    width: 130px;
    background-color: var(--kungalgame-white);
    border: 1px solid var(--kungalgame-blue-1);
    position: absolute;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow);
  }

  .item {
    cursor: pointer;
    color: var(--kungalgame-blue-5);
    height: 30px;
    display: flex;
    justify-content: center;
    align-items: center;
    background-color: var(--kungalgame-trans-white-9);
    border: none;
    font-size: 17px;

    span {
      margin-left: 10px;
    }

    &:hover {
      &:not(.selected) {
        background-color: var(--kungalgame-trans-blue-1);
      }
    }
  }

  .selected {
    background-color: var(--kungalgame-blue-4);
    color: var(--kungalgame-white);
  }
</style>
