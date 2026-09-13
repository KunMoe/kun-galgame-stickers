/**
 * The share image for pages with no single entity behind them -- discovery,
 * tags, author pages, search. Upstream is explicit that these should not be
 * forced into the `work` template.
 *
 * The `logo` field is the card's hero art, bled to 1080x608 behind the text --
 * not an icon slot. Handing it favicon.webp blew a 192px app icon up to the
 * full height of the card; title.webp is the sticker montage that was drawn at
 * exactly that size.
 *
 * Callers pass their own name and slogan, so a tag page and an author page no
 * longer share one anonymous card.
 */
export default defineCachedEventHandler(
  async (event) => {
    const config = useRuntimeConfig()
    const query = getQuery(event)
    const fallback = `${config.public.siteUrl}/title.webp`

    const url =
      buildOgUrl('site', {
        name: ogText(String(query.name ?? '鲲 Galgame 表情包'), 80) ?? '鲲 Galgame 表情包',
        slogan: ogText(query.slogan ? String(query.slogan) : '', 160),
        logo: fallback
      }) ?? fallback

    return sendRedirect(event, url, 302)
  },
  {
    name: 'og-site',
    getKey: (event) => {
      const query = getQuery(event)
      return `${query.name ?? ''}:${query.slogan ?? ''}`
    },
    swr: true,
    maxAge: 60 * 60 * 6,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
