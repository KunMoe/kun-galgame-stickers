import { error, redirect } from '@sveltejs/kit'
import type { RequestHandler } from './$types'
import { clearPkce, persistTokens, persistUser, readPkce } from '$lib/server/auth'
import { exchangeCodeForTokens, fetchOAuthUser, OAuthError } from '$lib/server/oauth'

export const GET: RequestHandler = async ({ url, cookies }) => {
  const oauthError = url.searchParams.get('error')
  if (oauthError) {
    clearPkce(cookies)
    error(400, `OAuth provider returned: ${oauthError}`)
  }

  const code = url.searchParams.get('code')
  const state = url.searchParams.get('state')
  const pkce = readPkce(cookies)
  clearPkce(cookies)

  if (!code || !state || !pkce) error(400, 'Missing OAuth callback parameters')
  if (state !== pkce.state) error(400, 'State mismatch — possible CSRF')

  try {
    const tokens = await exchangeCodeForTokens(code, pkce.codeVerifier)
    persistTokens(cookies, tokens)
    const user = await fetchOAuthUser(tokens.access_token)
    persistUser(cookies, user, tokens.expires_in)
  } catch (e) {
    if (e instanceof OAuthError) error(400, `OAuth exchange failed (${e.code}): ${e.message}`)
    throw e
  }

  redirect(303, pkce.returnTo)
}
