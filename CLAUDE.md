# kun-galgame-stickers (sticker / emoji pack) — AI Agent Project Guide

## 铁律 (Iron Rules — non-negotiable; these override every other guideline in this file)

1. **No background gradients in any UI, ever.** Never use gradient backgrounds in UI design (`bg-gradient-*`, `from-*/via-*/to-*`, `linear-gradient()`, `radial-gradient()`, `conic-gradient()`, etc.); use solid colors from the project's palette.
2. Prefer KunUI components (`KunButton`, `KunCard`, `KunModal`, …). Do not change KunUI itself — report bugs or missing pieces to the user.

A galgame **emoji pack** site. pnpm workspace: **Nuxt 4 + KunUI** frontend (`apps/web`) and **Go Fiber** API (`apps/api`). i18n is `@nuxtjs/i18n` (`zh-cn` default, `/en`, `/ja`). Database is PostgreSQL `kungalgame_sticker` via GORM; schema changes are SQL files in `apps/api/migrations`. Authentication is an **OAuth RP** of kun-galgame-infra (BFF httpOnly cookies; protocol endpoints are RFC-only). Cross-service contracts belong to infra, see `../kun-galgame-infra`.

Frontend layout follows kaguya-web practices: thin SFCs, domain code in `app/features/<domain>/`, `kunFetch` as the only API entry. Backend layout: `internal/platform/<domain>/{dto,handler,model,repository,service}`; handlers own HTTP, services do not import Fiber.

## Core Engineering Principles

> Shared baseline across all KUN Galgame repositories. Defaults, not dogma — apply judgment.

1. All commit messages must be written entirely in English.
2. All code comments must be written entirely in English.
3. Keep each source file under ~500 lines where practical; once a file grows past ~300 lines, consider splitting it (a guideline, not a hard rule).
4. Write every frontend function as an arrow function; compose/merge class names with `cn` wherever practical.
5. Deliberately balance elegant modularity against necessary duplication — choose per case instead of always favoring either.
6. Constantly verify that frontend and backend agree on the data: field shapes and response formats must match what each side expects. House envelope is `{ code, message, data }` with `code === 0` on success.
7. After every change, watch for unintended side effects elsewhere.
8. If a change requires running a migration, tell the user explicitly at the end — which command, and against which database.
9. Always seek the most modern, elegant solution that fits the project's current state; consult the latest official docs and resources online when useful.
10. Never let the pursuit of elegance or modularity make the code complex or hard to follow, and don't write over-defensive code.

## Database schema changes → migration reminder is mandatory

Whenever this change adds or edits `apps/api/migrations` (or otherwise changes the Postgres schema), **at the end of the task you must explicitly tell the user: whether the production schema needs to be synced, and which command to run** (`pnpm migrate` / `go run ./cmd/migrate -dir up` against `kungalgame_sticker`). Skipping it → production code reads a column that does not exist → outage (cf. the 2026-06 infra moemoepoint distribution outage: a missing column left the whole site unable to receive moemoepoint for ~29h).
