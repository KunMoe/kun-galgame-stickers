![kun-galgame-stickers](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/blob/svelte-kit/static/title.webp)

### **[English](README.md)** | **[日本語](docs/readme/jp.md)** | **[简体中文](Readme_zh_cn.md)** | **[繁體中文](docs/readme/cht.md)**

**站点：[sticker.kungal.com](https://sticker.kungal.com)** · **[Telegram](https://t.me/kungalgame)**

# 鲲 Galgame 表情包

鲲 Galgame 表情包是 [鲲 Galgame 论坛](https://www.kungal.com) 的子网站，专门收集 galgame 表情包，**不遵循**论坛的运营理念。

目前共 **7 套、498 张**（每套 80 张，第 7 套 18 张）。都是打游戏时截的角色纯表情。

目的是：

- 通过表情包的方式给更多人推荐好玩的萌萌游戏
- 萌！

以后可能会更新一些游戏的 SD_CG 和 CG 截图。现在只有人物纯表情。也可以加入文末的群组。

## Telegram 贴纸包

贴纸会同步更新到 Telegram，每套 80 张（第 7 套不足 80）。点击添加：

- [鲲 Galgame 表情包[1]](https://t.me/addstickers/KUNgal1)
- [鲲 Galgame 表情包[2]](https://t.me/addstickers/KUNgal2)
- [鲲 Galgame 表情包[3]](https://t.me/addstickers/KUNgal3)
- [鲲 Galgame 表情包[4]](https://t.me/addstickers/KUNgal4)
- [鲲 Galgame 表情包[5]](https://t.me/addstickers/KUNgal5)
- [鲲 Galgame 表情包[6]](https://t.me/addstickers/KUNgal6)
- [鲲 Galgame 表情包[7]](https://t.me/addstickers/KUNgal7)

**仓库里的图比 Telegram 贴纸集更清晰**，做成贴纸集时质量被压缩过。

**[点此下载原图](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases)**

## 规范

这个系列尽量遵循：

- **全部为 galgame（Visual Novel）表情包**，不包括动漫、漫画、插画等
- 正方形截图
- 尽量把人物的呆毛截全
- 以小只可爱软萌 ~~白毛~~ 为主

部分表情包截得太早，没有遵循这些规范。

## 相关游戏

下面是表情包里用到的游戏，可以自己去玩。这些我都打过，~~全部是萌萌游戏~~。

[查看表情包用了哪些游戏的截图](static/kun-galgame-stickers/introduction/game.md)

## 网站功能

- 浏览表情包与单张详情，游戏名 / 少女名来自 Postgres
- 使用鲲账号登录 / 注册（[kun-galgame-infra](https://github.com/KunMoe/kun-galgame-infra) 的 OAuth RP；httpOnly session cookie，浏览器里不放 token）
- 默认中文，英文走 `/en/...`
- 白天 / 黑夜 / 跟随系统
- 可下载原图 PNG；Telegram 索引在 [关于页](https://sticker.kungal.com/about)

图片目前仍作为仓库静态资源（列表图 `apps/web/public/stickers/`，原图 `apps/web/public/kun-galgame-stickers/`）。迁到对象存储是既定方向，还没做。

## 技术栈

| 层     | 选型                                                                                          |
| ------ | --------------------------------------------------------------------------------------------- |
| 前端   | [Nuxt 4](https://nuxt.com/) + [KunUI](https://ui.kungal.com/)（`apps/web`）                   |
| i18n   | `@nuxtjs/i18n` — 默认中文，`/en` 英语，`/ja` 日语                                             |
| API    | [Go Fiber](https://gofiber.io/) v3（`apps/api`）                                              |
| 数据库 | PostgreSQL（`kungalgame_sticker`），GORM；SQL 迁移在 `apps/api/migrations`                    |
| 鉴权   | 鲲 OAuth（PKCE、confidential client，RFC 6749 / 6750 / 7009 协议端点）                        |
| 部署   | Docker 镜像 → GHCR → [Dokploy](https://dokploy.com/)，域名 `sticker.kungal.com`               |

## 本地开发

需要 Node 24、pnpm 11 和 Go 1.26。复制 `.env.example` 为 `.env`，填入 `KUN_DATABASE_URL` 和 OAuth secret。

```bash
pnpm install
pnpm migrate      # 首次，针对 kungalgame_sticker
pnpm dev          # web http://127.0.0.1:5173  ·  api :9421
pnpm run typecheck
pnpm run lint
```

本地登录还需要 OAuth 服务（`.env` 里的 `KUN_OAUTH_SERVER_URL`）。只看表情包列表不需要登录。

## 部署

生产是共享 `dokploy-network` 上的两个容器：`web`（Nitro）和 `api`（Fiber），外加一次性 `migrate`。Traefik 把 `sticker.kungal.com/api` 指到 API，其余指到 web。CI 在 `svelte-kit` 分支 push 后构建 `sticker-web` / `sticker-api` / `sticker-migrate`。

完整步骤见 [docs/deploy/README.md](docs/deploy/README.md)。

## 待办

- [x] 鲲账号登录（OAuth）
- [x] Telegram 贴纸索引（本 README 与 `/about`）
- [ ] 鲲账号用户上传
- [ ] 面向第三方的公开 API 文档
- [ ] 表情图迁到对象存储
- [ ] SD_CG 板块

## FAQ

**Q：这些表情包会不断更新吗？**
A：当然会。我边打游戏边截，只要还在玩 galgame，就一定会更新。

**Q：我可以贡献表情吗？**
A：可以。熟悉 GitHub 的话提 PR，把表情放到 `static/kun-galgame-stickers/Others/` 下你自己命名的目录。也可以发到下面的 galgame 交流群。

**Q：为什么第一套的图片格式不统一？**
A：有一些是直接从 QQ 搬过来的，那时还没开始专门收集。以后更新都是 png。

**Q：为什么很多游戏一整部只有一个人的表情？**
A：~~我是单线战士~~

如果觉得还不错，欢迎点个 star。

鲲的 galgame 交流群：850949010

TG 群：https://t.me/kungalgame

Tips：我们没有群规，被鲨了可以重新加。
