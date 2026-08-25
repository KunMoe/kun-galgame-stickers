import { kunFetch } from '~/utils/kunFetch'

export interface AuthUser {
  sub: string
  id: number
  name: string
  email: string
  picture: string
  roles: string[]
}

export const useAuthUser = () => useState<AuthUser | null>('auth-user', () => null)

export const fetchMe = (): Promise<AuthUser | null> => kunFetch<AuthUser>('/auth/me')

export const exchangeOAuthCode = (code: string, codeVerifier: string): Promise<AuthUser | null> =>
  kunFetch<AuthUser>('/auth/oauth/callback', {
    method: 'POST',
    body: { code, code_verifier: codeVerifier }
  })

export const logoutLocal = (): Promise<unknown> => kunFetch('/auth/logout', { method: 'POST' })
