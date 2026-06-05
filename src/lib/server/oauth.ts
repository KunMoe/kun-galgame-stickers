import { env } from '$env/dynamic/private'

export interface OAuthTokens {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  scope?: string
}

export interface OAuthUser {
  sub: string
  id?: number
  name?: string
  email?: string
  picture?: string
  roles?: string[]
  updated_at?: number
}

interface KunResponse<T> {
  code: number
  message: string
  data: T
}

export class OAuthError extends Error {
  constructor(
    public readonly code: number,
    message: string
  ) {
    super(message)
    this.name = 'OAuthError'
  }
}

const postJson = async <T>(path: string, body: Record<string, unknown>): Promise<T> => {
  const res = await fetch(`${env.KUN_OAUTH_SERVER_URL}${path}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body)
  })
  const json = (await res.json().catch(() => null)) as KunResponse<T> | null
  if (!json) throw new OAuthError(-1, `OAuth server returned non-JSON (HTTP ${res.status})`)
  if (json.code !== 0) throw new OAuthError(json.code, json.message)
  return json.data
}

export const exchangeCodeForTokens = (code: string, codeVerifier: string): Promise<OAuthTokens> =>
  postJson<OAuthTokens>('/oauth/token', {
    grant_type: 'authorization_code',
    code,
    redirect_uri: env.KUN_OAUTH_REDIRECT_URI,
    client_id: env.KUN_OAUTH_CLIENT_ID,
    client_secret: env.KUN_OAUTH_CLIENT_SECRET,
    code_verifier: codeVerifier
  })

export const refreshOAuthTokens = (refreshToken: string): Promise<OAuthTokens> =>
  postJson<OAuthTokens>('/oauth/token', {
    grant_type: 'refresh_token',
    refresh_token: refreshToken,
    client_id: env.KUN_OAUTH_CLIENT_ID,
    client_secret: env.KUN_OAUTH_CLIENT_SECRET
  })

export const revokeOAuthToken = async (token: string): Promise<void> => {
  // RFC 7009: always 200; treat any thrown error as best-effort failure.
  await postJson<unknown>('/oauth/revoke', { token }).catch(() => undefined)
}

export const fetchOAuthUser = async (accessToken: string): Promise<OAuthUser> => {
  const res = await fetch(`${env.KUN_OAUTH_SERVER_URL}/oauth/userinfo`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  })
  const json = (await res.json().catch(() => null)) as KunResponse<OAuthUser> | null
  if (!json) throw new OAuthError(-1, `userinfo returned non-JSON (HTTP ${res.status})`)
  if (json.code !== 0) throw new OAuthError(json.code, json.message)
  return json.data
}

export const buildAuthorizeUrl = (
  state: string,
  codeChallenge: string,
  scope = 'openid profile email'
): string => {
  const params = new URLSearchParams({
    client_id: env.KUN_OAUTH_CLIENT_ID,
    redirect_uri: env.KUN_OAUTH_REDIRECT_URI,
    response_type: 'code',
    scope,
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256'
  })
  return `${env.KUN_OAUTH_SERVER_URL}/oauth/authorize?${params.toString()}`
}

/**
 * Build the unified-registration URL. Per the OAuth spec, downstream sites
 * don't host their own register form: they send the user to the OAuth web
 * frontend's `/auth/register`, passing the fully-built `/oauth/authorize`
 * URL as `redirect`. After the user registers (register === login), OAuth
 * forwards to that authorize URL, which — because this client is first-party
 * (auto_consent) — issues a code straight back to our `/auth/callback`.
 */
export const buildRegisterUrl = (authorizeUrl: string): string => {
  const base = env.KUN_OAUTH_WEB_URL.replace(/\/$/, '')
  return `${base}/auth/register?redirect=${encodeURIComponent(authorizeUrl)}`
}
