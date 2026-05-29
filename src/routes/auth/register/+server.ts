import { redirect } from '@sveltejs/kit'
import type { RequestHandler } from './$types'
import { isSafeReturnTo, persistPkce } from '$lib/server/auth'
import { buildAuthorizeUrl, buildRegisterUrl } from '$lib/server/oauth'
import { generateCodeChallenge, generateCodeVerifier, generateState } from '$lib/server/pkce'

/**
 * Unified registration entry point. Identical PKCE setup to /auth/login, but
 * instead of going straight to /oauth/authorize we send the user to the OAuth
 * web frontend's register page, carrying the authorize URL as `redirect`.
 * Registration is "register === login": once done, OAuth forwards to authorize
 * and a code lands back on our /auth/callback — which needs no special casing.
 */
export const GET: RequestHandler = async ({ url, cookies }) => {
  const requested = url.searchParams.get('return_to') ?? '/'
  const returnTo = isSafeReturnTo(requested) ? requested : '/'

  const codeVerifier = generateCodeVerifier()
  const codeChallenge = await generateCodeChallenge(codeVerifier)
  const state = generateState()

  persistPkce(cookies, { state, codeVerifier, returnTo })

  redirect(302, buildRegisterUrl(buildAuthorizeUrl(state, codeChallenge)))
}
