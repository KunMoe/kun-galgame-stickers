import type { Messages } from './shape'

const messages: Messages = {
  meta: {
    title: '鲲 Galgame 表情包',
    description: 'Galgame 表情包网站, 鲲 Galgame, Galgame 表情包下载, Galgame, 表情包, 下载',
    og: '鲲 Galgame',
    ogDescription:
      '鲲 Galgame 是一个 Galgame 联盟, 是世界上最萌的 Galgame 集体! Galgame 交流讨论, Galgame 技术交流。为营造最好的氛围而努力! 永远不会有广告! 永远免费!'
  },
  header: {
    title: '鲲 Galgame 表情包',
    home: '主页',
    about: '关于',
    light: '白天',
    dark: '黑夜',
    system: '系统'
  },
  home: {
    sticker: '鲲 Galgame 表情包',
    kun: '鲲 Galgame | 表情包',
    open: '本网站开源, 源码在'
  },
  sticker: {
    title: 'Galgame 表情包',
    description: 'Galgame 表情包预览, 下载',
    game: '游戏名',
    lass: '少女名',
    introduction: '简介',
    original: '原图',
    download: '原图下载',
    backToTop: '返回顶部',
    notFound: '找不到该表情包'
  },
  footer: {
    poweredBy: '由',
    forumName: '鲲 Galgame 论坛',
    forumSuffix: '提供支持'
  },
  auth: {
    login: '使用鲲账号登录',
    register: '注册鲲账号',
    logout: '登出',
    account: '账号',
    profile: '个人资料',
    anonymous: '未登录'
  },
  about: {
    title: '关于 - 鲲 Galgame 表情包',
    description: 'Galgame 表情包网站, 鲲 Galgame, Galgame 表情包下载',
    repo: 'kun-galgame-stickers-sveltekit',
    introduction: {
      heading: '介绍',
      lines: ['做了一个视频介绍这个项目啦', '目前, 这个网站的表情包数量已经超过 400 张了!'],
      bilibiliPrefix: '本视频同步发布于 Bilibili, 喜欢可以来点个赞哦',
      bilibiliLink: '350多张纯 galgame 表情包, 我建了一个 galgame 表情包仓库',
      purposeIntro: '这是由我个人在各种 galgame 里截的表情包, 目的是',
      purpose: ['通过表情包的方式给更多人推荐好玩的萌萌游戏', '萌!'],
      future: '以后可能会更新一些游戏的 SD_CG 和 CG 的截图作为表情包, 现在只有游戏中人物的纯表情'
    },
    telegram: {
      heading: 'Telegram 贴纸包',
      intro: '这套贴纸会同步在 Telegram 的贴纸集内更新, 每套 80 张, 您也可以点击下方的链接添加贴纸',
      packLabel: '鲲 Galgame 表情包',
      note: '仓库中的表情包比 Telegram 中的贴纸集清晰, 因为做成 Telegram 贴纸集时对质量做了压缩',
      downloadIntro: '你可以点击下面的链接下载这些贴纸',
      downloadLink: '下载链接'
    },
    rules: {
      heading: '规范',
      intro: '这个表情包系列尽量遵循以下几点规范',
      items: [
        '全部为 galgame (Visual Novel) 的表情包, 不包括动漫、漫画、插画等',
        '正方形截图',
        '尽量把人物的呆毛截全',
        '以小只可爱软萌白毛为主'
      ],
      note: '由于部分表情包截图的时间过早, 所以没有遵循此规范'
    },
    games: {
      heading: '相关游戏',
      lines: [
        '下面是表情包中用到的游戏, 大家可以自己去玩相应的游戏',
        '当然这些游戏我都玩过, 全部是萌萌游戏',
        '大家可以点击这里查看表情包里面使用了哪些游戏截的图'
      ],
      linkLabel: '表情包游戏概览'
    },
    faq: {
      heading: 'FAQ',
      items: [
        {
          q: '这些表情包会不断更新吗?',
          a: '当然会。我边打游戏会边截表情包, 只要我还在玩 galgame, 那就一定会更新'
        },
        {
          q: '我可以为这个表情包系列贡献表情吗?',
          a: '当然可以! 如果您熟悉 GitHub, 可以提交 PR, 将自己的表情放在 Others 文件夹下您自己命名的目录中'
        },
        {
          q: '为什么第一套贴纸的图片格式不一样?',
          a: '有一些是直接从 QQ 搬过来的, 以前还没有专门收集这些贴纸。今后所有更新均为 png 格式'
        },
        {
          q: '为什么很多游戏一整部只有一个人的表情?',
          a: '我是单线战士'
        }
      ]
    },
    tips: {
      heading: 'Tips',
      lines: ['如果您觉得我们做得不错, 欢迎给我们点个 star 哦~', '我们的 Telegram 群组: https://t.me/kungalgame'],
      note: 'Tips: 我们没有群规, 被鲨了可以重新加'
    }
  }
}

export default messages
