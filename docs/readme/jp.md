![kun-galgame-stickers](https://github.com/KunMoe/kun-galgame-stickers/blob/svelte-kit/apps/web/public/title.webp)

### **[English](../../README.md)** | **[日本語](jp.md)** | **[简体中文](../../Readme_zh_cn.md)** | **[繁體中文](cht.md)**

**サイト: [sticker.kungal.com](https://sticker.kungal.com)** · **[Telegram](https://t.me/kungalgame)**

# 鯤 Galgame スタンプ

視覚小説（galgame）のスタンプ／絵文字パックサイトです。[鯤 Galgame フォーラム](https://www.kungal.com) のサブサイトで、フォーラムの運営方針には**従いません**。

現在 **7 パック、498 枚**（1 パック 80 枚、第 7 パックは 18 枚）。プレイ中に撮ったキャラの表情だけです。

目的は：

- スタンプを通して、おもしろい萌えゲーをもっと多くの人に薦めること
- 萌え！

いずれ SD_CG や CG の切り抜きも足すかもしれません。今は表情だけです。末尾のグループにもどうぞ。

## Telegram スタンプパック

Telegram にも同じセットを同期しています。1 セット 80 枚（第 7 パックはそれ未満）。追加はこちら：

- [鯤 Galgame スタンプ [1]](https://t.me/addstickers/KUNgal1)
- [鯤 Galgame スタンプ [2]](https://t.me/addstickers/KUNgal2)
- [鯤 Galgame スタンプ [3]](https://t.me/addstickers/KUNgal3)
- [鯤 Galgame スタンプ [4]](https://t.me/addstickers/KUNgal4)
- [鯤 Galgame スタンプ [5]](https://t.me/addstickers/KUNgal5)
- [鯤 Galgame スタンプ [6]](https://t.me/addstickers/KUNgal6)
- [鯤 Galgame スタンプ [7]](https://t.me/addstickers/KUNgal7)

リポジトリ内の画像の方が Telegram よりきれいです。スタンプセット化のときに圧縮されています。

**[原画をダウンロード](https://github.com/KunMoe/kun-galgame-stickers/releases)**

## 方針

できるだけ次に従います。

- ギャルゲー（Visual Novel）のスタンプだけ。アニメ・漫画・イラストは含めない
- 正方形の切り抜き
- 可能ならアホ毛まで入れる
- 小さくて可愛いソフト萌え、~~ふわふわ白毛の娘~~ が中心

初期に撮ったものは、この方針より前なので守っていません。

## 収録ゲーム

スタンプに使ったゲームです。自分で遊んでみてください。全部プレイ済みです。~~ぜんぶ萌えゲー。~~

[パックごとの収録ゲーム一覧](../../apps/web/public/kun-galgame-stickers/introduction/game.md)

## サイトの機能

- パックと単体スタンプの閲覧。ゲーム名／キャラ名は Postgres から
- 鯤アカウントでログイン／登録（[kun-galgame-infra](https://github.com/KunMoe/kun-galgame-infra) の OAuth RP。httpOnly の session cookie。ブラウザに token は置かない）
- 既定は中国語。英語は `/en/...`
- ライト／ダーク／システム追従
- 原画 PNG のダウンロード。[About](https://sticker.kungal.com/about) に Telegram 索引

画像はまだこのリポジトリの静的ファイルです（一覧 webp は `apps/web/public/stickers/`、原画は `apps/web/public/kun-galgame-stickers/`）。オブジェクトストレージへの移行は予定であり、未着手です。

## 技術スタック

| 層       | 選定                                                                                          |
| -------- | --------------------------------------------------------------------------------------------- |
| フロント | [Nuxt 4](https://nuxt.com/) + [KunUI](https://ui.kungal.com/)（`apps/web`）                   |
| i18n     | `@nuxtjs/i18n` — 既定は中国語、`/en` 英語、`/ja` 日本語                                       |
| API      | [Go Fiber](https://gofiber.io/) v3（`apps/api`）                                              |
| DB       | PostgreSQL（`kungalgame_sticker`）、GORM。SQL マイグレーションは `apps/api/migrations`        |
| 認証     | 鯤 OAuth（PKCE、confidential client、RFC 6749 / 6750 / 7009）                                 |
| デプロイ | Docker イメージ → GHCR → [Dokploy](https://dokploy.com/)、`sticker.kungal.com`                |

## 開発

Node 24、pnpm 11、Go 1.26 が必要です。`.env.example` を `.env` にコピーし、`KUN_DATABASE_URL` と OAuth secret を入れてください。

```bash
pnpm install
pnpm migrate      # 初回、kungalgame_sticker に対して
pnpm dev          # web http://127.0.0.1:5173  ·  api :9421
pnpm run typecheck
pnpm run lint
```

ローカルでログインするには OAuth サーバ（`.env` の `KUN_OAUTH_SERVER_URL`）も必要です。一覧を見るだけなら不要です。

## デプロイ

本番は共有 `dokploy-network` 上の常駐 2 コンテナ：`web`（Nitro）と `sticker-api`（Fiber）。毎回の compose `up` でワンショット `migrate` が先に走り、成功してから API が起動します（kungal / moyu / infra と同じ。手作業の migrate は不要）。Traefik は `sticker.kungal.com/api` を `sticker-api:9421` に、それ以外を `web:3000` に回します。CI は `svelte-kit` ブランチへの push で `sticker-web` / `sticker-api` / `sticker-migrate` をビルドします。

手順は [docs/deploy/README.md](../deploy/README.md) を見てください。

## ロードマップ

- [x] 鯤アカウントログイン（OAuth）
- [x] Telegram パック索引（本 README と `/about`）
- [ ] 鯤アカウントによるユーザー投稿
- [ ] 第三者向け公開 API ドキュメント
- [ ] スタンプ画像のオブジェクトストレージ移行
- [ ] SD_CG セクション

## FAQ

**Q: スタンプは更新され続けますか？**
A: はい。ゲームをやりながら撮っています。ギャルゲーを続けている限り更新します。

**Q: スタンプを投稿できますか？**
A: できます。GitHub が使えるなら PR を出して、`apps/web/public/kun-galgame-stickers/Others/` の下に自分のフォルダを作って入れてください。下の交流グループに投げても構いません。

**Q: 第 1 パックだけ画像形式が混在しているのはなぜ？**
A: 本格収集の前に QQ から持ってきたものがあるからです。今後の更新はすべて PNG です。

**Q: なぜ多くの作品で一人分の表情しかないのですか？**
A: 単ルート戦士だからです。

気に入ったら GitHub の star をください。

Telegram グループ: https://t.me/kungalgame

Tips: グループルールはありません。蹴られても入り直せます。
