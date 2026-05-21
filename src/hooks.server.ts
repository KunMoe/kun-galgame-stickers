import type { Handle } from '@sveltejs/kit'
import { PUBLIC_KUN_STICKER_THEME } from '$env/static/public'
import { detectLocaleFromPath } from '$lib/i18n'
import { type Theme } from '$lib/i18n/types'

const VALID_THEMES = new Set<Theme>(['light', 'dark', 'system'])
const isTheme = (v: string): v is Theme => VALID_THEMES.has(v as Theme)

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.lang = detectLocaleFromPath(event.url.pathname).locale

  const cookieTheme = event.cookies.get(PUBLIC_KUN_STICKER_THEME)
  event.locals.theme = cookieTheme && isTheme(cookieTheme) ? cookieTheme : 'system'

  return resolve(event, {
    transformPageChunk: ({ html }) =>
      html
        .replace('%kun.lang%', event.locals.lang)
        .replace('%kun.theme%', event.locals.theme)
  })
}
