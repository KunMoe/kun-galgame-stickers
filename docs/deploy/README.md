# 表情包站 · 部署指南（Docker + Dokploy）

> 站点是 **Nuxt 4 前端 + Go Fiber API**（与 forum / patch 同构）。生产 compose 是 `web` + `sticker-api` + 一次性 `migrate`。每次 Dokploy `up` 会先跑 migrate（`sticker-api` `depends_on: service_completed_successfully`），和 kungal / moyu / infra 一样，不用手工 migrate。

把 **kun-galgame-stickers** 作为 Dokploy 应用接入鲲 Galgame 生态。它自己不拥有任何基础设施，运行时通过**服务名**复用枢纽（nextmoe-infra）的 `postgres` 与 `oauth`。

> 枢纽 / kungal / moyu 的部署见 `nextmoe-infra/docs/deploy/`。本站与它们**平级**，共享同一个 `dokploy-network`。Dokploy 安装、DNS、Traefik、网络等通用前提以那边的 [`12-dokploy.md`](../../../nextmoe-infra/docs/deploy/12-dokploy.md) 为准，本文只讲本站特有的部分。

仓库：[`KunMoe/kun-galgame-stickers`](https://github.com/KunMoe/kun-galgame-stickers)（从 `KUN1007/kun-galgame-stickers-sveltekit` 迁过来）。

---

## 0 · 一句话与拓扑

> `migrate`（一次性）→ `sticker-api:9421` → `web:3000`。Traefik 按域名 `sticker.kungal.com` 回源到 `web:3000`，`/api` 回源到 `sticker-api:9421`；DB 走 `postgres:5432`；OAuth token 走内部 `oauth:9277`。

```
Internet ─► :443 ─► Traefik(Dokploy) ─► sticker.kungal.com      ─► web:3000
                                    └─► sticker.kungal.com/api ─► sticker-api:9421
                                                                      │
                  dokploy-network（与 infra 共享）                   ├─► postgres:5432  (kungalgame_sticker)
                                                                      └─► oauth:9277
```

| 项目         | 值                                                                                       |
| ------------ | ---------------------------------------------------------------------------------------- |
| 公网域名     | `sticker.kungal.com`                                                                     |
| 容器         | `web:3000`（Nitro）、`sticker-api:9421`（Fiber）、`migrate`（一次性）                     |
| 数据库       | 枢纽 Postgres 上的 `kungalgame_sticker`                                                  |
| OAuth client | `c5cd7b074804ba134934eb6c175a8f4d`（confidential，已注册）                               |
| 回调         | `https://sticker.kungal.com/auth/callback`、`http://127.0.0.1:5173/auth/callback`（dev） |

共享 `dokploy-network` 上服务名必须唯一，所以 API 叫 **`sticker-api`**，不要用裸名 `api`（会和 kungal-api / moyu-api 抢 DNS）。

---

## 1 · 前置条件

- 枢纽 infra 已上线，`postgres` / `oauth` 健康（本站强依赖二者）。
- Dokploy 已装好，`dokploy-network` 存在（由 infra 应用创建，external 共享）。
- DNS：`sticker.kungal.com` 的 A 记录指向服务器公网 IP（Traefik 自动签发证书）。
- OAuth client 已注册（见 §0 表格）；**redirect_uris 已包含** `https://sticker.kungal.com/auth/callback`。
- Dokploy 上本应用有两条域名路由：`/` → `web:3000`，`/api`（更高优先级的 PathPrefix）→ `sticker-api:9421`。

---

## 2 · 镜像

三份 Dockerfile 产物，**不在生产机上 build**（镜像重、单机 build 有拖垮风险）。

| 镜像                              | Dockerfile                 | 入口                                      |
| --------------------------------- | -------------------------- | ----------------------------------------- |
| `ghcr.io/kunmoe/sticker-web`      | [`docker/nuxt.Dockerfile`](../../docker/nuxt.Dockerfile) | `node .output/server/index.mjs`（Nitro） |
| `ghcr.io/kunmoe/sticker-api`      | [`docker/go.Dockerfile`](../../docker/go.Dockerfile) `CMD=api` | `/app`                                   |
| `ghcr.io/kunmoe/sticker-migrate`  | [`docker/go.Dockerfile`](../../docker/go.Dockerfile) `CMD=migrate` | `/app`，工作目录 `/`，默认 `-dir up`     |

```
push 到 svelte-kit ─► GitHub Actions 构建三镜像 ─► 推 ghcr.io/kunmoe/sticker-{web,api,migrate}:{latest, <sha>}
                                              │ 构建完成后触发 Dokploy webhook
                                              ▼
                          Dokploy ─► pull :latest ─► migrate 跑完 ─► sticker-api healthy ─► web
```

- workflow：[`.github/workflows/build.yml`](../../.github/workflows/build.yml)（push `svelte-kit` 触发，`GITHUB_TOKEN` 推 GHCR，GHA 层缓存）。
- 每个镜像打两个 tag：`:latest`（Dokploy 监听）+ `:<git-sha>`（回滚锚点）。
- prod compose 里 `pull_policy: always`（强制每次部署重新拉 `:latest`，否则会跑本地旧缓存）。
- 运行时配置不烤进镜像（见 §3），所以**一个镜像通用于任何环境**。

> ⚠️ **务必关掉 Dokploy 这个 app 的 Auto Deploy**，并在仓库配 `DOKPLOY_WEBHOOK_STICKER` secret。否则 push 一到 Dokploy 就部署（不等 CI 构建完），`pull :latest` 拉到的是**上一次**的镜像。正确触发是 workflow 的 `deploy` job（`needs: build`，构建完才 `curl` webhook）。详见 infra 的 [`13-registry-ci.md`](../../../nextmoe-infra/docs/deploy/13-registry-ci.md) §13.4。

### 镜像体积

`sticker-web` 目前偏大，因为表情图还在 `apps/web/public/`（列表 webp + telegram PNG 原图），会被打进 Nitro 产物。迁到 infra image service 之后镜像会明显变小。功能不受影响，只是构建 / 拉取慢一些。

---

## 3 · 环境变量

运行时配置全部走容器 `environment`（**不烤进镜像**）。[`docker-compose.prod.yml`](../../docker-compose.prod.yml) 已把非密值写死；Dokploy Environment 面板真正要填的只有：

| 变量                       | 说明                                      |
| -------------------------- | ----------------------------------------- |
| `POSTGRES_PASSWORD`        | 必须与 infra Postgres 一致                |
| `KUN_OAUTH_CLIENT_SECRET`  | 表情包站 confidential client secret       |

compose 里已经固定的值（不要在面板里改掉语义）：

| 变量                              | 生产值                                      | 谁读                         |
| --------------------------------- | ------------------------------------------- | ---------------------------- |
| `KUN_DATABASE_URL`                | `postgresql://postgres:…@postgres:5432/kungalgame_sticker?sslmode=disable` | API / migrate                |
| `KUN_OAUTH_SERVER_URL`            | `http://oauth:9277/api/v1`                  | **仅 API**（容器内直连 OP）  |
| `KUN_OAUTH_WEB_URL`               | `https://oauth.kungal.com`                  | API（注册跳转用）            |
| `KUN_OAUTH_REDIRECT_URI`          | `https://sticker.kungal.com/auth/callback`  | API 换 token                 |
| `KUN_OAUTH_CLIENT_ID`             | `c5cd7b074804ba134934eb6c175a8f4d`          | API                          |
| `CORS_ALLOW_ORIGINS`              | `https://sticker.kungal.com`                | API                          |
| `NUXT_API_BASE_URL`               | `http://sticker-api:9421`                   | **web SSR**（容器内打 API）  |
| `NUXT_PUBLIC_API_BASE_URL`        | `https://sticker.kungal.com`                | 浏览器经 Traefik `/api`      |
| `NUXT_PUBLIC_SITE_URL`            | `https://sticker.kungal.com`                | canonical / sitemap / og     |
| `NUXT_PUBLIC_OAUTH_SERVER_URL`    | `https://oauth.kungal.com/api/v1`           | **浏览器** authorize / logout |
| `NUXT_PUBLIC_OAUTH_FRONTEND_URL`  | `https://oauth.kungal.com`                  | 注册页                       |
| `NUXT_PUBLIC_OAUTH_REDIRECT_URI`  | `https://sticker.kungal.com/auth/callback`  | PKCE 回调                    |

> 浏览器 OAuth 必须走公网 `https://oauth.kungal.com`；API 换 token 走内部 `oauth:9277`。这是 Nuxt + Fiber 拆开之后的分工，不再需要 SvelteKit 时代的 `ORIGIN` / `PROTOCOL_HEADER`。

---

## 4 · Schema 迁移

生产 compose **每次 `up` 都会跑** `sticker-migrate`，成功后才起 `sticker-api`。SQL 在 `apps/api/migrations/`，工具是 golang-migrate（不是 Prisma）。

`000001_baseline` 用 `CREATE TABLE IF NOT EXISTS`：现网 `kungalgame_sticker.sticker` 不会被重建，第一次跑只是把版本记进 `schema_migrations`。

手工（一般不需要）：

```bash
docker compose -f docker-compose.prod.yml run --rm migrate          # 默认 up
docker compose -f docker-compose.prod.yml run --rm migrate -dir down
```

库本身由 infra 的 Postgres 托管。活集群上 `kungalgame_sticker` 已存在、数据已在；新环境从零建库才需要：

```bash
# 在 Dokploy 的 infra 应用 Terminal
docker compose -f docker-compose.prod.yml exec postgres \
  psql -U postgres -d kun_galgame_infra -c 'CREATE DATABASE kungalgame_sticker;'
```

> 建议把 `kungalgame_sticker` 留在 infra 的 `docker/initdb.d/01-create-databases.sh`，**仅为将来从零重建可复现**——对今天的活集群无效。

改 schema 时：加 `apps/api/migrations/00000N_*.sql`，随镜像发布；**不要**再跑 `prisma db push`。

---

## 5 · OAuth client

client 已注册（`c5cd7b074804ba134934eb6c175a8f4d`），`redirect_uris` 已含生产与 dev 回调。无需再改库；只要把 **secret 填进 Dokploy 面板** 的 `KUN_OAUTH_CLIENT_SECRET` 即可。

---

## 6 · Dokploy 部署步骤

1. **Compose 应用** `kun-galgame-sticker`，compose 文件用 `docker-compose.prod.yml`（已是 `image:` + `pull_policy: always`，Dokploy 只拉不 build）。
2. **填 Environment**：`POSTGRES_PASSWORD`、`KUN_OAUTH_CLIENT_SECRET`。
3. **接好 CI 触发**（两步缺一不可）：
   - 复制本 app 的 Dokploy **部署 Webhook URL** → 填进仓库 Actions secret **`DOKPLOY_WEBHOOK_STICKER`**。
   - **关掉本 app 的 Auto Deploy**（消除 push 早触发的赛跑，见 §2 警告）。
   - GHCR：本镜像随仓库公开即可免凭证拉；私有则在 Dokploy Settings → Registry 配 `read:packages` 的 PAT。
4. **配域名**（两条，顺序 / 优先级很重要）：
   - `sticker.kungal.com` 路径 `/` → 服务 `web` 端口 `3000`
   - `sticker.kungal.com` 路径 `/api` → 服务 **`sticker-api`** 端口 `9421`（PathPrefix，高于 `/`）
5. **首次 / 之后部署**：`push` 到 `svelte-kit`（或 Actions 手动 `workflow_dispatch`）→ CI 构建推 GHCR → `deploy` job 触发 webhook → Dokploy 拉 `:latest`，`migrate` → `sticker-api` → `web`。
6. **验证**（见 §7）。

> 共享网络：compose 已把 `default` 网络设为 external 的 `dokploy-network`。不要发布宿主端口（`expose` 即可，Traefik 内部回源）。
>
> **回滚**：把 compose 的 `image:` 临时 pin 到某个 `ghcr.io/kunmoe/sticker-{web,api,migrate}:<git-sha>` 再 redeploy。三个镜像尽量 pin 同一 sha。

---

## 7 · 验证 / 烟雾测试

```bash
curl -I https://sticker.kungal.com/                      # 200，有效证书
curl -I https://sticker.kungal.com/en                    # 200
curl -I https://sticker.kungal.com/ja/about              # 200
curl -s https://sticker.kungal.com/api/v1/sticker/packs  # code === 0，7 套
curl -I https://sticker.kungal.com/sticker/7             # 200
curl -I https://sticker.kungal.com/sitemap.xml           # 307 → /sitemap_index.xml（按语言拆成 zh-CN / en-US / ja-JP）
curl -I https://sticker.kungal.com/sitemap_index.xml     # 200（运行时生成，含 hreflang）
curl -I https://sticker.kungal.com/robots.txt            # 200，Sitemap 指向 sitemap_index.xml
curl -s https://sticker.kungal.com/api/healthz           # 若 Traefik 把 /api 全打到 API；否则 web 无此路由
```

- 登录：点站内登录 → 跳 `https://oauth.kungal.com/oauth/authorize` → 授权后回 `/auth/callback` → 落地。
- `sticker-api` 日志应有 Fiber 监听 `:9421`；`web` 是 Nitro `:3000`。不要再找 Prisma / `kun-love-ren`。

---

## 8 · 本地开发（推荐）与本地 Docker（可选）

日常开发不走生产 compose：

```bash
cp .env.example .env   # 填 KUN_DATABASE_URL 和 OAuth secret
pnpm install
pnpm migrate           # 首次，针对 kungalgame_sticker
pnpm dev               # web http://127.0.0.1:5173  ·  api :9421
```

本地 Docker 只用来验镜像：

```bash
docker build -f docker/go.Dockerfile --build-arg CMD=api -t sticker-api:local .
docker build -f docker/nuxt.Dockerfile -t sticker-web:local .
```

生产拓扑依赖 `dokploy-network` 上的 `postgres` / `oauth`，单机 compose 不能原样 `up`。

---

## 9 · 运维 / 取舍

- **日志 / 重部署 / 回滚**：用 Dokploy 面板；预构建镜像模式下回滚 = 三个 GHCR tag 一起切回。
- **Schema 变更**：加 `apps/api/migrations/` 里的 SQL，随 `sticker-migrate` 镜像发布。上线后 compose `up` 会自动 apply。记得在任务结束时明确说生产库要不要 sync。
- **证书 / 反代**：Traefik 托管，勿叠加 Caddy/nginx/CF Tunnel。
- **图片体积**：见 §2；迁 infra image service 后从 `apps/web/public/stickers/` 和 telegram 原图瘦身，是后续独立事项。

---

## 关联文件

- [`docker/nuxt.Dockerfile`](../../docker/nuxt.Dockerfile) · [`docker/go.Dockerfile`](../../docker/go.Dockerfile) · [`.dockerignore`](../../.dockerignore) · [`docker-compose.prod.yml`](../../docker-compose.prod.yml)
- [`.env.example`](../../.env.example) — 完整环境变量样板
- infra 部署文档：`../../../nextmoe-infra/docs/deploy/`（Dokploy、网络、CI、备份）
