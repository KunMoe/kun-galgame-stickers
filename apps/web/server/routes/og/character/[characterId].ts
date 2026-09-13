import { resolveMultilingual } from '~/features/pack/types'

interface Envelope<T> {
  code: number
  data?: T
}

interface CharacterShape {
  character: {
    name: Record<string, string | undefined>
    image_url: string
  }
  appearances?: { name: Record<string, string | undefined> }[]
  stickers?: unknown[]
}

/**
 * The share image for a character. The `character` template draws the portrait
 * with `contain`, which is what these need -- catalog serves a 250x300 bust,
 * and anything that crops to a square takes the top off someone's head.
 */
export default defineCachedEventHandler(
  async (event) => {
    const characterId = getRouterParam(event, 'characterId')
    const config = useRuntimeConfig()
    const fallback = `${config.public.siteUrl}/title.webp`

    const page = await $fetch<Envelope<CharacterShape>>(
      `${config.apiBaseUrl}/api/v1/characters/${characterId}`,
      { timeout: 8000 }
    ).catch(() => null)

    if (!page || page.code !== 0 || !page.data) {
      return sendRedirect(event, fallback, 302)
    }
    const { character, appearances, stickers } = page.data
    const locale = String(getQuery(event).locale ?? 'zh-cn')

    // Most catalog characters carry only their Japanese name, which is what
    // resolveMultilingual then returns for every locale -- printing it again as
    // the subtitle put the same string on the card twice.
    const name = resolveMultilingual(character.name, locale)
    const original = character.name['und']

    const url =
      buildOgUrl('character', {
        name: ogText(name, 120) ?? 'Character',
        originalName: ogText(original === name ? '' : original, 120),
        portrait: character.image_url || undefined,
        // The work goes in the footer, not in the template's `work` field:
        // that field clamps the name above it to a single line, and these names
        // are long enough that a card about a character cut hers in half.
        voice: ogText(
          appearances?.[0] ? resolveMultilingual(appearances[0].name, locale) : '',
          80
        ),
        badges: stickers?.length ? [`${stickers.length} 张贴纸`] : []
      }) ?? character.image_url ?? fallback

    return sendRedirect(event, url, 302)
  },
  {
    name: 'og-character',
    getKey: (event) =>
      `${getRouterParam(event, 'characterId')}:${getQuery(event).locale ?? 'zh-cn'}`,
    swr: true,
    maxAge: 60 * 60,
    staleMaxAge: 60 * 60 * 24
  }
)
