import { redirect } from '@sveltejs/kit'
import type { RequestHandler } from './$types'
import { COOKIES, clearSession } from '$lib/server/auth'
import { revokeOAuthToken } from '$lib/server/oauth'

const isSafeReturnTo = (value: string): boolean =>
  value.startsWith('/') && !value.startsWith('//') && !value.startsWith('/\\')

export const POST: RequestHandler = async ({ request, cookies }) => {
  const refreshToken = cookies.get(COOKIES.refresh)
  if (refreshToken) await revokeOAuthToken(refreshToken)
  clearSession(cookies)

  const form = await request.formData().catch(() => null)
  const requested = form?.get('return_to')
  const returnTo =
    typeof requested === 'string' && isSafeReturnTo(requested) ? requested : '/'

  redirect(303, returnTo)
}
