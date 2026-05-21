import type { Locale, Theme } from '$lib/i18n/types'

declare global {
  namespace App {
    interface Locals {
      lang: Locale
      theme: Theme
    }
    interface PageData {
      lang: Locale
      theme: Theme
    }
  }
}

export {}
