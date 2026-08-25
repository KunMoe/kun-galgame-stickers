![kun-galgame-stickers](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/blob/svelte-kit/static/title.webp)

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

**[原画をダウンロード](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases)**

## 方針

できるだけ次に従います。

- ギャルゲー（Visual Novel）のスタンプだけ。アニメ・漫画・イラストは含めない
- 正方形の切り抜き
- 可能ならアホ毛まで入れる
- 小さくて可愛いソフト萌え、~~ふわふわ白毛の娘~~ が中心

初期に撮ったものは、この方針より前なので守っていません。

## 収録ゲーム

スタンプに使ったゲームです。自分で遊んでみてください。全部プレイ済みです。~~ぜんぶ萌えゲー。~~

[パックごとの収録ゲーム一覧](../../static/kun-galgame-stickers/introduction/game.md)

## サイトの機能

- パックと単体スタンプの閲覧。ゲーム名／キャラ名は Postgres から
- 鯤アカウントでログイン／登録（[kun-galgame-infra](https://github.com/KunMoe/kun-galgame-infra) の OAuth RP。httpOnly の session cookie。ブラウザに token は置かない）
- 既定は中国語。英語は `/en/...`
- ライト／ダーク／システム追従
- 原画 PNG のダウンロード。[About](https://sticker.kungal.com/about) に Telegram 索引

画像はまだこのリポジトリの静的ファイルです（一覧 webp は `static/stickers/`、原画は `static/kun-galgame-stickers/`）。オブジェクトストレージへの移行は予定であり、未着手です。

## 技術スタック

| 層       | 選定                                                                           |
| -------- | ------------------------------------------------------------------------------ |
| アプリ   | [SvelteKit](https://svelte.dev/docs/kit) 2 + Svelte 5 runes、`adapter-node`    |
| UI       | [Tailwind CSS](https://tailwindcss.com/) 4 + `@iconify/svelte`                 |
| DB       | PostgreSQL（`kungalgame_sticker`）、[Prisma](https://www.prisma.io/) 7         |
| 認証     | 鯤 OAuth（PKCE、confidential client、RFC 6749 / 6750 / 7009）                  |
| デプロイ | Docker イメージ → GHCR → [Dokploy](https://dokploy.com/)、`sticker.kungal.com` |

## 開発

Node 24 と pnpm 9+ が必要です。`.env.example` を `.env` にコピーし、`KUN_DATABASE_URL` と OAuth secret を入れてください。

```bash
pnpm install
# prisma CLI はプロジェクト依存関係ではない。バージョンは @prisma/client に合わせる
pnpm dlx prisma@7.9.1 generate
pnpm dev          # http://127.0.0.1:5173
pnpm run check
pnpm run lint
```

ローカルでログインするには OAuth サーバ（`.env` の `KUN_OAUTH_SERVER_URL`）も必要です。一覧を見るだけなら不要です。

## デプロイ

本番は共有 `dokploy-network` 上の単一 Node コンテナで、ハブの Postgres と OAuth を使います。CI は `svelte-kit` ブランチへの push で `ghcr.io/kunmoe/sticker-web` をビルドします。

手順（イメージ、環境変数、初回の `kungalgame_sticker` 作成、Dokploy）は [docs/deploy/README.md](../deploy/README.md) を見てください。

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
A: できます。GitHub が使えるなら PR を出して、`static/kun-galgame-stickers/Others/` の下に自分のフォルダを作って入れてください。下の交流グループに投げても構いません。

**Q: 第 1 パックだけ画像形式が混在しているのはなぜ？**
A: 本格収集の前に QQ から持ってきたものがあるからです。今後の更新はすべて PNG です。

**Q: なぜ多くの作品で一人分の表情しかないのですか？**
A: 単ルート戦士だからです。

気に入ったら GitHub の star をください。

Telegram グループ: https://t.me/kungalgame

Tips: グループルールはありません。蹴られても入り直せます。
