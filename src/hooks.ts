import type { Reroute } from '@sveltejs/kit'
import { detectLocaleFromPath } from '$lib/i18n'

export const reroute: Reroute = ({ url }) => {
  const { pathname } = detectLocaleFromPath(url.pathname)
  return pathname
}
