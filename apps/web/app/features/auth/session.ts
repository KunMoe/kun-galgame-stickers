import type { Cosmetics } from '~/features/user/cosmetics'

export interface AuthUser {
  sub: string
  id: number
  name: string
  picture: string
  roles: string[]
  cosmetics?: Cosmetics
}

export const useAuthUser = () => useState<AuthUser | null>('auth-user', () => null)

export const fetchMe = (): Promise<AuthUser | null> => kunFetchOrNull<AuthUser>('/auth/me')

export const exchangeOAuthCode = (code: string, codeVerifier: string): Promise<AuthUser | null> =>
  kunFetchOrNull<AuthUser>('/auth/oauth/callback', {
    method: 'POST',
    body: { code, code_verifier: codeVerifier }
  })

export const logoutLocal = (): Promise<unknown> =>
  kunFetchOrNull('/auth/logout', { method: 'POST' })
