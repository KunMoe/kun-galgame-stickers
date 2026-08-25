import { generateCodeChallenge, generateCodeVerifier, generateState, isSafeReturnTo } from './oauth-pkce'

const cookieOpts = {
  maxAge: 60 * 10,
  sameSite: 'lax' as const,
  secure: import.meta.client ? location.protocol === 'https:' : false,
  path: '/'
}

const buildAuthorizeUrl = async (returnTo?: string): Promise<string> => {
  const config = useRuntimeConfig()
  const codeVerifier = generateCodeVerifier()
  const codeChallenge = await generateCodeChallenge(codeVerifier)
  const state = generateState()

  const verifier = useCookie('oauth_code_verifier', cookieOpts)
  const oauthState = useCookie('oauth_state', cookieOpts)
  const storedReturn = useCookie('oauth_return_to', cookieOpts)
  verifier.value = codeVerifier
  oauthState.value = state
  storedReturn.value = returnTo && isSafeReturnTo(returnTo) ? returnTo : '/'

  const params = new URLSearchParams({
    client_id: String(config.public.oauthClientId),
    redirect_uri: String(config.public.oauthRedirectUri),
    response_type: 'code',
    scope: 'openid profile email',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256'
  })
  return `${config.public.oauthServerUrl}/oauth/authorize?${params}`
}

export const startOAuthLogin = async (returnTo?: string): Promise<void> => {
  window.location.href = await buildAuthorizeUrl(returnTo)
}

export const startOAuthRegister = async (returnTo?: string): Promise<void> => {
  const config = useRuntimeConfig()
  const authorizeUrl = await buildAuthorizeUrl(returnTo)
  window.location.href = `${config.public.oauthFrontendUrl}/auth/register?redirect=${encodeURIComponent(authorizeUrl)}`
}

export const startOAuthLogout = (): void => {
  const config = useRuntimeConfig()
  const params = new URLSearchParams({
    client_id: String(config.public.oauthClientId),
    redirect: `${window.location.origin}/`
  })
  window.location.href = `${config.public.oauthServerUrl}/oauth/logout?${params}`
}

export const consumeOAuthReturnTo = (): string => {
  const stored = useCookie('oauth_return_to')
  const value = stored.value
  stored.value = null
  if (typeof value === 'string' && isSafeReturnTo(value)) return value
  return '/'
}
