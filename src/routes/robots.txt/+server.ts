import type { RequestHandler } from './$types'
import { SITE_URL } from '$lib/config'

export const prerender = true

export const GET: RequestHandler = () =>
  new Response(
    `User-agent: *
Allow: /

Sitemap: ${SITE_URL}/sitemap.xml
`,
    { headers: { 'content-type': 'text/plain' } }
  )
