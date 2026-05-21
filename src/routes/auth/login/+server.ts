import { redirect } from '@sveltejs/kit'
import type { RequestHandler } from './$types'
import { persistPkce } from '$lib/server/auth'
import { buildAuthorizeUrl } from '$lib/server/oauth'
import { generateCodeChallenge, generateCodeVerifier, generateState } from '$lib/server/pkce'

const isSafeReturnTo = (value: string): boolean =>
  value.startsWith('/') && !value.startsWith('//') && !value.startsWith('/\\')

export const GET: RequestHandler = async ({ url, cookies }) => {
  const requested = url.searchParams.get('return_to') ?? '/'
  const returnTo = isSafeReturnTo(requested) ? requested : '/'

  const codeVerifier = generateCodeVerifier()
  const codeChallenge = await generateCodeChallenge(codeVerifier)
  const state = generateState()

  persistPkce(cookies, { state, codeVerifier, returnTo })

  redirect(302, buildAuthorizeUrl(state, codeChallenge))
}
