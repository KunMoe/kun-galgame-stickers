![kun-galgame-stickers](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/blob/svelte-kit/static/title.webp)

### **[English](README.md)** | **[日本語](docs/readme/jp.md)** | **[简体中文](Readme_zh_cn.md)** | **[繁體中文](docs/readme/cht.md)**

**Site: [sticker.kungal.com](https://sticker.kungal.com)** · **[Telegram](https://t.me/kungalgame)**

# KUN Visual Novel Stickers

A sticker / emoji-pack site for visual novels (galgame). It is a subsite of [KUN Visual Novel](https://www.kungal.com) and does **not** follow the forum's operating rules.

There are currently **7 packs and 498 stickers** (80 per pack; pack 7 has 18). They are screenshots of character expressions, taken while playing the games.

The point of this collection is:

- Recommending fun and cute games to more people through stickers
- Moe!

SD_CG and CG stills may show up later. For now it is only pure character expressions. Join the group at the end of this page if you like.

## Telegram sticker packs

Packs are mirrored to Telegram, 80 stickers per set (pack 7 is smaller). Add them here:

- [KUN Galgame Sticker Pack [1]](https://t.me/addstickers/KUNgal1)
- [KUN Galgame Sticker Pack [2]](https://t.me/addstickers/KUNgal2)
- [KUN Galgame Sticker Pack [3]](https://t.me/addstickers/KUNgal3)
- [KUN Galgame Sticker Pack [4]](https://t.me/addstickers/KUNgal4)
- [KUN Galgame Sticker Pack [5]](https://t.me/addstickers/KUNgal5)
- [KUN Galgame Sticker Pack [6]](https://t.me/addstickers/KUNgal6)
- [KUN Galgame Sticker Pack [7]](https://t.me/addstickers/KUNgal7)

The copies in this repository are sharper than the Telegram sets — Telegram compresses them.

**[Download the originals](https://github.com/KUN1007/kun-galgame-stickers-sveltekit/releases)**

## Standards

This series tries to follow:

- Visual novel (galgame) stickers only — no anime, manga, or illustrations
- Square screenshots
- Capture the character's ahoge in full whenever possible
- Mostly small, cute, soft-moe ~~fluffy white-haired lass~~ characters

Some early stickers do not follow these, because they were captured before the rules existed.

## Related games

Every game used here is one I have actually played. ~~They are all moe games.~~

[Overview of games in the sticker packs](static/kun-galgame-stickers/introduction/game.md)

## What the site does

- Browse packs and single stickers, with game / character names from Postgres
- Sign in / register with a KUN account (OAuth RP of [kun-galgame-infra](https://github.com/KunMoe/kun-galgame-infra); httpOnly session cookie, no tokens in the browser)
- Chinese by default; English under `/en/...`
- Light / dark / system theme
- Download original PNGs; Telegram links on the [About](https://sticker.kungal.com/about) page

Images still live as static files in this repo (`apps/web/public/stickers/` for list webp, `apps/web/public/kun-galgame-stickers/` for originals). Moving them to object storage is planned, not done.

## Tech stack

| Layer    | Choice                                                                                       |
| -------- | -------------------------------------------------------------------------------------------- |
| Frontend | [Nuxt 4](https://nuxt.com/) + [KunUI](https://ui.kungal.com/) (`apps/web`)                   |
| i18n     | `@nuxtjs/i18n` — Chinese default, `/en`, `/ja`                                               |
| API      | [Go Fiber](https://gofiber.io/) v3 (`apps/api`)                                              |
| Database | PostgreSQL (`kungalgame_sticker`) through GORM; SQL migrations in `apps/api/migrations`      |
| Auth     | KUN OAuth (PKCE, confidential client, RFC 6749 / 6750 / 7009 protocol endpoints)             |
| Deploy   | Docker images → GHCR → [Dokploy](https://dokploy.com/) at `sticker.kungal.com`               |

## Development

Needs Node 24, pnpm 11, and Go 1.26. Copy `.env.example` to `.env` and fill in `KUN_DATABASE_URL` plus the OAuth secret.

```bash
pnpm install
pnpm migrate      # first time, against kungalgame_sticker
pnpm dev          # web http://127.0.0.1:5173  ·  api :9421
pnpm run typecheck
pnpm run lint
```

Login in local dev also needs a running OAuth server (`KUN_OAUTH_SERVER_URL` in `.env`). Pack listing works without it.

## Deployment

Production is two containers on the shared `dokploy-network`: `web` (Nitro) and `api` (Fiber), plus a one-shot `migrate`. Traefik should send `sticker.kungal.com/api` to the API and everything else to the web. CI builds `ghcr.io/kunmoe/sticker-web`, `sticker-api`, and `sticker-migrate` on push to `svelte-kit`.

See [docs/deploy/README.md](docs/deploy/README.md) for the full guide.

## Roadmap

- [x] KUN account login (OAuth)
- [x] Telegram pack index (this README and `/about`)
- [ ] User uploads with a KUN account
- [ ] Public API docs for third-party use
- [ ] Move sticker images to object storage
- [ ] SD_CG section for galgame SD_CG

## FAQ

**Q: Will these stickers keep being updated?**
A: Yes. I capture stickers while playing. As long as I still play visual novels, there will be updates.

**Q: Can I contribute stickers?**
A: Yes. If you use GitHub, open a PR and put your stickers in a named folder under `static/kun-galgame-stickers/Others/`. You can also drop them in the visual-novel group below.

**Q: Why is the first pack's image format mixed?**
A: Some files were copied from QQ before this collection was deliberate. Everything new is PNG.

**Q: Why do many games only have expressions from one character?**
A: I am a one-route warrior.

If you like this, a GitHub star is welcome.

Telegram group: https://t.me/kungalgame

Tips: there are no group rules. If you get kicked, you can rejoin.
