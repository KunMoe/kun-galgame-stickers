import type { Theme } from '$lib/i18n/types'

class ThemeState {
  current: Theme = $state('system')
}

export const theme = new ThemeState()

export const isDarkResolved = (t: Theme): boolean => {
  if (t === 'dark') return true
  if (t === 'light') return false
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
}
