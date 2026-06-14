import { redirect } from '@sveltejs/kit'
import type { RequestHandler } from './$types'
import { COOKIES, clearSession, isSafeReturnTo } from '$lib/server/auth'
import { revokeOAuthToken } from '$lib/server/oauth'
import { env } from '$env/dynamic/private'

export const POST: RequestHandler = async ({ request, cookies }) => {
  const refreshToken = cookies.get(COOKIES.refresh)
  if (refreshToken) await revokeOAuthToken(refreshToken)
  clearSession(cookies)

  const form = await request.formData().catch(() => null)
  const requested = form?.get('return_to')
  const returnTo =
    typeof requested === 'string' && isSafeReturnTo(requested) ? requested : '/'

  // RP-initiated logout. Revoking our refresh_token + clearing our session
  // cookie above only ends OUR session. The central OP (oauth.kungal.com) web
  // session survives — its localStorage user + its own refresh cookie — so the
  // next login would silently re-consent (first-party auto_consent) straight
  // back into the same account. So instead of redirecting home directly, send
  // the browser through the OP logout entrypoint, which clears the OP session
  // and then returns to `redirect`. The OP validates `redirect` against our
  // registered redirect_uris; derive our origin from the configured
  // redirect_uri so it matches the allow-list. See docs/oauth/07-logout.md.
  const ourOrigin = new URL(env.KUN_OAUTH_REDIRECT_URI).origin
  const params = new URLSearchParams({
    client_id: env.KUN_OAUTH_CLIENT_ID,
    redirect: new URL(returnTo, ourOrigin).toString()
  })
  redirect(303, `${env.KUN_OAUTH_SERVER_URL}/oauth/logout?${params.toString()}`)
}
