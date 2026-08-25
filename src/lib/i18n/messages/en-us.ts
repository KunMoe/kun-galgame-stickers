import type { Messages } from './shape'

const messages: Messages = {
  meta: {
    title: 'KUN Visual Novel Stickers',
    description:
      'Visual Novel Stickers Website, KUN Visual Novel, Visual Novel Stickers Download, Visual Novel, Stickers, Download',
    og: 'KUN Visual Novel',
    ogDescription:
      'KUN Visual Novel is a Visual Novel Alliance. The CUTEST Visual Novel Group. Visual Novel Communication, Visual Novel Technique. To Create The Best Atmosphere! NO ADs Forever! Free Forever!'
  },
  header: {
    title: 'KUN Visual Novel Stickers',
    home: 'Home',
    about: 'About',
    light: 'Light',
    dark: 'Dark',
    system: 'System'
  },
  home: {
    sticker: 'KUN Visual Novel Sticker',
    kun: 'KUN Visual Novel | Stickers',
    open: 'Open Source, Source Code on'
  },
  sticker: {
    title: 'Visual Novel Sticker Pack',
    description: 'Visual Novel sticker pack preview and downloads',
    game: 'Game',
    lass: 'Bishōjo',
    introduction: 'Introduction',
    original: 'Original',
    download: 'Download original',
    backToTop: 'Back to top',
    notFound: 'Sticker not found'
  },
  footer: {
    poweredBy: 'Powered by',
    forumName: 'KUN Visual Novel Forum',
    forumSuffix: ''
  },
  auth: {
    login: 'Sign in with KUN',
    register: 'Create a KUN account',
    logout: 'Sign out',
    account: 'Account',
    profile: 'Profile',
    anonymous: 'Not signed in',
    logoutTitle: 'Sign out',
    logoutPrompt: 'Choose what to sign out of:',
    logoutEverywhere: 'Sign out of this site and OAuth',
    logoutEverywhereHint:
      'Signs out of both this site and your OAuth account; other signed-in sites sign out on their next refresh; re-authentication is required next time. Best for public / shared devices.',
    logoutLocal: 'This site only',
    logoutLocalHint:
      'Signs out of this site only; your OAuth account and other sites stay signed in, and signing back in here is instant. Best for your own device.',
    cancel: 'Cancel'
  },
  about: {
    title: 'About - KUN Visual Novel Stickers',
    description:
      'Visual Novel Stickers Website, KUN Visual Novel, Visual Novel Stickers Download, Visual Novel, Stickers, Download',
    repo: 'kun-galgame-stickers-sveltekit',
    introduction: {
      heading: 'Introduction',
      lines: [
        'A video introducing this project has been made',
        'The website now hosts over 400 stickers!'
      ],
      bilibiliPrefix:
        'This video is also released on Bilibili — give it a thumbs up if you like it',
      bilibiliLink: 'I built a Galgame sticker repository with over 350 pure galgame stickers',
      purposeIntro:
        'These are stickers I personally captured from various visual novels, with the purpose of:',
      purpose: ['Recommending fun and cute games to more people through stickers', 'Moe!'],
      future:
        'In the future, some SD_CG and CG screenshots from games may be added. For now we only have pure character expressions.'
    },
    telegram: {
      heading: 'Telegram Sticker Packs',
      intro:
        'These stickers are mirrored to Telegram in packs of 80 — click the links below to add them',
      packLabel: 'KUN Visual Novel Sticker Pack',
      note: 'The stickers in the repository are clearer than those in Telegram because the Telegram set is compressed',
      downloadIntro: 'You can download the original stickers below',
      downloadLink: 'Download'
    },
    rules: {
      heading: 'Standards',
      intro: 'This sticker collection tries to follow these standards',
      items: [
        'Visual Novel (galgame) stickers only — no anime, manga or illustrations',
        'Square screenshots',
        "Capture the character's ahoge in full whenever possible",
        'Mostly small, cute, soft-moe, fluffy white-haired lass characters'
      ],
      note: 'Some early stickers do not follow these standards because they were captured too early'
    },
    games: {
      heading: 'Related Games',
      lines: [
        'Below are the games used in the stickers — go play them yourself',
        "Of course I have played all of these games — they're all moe games",
        'Click here to see exactly which games were used'
      ],
      linkLabel: 'Sticker pack game overview'
    },
    faq: {
      heading: 'FAQ',
      items: [
        {
          q: 'Will these stickers be continuously updated?',
          a: 'Of course. I capture stickers while playing games — as long as I keep playing visual novels, updates will keep coming'
        },
        {
          q: 'Can I contribute stickers to this collection?',
          a: 'Absolutely! If you are familiar with GitHub, submit a PR placing your stickers in a subfolder under Others'
        },
        {
          q: 'Why is the image format of the first sticker pack different?',
          a: 'Some were copied directly from QQ before I had started collecting deliberately. All future updates will be in png format'
        },
        {
          q: 'Why does many games only have expressions from a single character?',
          a: 'I am a one-route warrior'
        }
      ]
    },
    tips: {
      heading: 'Tips',
      lines: [
        'If you think we are doing well, feel free to give us a star',
        'Our Telegram group: https://t.me/kungalgame'
      ],
      note: 'Tips: There are no group rules — if you get kicked you can rejoin'
    }
  }
}

export default messages
