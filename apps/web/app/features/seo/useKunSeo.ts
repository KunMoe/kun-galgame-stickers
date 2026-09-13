/**
 * A JSON-LD node. Typed locally rather than pulling in schema-dts: this site
 * emits three shapes and a types-only dependency is not worth carrying for it.
 */
export interface JsonLd extends Record<string, unknown> {
  '@context': 'https://schema.org'
  '@type': string
}

export interface KunSeoInput {
  title: string
  description: string
  /** Share image: an absolute URL, or a site-relative path. */
  image?: string
  /** og:type — 'website' for listings, 'article' for a single thing. */
  type?: 'website' | 'article' | 'profile'
  /** Keep a thin or private page out of the index. */
  noindex?: boolean
  /** JSON-LD for the page's subject, emitted verbatim. */
  jsonLd?: JsonLd
}

/**
 * One place that emits a page's whole metadata block.
 *
 * Pages used to set a title, a description and sometimes an og:image, which
 * left og:url, og:type, og:site_name, the Twitter card and -- on a site that
 * publishes the same page in three languages -- every hreflang alternate
 * unstated. Search engines then pick a canonical themselves, and the three
 * locales compete with each other.
 *
 * Canonical is the path without its query string: ?page=2 and ?work=4 are
 * views of one document, not documents of their own.
 */
export const useKunSeo = (input: MaybeRefOrGetter<KunSeoInput>) => {
  const config = useRuntimeConfig()
  const route = useRoute()
  const { locale, locales, t } = useI18n()
  const switchLocalePath = useSwitchLocalePath()

  const site = config.public.siteUrl.replace(/\/$/, '')
  const resolved = computed(() => toValue(input))
  const canonical = computed(() => `${site}${route.path}`)
  const image = computed(() => {
    const value = resolved.value.image
    if (!value) return `${site}/title.webp`
    return value.startsWith('/') ? `${site}${value}` : value
  })

  // og:locale wants a POSIX-ish tag; the router keeps the short codes.
  const ogLocale = computed(() => {
    const found = (locales.value as { code: string; language?: string }[]).find(
      (item) => item.code === locale.value
    )
    return (found?.language ?? 'zh-CN').replace('-', '_')
  })

  // The locale list is static config, so the link set is built once and only
  // the hrefs stay reactive -- which is also the shape useHead types cleanly.
  const localeList = locales.value as { code: string; language?: string }[]

  useHead({
    link: [
      { rel: 'canonical', href: () => canonical.value },
      ...localeList.map((item) => ({
        rel: 'alternate' as const,
        hreflang: item.language ?? item.code,
        href: () => `${site}${switchLocalePath(item.code as 'zh-cn' | 'en' | 'ja')}`
      })),
      {
        rel: 'alternate' as const,
        hreflang: 'x-default',
        href: () => `${site}${switchLocalePath('zh-cn')}`
      }
    ],
    script: () =>
      resolved.value.jsonLd
        ? [{ type: 'application/ld+json', innerHTML: JSON.stringify(resolved.value.jsonLd) }]
        : []
  })

  useSeoMeta({
    title: () => resolved.value.title,
    description: () => resolved.value.description,
    ogTitle: () => resolved.value.title,
    ogDescription: () => resolved.value.description,
    ogType: () => resolved.value.type ?? 'website',
    ogUrl: () => canonical.value,
    ogSiteName: () => t('meta.title'),
    ogLocale: () => ogLocale.value,
    ogImage: () => image.value,
    ogImageWidth: 1200,
    ogImageHeight: 630,
    ogImageAlt: () => resolved.value.title,
    twitterCard: 'summary_large_image',
    twitterTitle: () => resolved.value.title,
    twitterDescription: () => resolved.value.description,
    twitterImage: () => image.value,
    robots: () => (resolved.value.noindex ? 'noindex, follow' : 'index, follow')
  })
}

/**
 * Path of the share image a Nitro route renders for this subject. Deliberately
 * relative and composable-free: it is called from inside useKunSeo's lazy
 * getter, which runs when the head resolves rather than during setup, and
 * reaching for useRuntimeConfig() there throws NUXT_E1001. useKunSeo makes it
 * absolute against the site URL it captured at setup.
 */
export const kunOgImage = (
  path: string,
  locale: string,
  params: Record<string, string> = {}
): string => {
  const query = new URLSearchParams({ locale })
  for (const [key, value] of Object.entries(params)) if (value) query.set(key, value)
  return `/og/${path}?${query}`
}
