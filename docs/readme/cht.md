![kun-galgame-stickers](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/blob/svelte-kit/static/title.webp)

### **[English](../../README.md)** | **[日本語](jp.md)** | **[简体中文](../../Readme_zh_cn.md)** | **[繁體中文](cht.md)**

**網站：[sticker.kungal.com](https://sticker.kungal.com)** · **[Telegram](https://t.me/kungalgame)**

# 鯤 Galgame 表情包

鯤 Galgame 表情包是 [鯤 Galgame 論壇](https://www.kungal.com) 的子網站，專門收集 galgame 表情包，**不遵循**論壇的營運理念。

目前共 **7 套、498 張**（每套 80 張，第 7 套 18 張）。都是打遊戲時截的角色純表情。

目的是：

- 用表情包把好玩的萌系遊戲推薦給更多人
- 萌！

以後可能會更新一些遊戲的 SD_CG 與 CG 截圖。現在只有人物純表情。也可以加入文末的群組。

## Telegram 貼圖包

貼圖會同步更新到 Telegram，每套 80 張（第 7 套不足 80）。點擊新增：

- [鯤 Galgame 表情包[1]](https://t.me/addstickers/KUNgal1)
- [鯤 Galgame 表情包[2]](https://t.me/addstickers/KUNgal2)
- [鯤 Galgame 表情包[3]](https://t.me/addstickers/KUNgal3)
- [鯤 Galgame 表情包[4]](https://t.me/addstickers/KUNgal4)
- [鯤 Galgame 表情包[5]](https://t.me/addstickers/KUNgal5)
- [鯤 Galgame 表情包[6]](https://t.me/addstickers/KUNgal6)
- [鯤 Galgame 表情包[7]](https://t.me/addstickers/KUNgal7)

**倉庫裡的圖比 Telegram 貼圖集更清晰**，做成貼圖集時品質被壓縮過。

**[點此下載原圖](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases)**

## 規範

這個系列盡量遵循：

- **全部為 galgame（Visual Novel）表情包**，不包括動漫、漫畫、插畫等
- 正方形截圖
- 盡量把人物的呆毛截全
- 以小隻可愛軟萌 ~~白毛~~ 為主

部分表情包截得太早，沒有遵循這些規範。

## 相關遊戲

下面是表情包裡用到的遊戲，可以自己去玩。這些我都打過，~~全部是萌萌遊戲~~。

[查看表情包用了哪些遊戲的截圖](../../static/kun-galgame-stickers/introduction/game.md)

## 網站功能

- 瀏覽表情包與單張詳情，遊戲名／少女名來自 Postgres
- 使用鯤帳號登入／註冊（[kun-galgame-infra](https://github.com/KunMoe/kun-galgame-infra) 的 OAuth RP；httpOnly session cookie，瀏覽器裡不放 token）
- 預設中文，英文走 `/en/...`
- 白天／黑夜／跟隨系統
- 可下載原圖 PNG；Telegram 索引在 [關於頁](https://sticker.kungal.com/about)

圖片目前仍作為倉庫靜態資源（列表圖 `static/stickers/`，原圖 `static/kun-galgame-stickers/`）。遷到物件儲存是既定方向，還沒做。

## 技術棧

| 層     | 選型                                                                            |
| ------ | ------------------------------------------------------------------------------- |
| 應用   | [SvelteKit](https://svelte.dev/docs/kit) 2 + Svelte 5 runes，`adapter-node`     |
| UI     | [Tailwind CSS](https://tailwindcss.com/) 4 + `@iconify/svelte`                  |
| 資料庫 | PostgreSQL（`kungalgame_sticker`），[Prisma](https://www.prisma.io/) 7          |
| 鑑權   | 鯤 OAuth（PKCE、confidential client，RFC 6749 / 6750 / 7009 協定端點）          |
| 部署   | Docker 映像 → GHCR → [Dokploy](https://dokploy.com/)，網域 `sticker.kungal.com` |

## 本機開發

需要 Node 24 和 pnpm 9+。複製 `.env.example` 為 `.env`，填入 `KUN_DATABASE_URL` 和 OAuth secret。

```bash
pnpm install
# prisma CLI 不是專案依賴，版本與 @prisma/client 對齊
pnpm dlx prisma@7.9.1 generate
pnpm dev          # http://127.0.0.1:5173
pnpm run check
pnpm run lint
```

本機登入還需要 OAuth 服務（`.env` 裡的 `KUN_OAUTH_SERVER_URL`）。只看表情包列表不需要登入。

## 部署

生產是共享 `dokploy-network` 上的單個 Node 容器，複用樞紐的 Postgres 和 OAuth。CI 在 `svelte-kit` 分支 push 後建構 `ghcr.io/kunmoe/sticker-web`。

完整步驟（映像、環境變數、一次性建庫 `kungalgame_sticker`、Dokploy）見 [docs/deploy/README.md](../deploy/README.md)。

## 待辦

- [x] 鯤帳號登入（OAuth）
- [x] Telegram 貼圖索引（本 README 與 `/about`）
- [ ] 鯤帳號使用者上傳
- [ ] 面向第三方的公開 API 文件
- [ ] 表情圖遷到物件儲存
- [ ] SD_CG 板塊

## FAQ

**Q：這些表情包會不斷更新嗎？**
A：當然會。我邊打遊戲邊截，只要還在玩 galgame，就一定會更新。

**Q：我可以貢獻表情嗎？**
A：可以。熟悉 GitHub 的話提 PR，把表情放到 `static/kun-galgame-stickers/Others/` 下你自己命名的目錄。也可以發到下面的 galgame 交流群。

**Q：為什麼第一套的圖片格式不統一？**
A：有一些是直接從 QQ 搬過來的，那時還沒開始專門收集。以後更新都是 png。

**Q：為什麼很多遊戲一整部只有一個人的表情？**
A：~~我是單線戰士~~

如果覺得還不錯，歡迎點個 star。

鯤的 galgame 交流群：850949010

TG 群：https://t.me/kungalgame

Tips：我們沒有群規，被鯊了可以重新加。
