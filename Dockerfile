# syntax=docker/dockerfile:1
#
# Multi-stage build for the SvelteKit (adapter-node) sticker site.
#
# Runtime config is NOT baked: KUN_DATABASE_URL / KUN_OAUTH_* are read at
# runtime via $env/dynamic/private and injected by Dokploy's Environment panel
# (or docker-compose.prod.yml). The only build-time value is the non-secret
# cookie name PUBLIC_KUN_STICKER_THEME ($env/static/public), baked into the
# client bundle. This keeps secrets out of any image pushed to a registry.
#
# Node 24 + pnpm 9 (pnpm-lock.yaml is lockfileVersion 9.0).
ARG NODE_VERSION=24

# ---- base: node + pnpm ------------------------------------------------------
FROM node:${NODE_VERSION}-trixie-slim AS base
RUN npm install -g pnpm@9
WORKDIR /repo

# ---- deps: full dependency graph, cached on the lockfile --------------------
FROM base AS deps
COPY package.json pnpm-lock.yaml .npmrc ./
RUN pnpm install --frozen-lockfile

# ---- build: prisma client + vite/adapter-node build, then prune to prod -----
FROM deps AS build
# Non-secret public cookie name, baked into the client bundle at build time.
ARG PUBLIC_KUN_STICKER_THEME=kun-sticker-theme
ENV PUBLIC_KUN_STICKER_THEME=${PUBLIC_KUN_STICKER_THEME}
COPY . .
# Prisma's config loader (prisma.config.ts) resolves env('KUN_DATABASE_URL')
# eagerly, so ANY prisma CLI command — even `generate`, which never connects —
# needs the var merely SET. Provide a throwaway value here; the real URL is
# injected at runtime (this builder-stage ENV is not inherited by `run`).
ENV KUN_DATABASE_URL=postgresql://build:build@localhost:5432/build
# Generate the Prisma client (writes into @prisma/client), then build. The
# `prisma` CLI is intentionally NOT a project dependency — listing it makes the
# lockfile resolve it as @prisma/client's optional peer, dragging a ~350MB tree
# (studio-core, effect, pglite, engines, typescript…) into the runtime image.
# `pnpm dlx` runs the CLI from a throwaway cache instead; pin it to the
# @prisma/client version. The adapter-node output lands in `kun-love-ren/`.
RUN pnpm dlx prisma@7.8.0 generate
RUN pnpm run build
# Drop dev tooling (vite/svelte/eslint/…) from node_modules; the generated
# @prisma/client + @prisma/adapter-pg + pg are prod deps and survive the prune.
RUN pnpm prune --prod

# ---- run: node + self-contained server output + pruned prod node_modules ----
FROM node:${NODE_VERSION}-trixie-slim AS run
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000
WORKDIR /app
# package.json carries `"type": "module"` so node treats the .js output as ESM.
COPY --from=build /repo/node_modules ./node_modules
COPY --from=build /repo/kun-love-ren ./kun-love-ren
COPY package.json ./
USER node
EXPOSE 3000
CMD ["node", "kun-love-ren/index.js"]
