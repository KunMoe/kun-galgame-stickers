interface Pack {
  sid: number
  count: number
}

interface Envelope<T> {
  code: number
  data?: T
}

interface SitemapEntry {
  loc: string
  changefreq: 'weekly'
  priority: number
  _i18nTransform: true
}

const entry = (loc: string, priority: number): SitemapEntry => ({
  loc,
  changefreq: 'weekly',
  priority,
  _i18nTransform: true
})

export default defineCachedEventHandler(
  async () => {
    const urls: SitemapEntry[] = [entry('/', 1), entry('/about', 0.8)]
    const apiBase = useRuntimeConfig().apiBaseUrl

    try {
      const resp = await $fetch<Envelope<Pack[]>>(`${apiBase}/api/v1/sticker/packs`, {
        timeout: 15000
      })
      const packs = resp?.code === 0 ? (resp.data ?? []) : []
      for (const pack of packs) {
        urls.push(entry(`/sticker/${pack.sid}`, 0.7))
        for (let pid = 1; pid <= pack.count; pid++) {
          urls.push(entry(`/sticker/${pack.sid}-${pid}`, 0.5))
        }
      }
    } catch {
      // Static pages still ship from this handler; pack URLs retry on the next cache miss.
    }

    return urls
  },
  {
    name: 'sitemap-urls',
    getKey: () => 'all',
    swr: true,
    maxAge: 60 * 60 * 6,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
