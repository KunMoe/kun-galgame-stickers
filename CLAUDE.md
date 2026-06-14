# kun-galgame-stickers（sticker / 表情包）— AI 代理项目指南

galgame **表情包**站。SvelteKit（Svelte 5 runes）+ Tailwind v4 + `@iconify/svelte`；i18n 在 `src/lib/i18n/messages/{en-us,zh-cn}.ts`（改文案要同步 `shape.ts` 类型 + 两个 locale）。数据库用 **Prisma**（`prisma/schema.prisma`，PG 库 `kungalgame_sticker`）。鉴权是 kun-galgame-infra 的 **OAuth RP**（BFF 不透明会话，httpOnly cookie；OAuth / 身份 / 萌萌点等跨服务契约全归 infra，见 `../kun-galgame-infra`）。

## 数据库 schema 变更 → 必须提醒迁移

只要本次改动动了 `prisma/schema.prisma`（加/改 model 或字段），**任务结束时必须明确告诉用户：是否需要同步生产 schema、跑哪个命令**（`prisma migrate deploy` 或 `prisma db push`）。漏跑 → 线上代码读到不存在的列 → 故障（参考 2026-06 infra 萌萌点发放故障：缺一列导致全站 ~29h 拿不到萌萌点）。
