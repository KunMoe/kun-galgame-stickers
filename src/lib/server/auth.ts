import type { Cookies } from '@sveltejs/kit'
import { dev } from '$app/environment'
import { fetchOAuthUser, refreshOAuthTokens, type OAuthTokens, type OAuthUser } from './oauth'

export const COOKIES = {
  access: 'kun_oauth_access',
  refresh: 'kun_oauth_refresh',
  user: 'kun_oauth_user',
  pkce: 'kun_oauth_pkce'
} as const

/** Guard open-redirect: only allow same-origin absolute paths as return targets. */
export const isSafeReturnTo = (value: string): boolean =>
  value.startsWith('/') && !value.startsWith('//') && !value.startsWith('/\\')

const REFRESH_TTL_SEC = 60 * 60 * 24 * 7 // 7 days — matches OAuth refresh_token TTL
const PKCE_TTL_SEC = 60 * 10 // 10 minutes — covers the OAuth round-trip
const ACCESS_SAFETY_WINDOW_SEC = 30

const baseCookie = () => ({
  path: '/' as const,
  httpOnly: true,
  secure: !dev,
  sameSite: 'lax' as const
})

interface JwtClaims {
  sub: string
  name?: string
  email?: string
  roles?: string[]
  exp: number
  iat: number
}

export const decodeJwt = <T>(token: string): T | null => {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = payload + '=='.slice(0, (4 - (payload.length % 4)) % 4)
    return JSON.parse(atob(padded)) as T
  } catch {
    return null
  }
}

export const persistTokens = (cookies: Cookies, tokens: OAuthTokens) => {
  cookies.set(COOKIES.access, tokens.access_token, {
    ...baseCookie(),
    maxAge: tokens.expires_in
  })
  cookies.set(COOKIES.refresh, tokens.refresh_token, {
    ...baseCookie(),
    maxAge: REFRESH_TTL_SEC
  })
}

export const persistUser = (cookies: Cookies, user: OAuthUser, maxAge: number) => {
  cookies.set(COOKIES.user, JSON.stringify(user), {
    ...baseCookie(),
    maxAge
  })
}

export const clearSession = (cookies: Cookies) => {
  cookies.delete(COOKIES.access, { path: '/' })
  cookies.delete(COOKIES.refresh, { path: '/' })
  cookies.delete(COOKIES.user, { path: '/' })
}

export interface PkceState {
  state: string
  codeVerifier: string
  returnTo: string
}

export const persistPkce = (cookies: Cookies, value: PkceState) => {
  cookies.set(COOKIES.pkce, JSON.stringify(value), {
    ...baseCookie(),
    maxAge: PKCE_TTL_SEC
  })
}

export const readPkce = (cookies: Cookies): PkceState | null => {
  const raw = cookies.get(COOKIES.pkce)
  if (!raw) return null
  try {
    return JSON.parse(raw) as PkceState
  } catch {
    return null
  }
}

export const clearPkce = (cookies: Cookies) => {
  cookies.delete(COOKIES.pkce, { path: '/' })
}

/**
 * Resolve the current user from cookies, transparently refreshing the
 * access token via the refresh token when needed. Side-effects: writes
 * refreshed tokens + cached user info back into cookies.
 *
 * Returns `null` when there is no session, or when the refresh token has
 * been rejected by the OAuth server (in which case all session cookies
 * are cleared so the user goes back to the logged-out state).
 */
export const resolveUser = async (cookies: Cookies): Promise<OAuthUser | null> => {
  const accessToken = cookies.get(COOKIES.access)
  const now = Math.floor(Date.now() / 1000)

  if (accessToken) {
    const claims = decodeJwt<JwtClaims>(accessToken)
    if (claims && claims.exp > now + ACCESS_SAFETY_WINDOW_SEC) {
      const cached = cookies.get(COOKIES.user)
      if (cached) {
        try {
          return JSON.parse(cached) as OAuthUser
        } catch {
          // fall through to a fresh userinfo fetch
        }
      }
      try {
        const user = await fetchOAuthUser(accessToken)
        persistUser(cookies, user, claims.exp - now)
        return user
      } catch {
        return null
      }
    }
  }

  const refreshToken = cookies.get(COOKIES.refresh)
  if (!refreshToken) {
    // access token missing/expired and no refresh — make sure the stale
    // user cookie doesn't survive a real logout.
    if (cookies.get(COOKIES.user)) clearSession(cookies)
    return null
  }

  try {
    const tokens = await refreshOAuthTokens(refreshToken)
    persistTokens(cookies, tokens)
    const user = await fetchOAuthUser(tokens.access_token)
    persistUser(cookies, user, tokens.expires_in)
    return user
  } catch {
    clearSession(cookies)
    return null
  }
}
