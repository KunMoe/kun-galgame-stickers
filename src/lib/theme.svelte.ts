import { browser } from '$app/environment'
import { PUBLIC_KUN_STICKER_THEME } from '$env/static/public'
import type { Theme } from '$lib/i18n/types'

class ThemeState {
  current: Theme = $state('system')

  init(initial: Theme) {
    this.current = initial
  }

  set(next: Theme) {
    this.current = next
    if (!browser) return
    document.documentElement.dataset.colorScheme = next
    document.cookie = `${PUBLIC_KUN_STICKER_THEME}=${next}; path=/; max-age=${60 * 60 * 24 * 365}; samesite=lax`
  }
}

export const theme = new ThemeState()
