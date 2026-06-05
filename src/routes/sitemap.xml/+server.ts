import type { RequestHandler } from './$types'
import { LOCALES } from '$lib/i18n/types'
import { localizedPath } from '$lib/i18n'
import { SITE_URL, STICKER_PACK_COUNT } from '$lib/config'
import { findStickerPack } from '$lib/server/stickers'

// Generated at runtime (not prerendered): it queries the DB, and runtime-only
// secrets ($env/dynamic/private) can't be read during build-time prerendering.
// Crawler-facing only; the 1h cache-control header keeps DB load negligible.

export const GET: RequestHandler = async () => {
  const packs = await Promise.all(
    Array.from({ length: STICKER_PACK_COUNT }, (_, i) => findStickerPack(i + 1).catch(() => []))
  )

  const paths = [
    '/',
    '/about',
    ...Array.from({ length: STICKER_PACK_COUNT }, (_, i) => `/sticker/${i + 1}`),
    ...packs.flatMap((pack) => pack.map((s) => `/sticker/${s.sid}-${s.pid}`))
  ]

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${paths
  .map((path) => {
    const alternates = LOCALES.map(
      (locale) =>
        `    <xhtml:link rel="alternate" hreflang="${locale}" href="${SITE_URL}${localizedPath(locale, path)}" />`
    ).join('\n')
    return `  <url>
    <loc>${SITE_URL}${path}</loc>
${alternates}
  </url>`
  })
  .join('\n')}
</urlset>`

  return new Response(xml, {
    headers: {
      'content-type': 'application/xml',
      'cache-control': 'public, max-age=3600'
    }
  })
}
