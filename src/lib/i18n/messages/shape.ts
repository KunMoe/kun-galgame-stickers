export interface Messages {
  meta: {
    title: string
    description: string
    og: string
    ogDescription: string
  }
  header: {
    title: string
    home: string
    about: string
    light: string
    dark: string
    system: string
  }
  home: {
    sticker: string
    kun: string
    open: string
  }
  sticker: {
    title: string
    description: string
    game: string
    lass: string
    introduction: string
    original: string
    download: string
    backToTop: string
    notFound: string
  }
  footer: {
    poweredBy: string
    forumName: string
    forumSuffix: string
  }
  auth: {
    login: string
    logout: string
    account: string
    profile: string
    anonymous: string
  }
  about: {
    title: string
    description: string
    repo: string
    introduction: {
      heading: string
      lines: string[]
      bilibiliPrefix: string
      bilibiliLink: string
      purposeIntro: string
      purpose: string[]
      future: string
    }
    telegram: {
      heading: string
      intro: string
      packLabel: string
      note: string
      downloadIntro: string
      downloadLink: string
    }
    rules: {
      heading: string
      intro: string
      items: string[]
      note: string
    }
    games: {
      heading: string
      lines: string[]
      linkLabel: string
    }
    faq: {
      heading: string
      items: { q: string; a: string }[]
    }
    tips: {
      heading: string
      lines: string[]
      note: string
    }
  }
}
