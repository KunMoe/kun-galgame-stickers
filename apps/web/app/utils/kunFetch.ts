interface KunApiResponse<T> {
  code: number
  message: string
  data: T
}

const SSR_FORWARDED = ['kun_oauth_access', 'kun_oauth_refresh', 'kun_oauth_user']

const extractForwardedCookies = (cookieHeader?: string): string | undefined => {
  if (!cookieHeader) return undefined
  const kept: string[] = []
  for (const part of cookieHeader.split(';')) {
    const trimmed = part.trim()
    if (SSR_FORWARDED.some((name) => trimmed.startsWith(`${name}=`))) kept.push(trimmed)
  }
  return kept.length > 0 ? kept.join('; ') : undefined
}

const apiBase = (): string => {
  const config = useRuntimeConfig()
  const root = import.meta.server ? config.apiBaseUrl : config.public.apiBaseUrl
  return `${root}/api/v1`
}

export const kunFetch = async <T>(
  path: string,
  options?: Record<string, unknown>
): Promise<T | null> => {
  const headers = new Headers((options as { headers?: HeadersInit } | undefined)?.headers)
  if (import.meta.server) {
    const forwarded = extractForwardedCookies(useRequestHeaders(['cookie']).cookie)
    if (forwarded) headers.set('cookie', forwarded)
  }

  try {
    const resp = await $fetch<KunApiResponse<T>>(`${apiBase()}${path}`, {
      credentials: 'include',
      ...options,
      headers
    })
    if (!resp || resp.code !== 0) return null
    return resp.data
  } catch {
    return null
  }
}
