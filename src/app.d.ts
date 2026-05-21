import type { OAuthUser } from '$lib/server/oauth'
import type { Locale, Theme } from '$lib/i18n/types'

declare global {
  namespace App {
    interface Locals {
      lang: Locale
      theme: Theme
      user: OAuthUser | null
    }
    interface PageData {
      lang: Locale
      theme: Theme
      user: OAuthUser | null
    }
  }
}

export {}
