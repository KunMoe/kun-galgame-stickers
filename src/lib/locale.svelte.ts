import { browser } from '$app/environment'
import { goto } from '$app/navigation'
import { page } from '$app/state'
import { DEFAULT_LOCALE, type Locale } from '$lib/i18n/types'
import { detectLocaleFromPath, localizedPath } from '$lib/i18n'

class LocaleState {
  current: Locale = $state(DEFAULT_LOCALE)

  init(initial: Locale) {
    this.current = initial
  }

  async switchTo(next: Locale) {
    if (!browser) return
    const { pathname: basePath } = detectLocaleFromPath(page.url.pathname)
    const target = localizedPath(next, basePath) + page.url.search
    this.current = next
    await goto(target, { invalidateAll: true })
  }
}

export const locale = new LocaleState()
