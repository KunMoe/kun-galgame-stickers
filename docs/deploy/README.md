# 表情包站 · 部署指南（Docker + Dokploy）

把 **kun-galgame-stickers**（SvelteKit / adapter-node 的表情包站）作为**第 4 个 Dokploy 应用**接入鲲 Galgame 生态。它是一个**无状态 SSR 服务**，自己不拥有任何基础设施，运行时通过**服务名**复用枢纽（kun-galgame-infra）的 `postgres` 与 `oauth`。

> 枢纽 / kungal / moyu 的部署见 `kun-galgame-infra/docs/deploy/`。本站与它们**平级**，共享同一个 `dokploy-network`。Dokploy 安装、DNS、Traefik、网络等通用前提以那边的 [`12-dokploy.md`](../../../kun-galgame-infra/docs/deploy/12-dokploy.md) 为准，本文只讲本站特有的部分。

---

## 0 · 一句话与拓扑

> 一个 `web` 容器（`node kun-love-ren/index.js`，监听 `:3000`）。Traefik 按域名 `sticker.kungal.com` 回源到 `web:3000`；DB 走 `postgres:5432`；OAuth 走公网 `https://oauth.kungal.com/api/v1`。

```
Internet ─► :443 ─► Traefik(Dokploy) ─► sticker.kungal.com ─► web:3000  (本应用)
                                                               │
                  dokploy-network（与 infra 共享）             ├─► postgres:5432   (枢纽，新库 kungalgame_sticker)
                                                               └─► oauth (公网 https://oauth.kungal.com/api/v1)
```

| 项目         | 值                                                                                       |
| ------------ | ---------------------------------------------------------------------------------------- |
| 公网域名     | `sticker.kungal.com`                                                                     |
| 容器内端口   | `3000`（adapter-node，`HOST=0.0.0.0`）                                                   |
| 数据库       | 枢纽 Postgres 上的**第 6 个库** `kungalgame_sticker`                                     |
| OAuth client | `c5cd7b074804ba134934eb6c175a8f4d`（confidential，已注册）                               |
| 回调         | `https://sticker.kungal.com/auth/callback`、`http://127.0.0.1:5173/auth/callback`（dev） |

---

## 1 · 前置条件

- 枢纽 infra 已上线，`postgres` / `oauth` 健康（本站强依赖二者）。
- Dokploy 已装好，`dokploy-network` 存在（由 infra 应用创建，external 共享）。
- DNS：`sticker.kungal.com` 的 A 记录指向服务器公网 IP（Traefik 自动签发证书）。
- OAuth client 已注册（见 §0 表格）；**redirect_uris 已包含** `https://sticker.kungal.com/auth/callback`。

---

## 2 · 镜像（Dockerfile）

仓库根的 [`Dockerfile`](../../Dockerfile) 是多阶段构建（Node 24 + pnpm 9）：

```
base → deps(全量安装) → build(prisma generate + vite build + prune --prod) → run(node + 精简 node_modules + 产物)
```

产物落在 `kun-love-ren/`（见 `svelte.config.js` 的 `out`），运行入口 `node kun-love-ren/index.js`。

### 镜像怎么来：CI 构建 → GHCR → Dokploy 拉取（与 infra/forum/patch 一致）

**不在生产机上 build**（镜像重、单机 build 有拖垮风险）。流程：

```
push 到 svelte-kit ─► GitHub Actions 构建 ─► 推 ghcr.io/kunmoe/sticker-web:{latest, <sha>}
                                              │ 构建完成后触发 Dokploy webhook
                                              ▼
                          Dokploy ─► pull :latest ─► Traefik 零宕机滚动
```

- workflow：[`.github/workflows/build.yml`](../../.github/workflows/build.yml)（push `svelte-kit` 触发，单镜像，`GITHUB_TOKEN` 推 GHCR，GHA 层缓存）。
- 镜像名 `ghcr.io/kunmoe/sticker-web`，打两个 tag：`:latest`（移动标签，Dokploy 监听）+ `:<git-sha>`（回滚锚点）。
- prod compose 里 `web` 用 `image: …:latest` + **`pull_policy: always`**（强制每次部署重新拉 `:latest`，否则会跑本地旧缓存）。
- 运行时配置不烤进镜像（见 §3），所以**一个镜像通用于任何环境**。

> ⚠️ **务必关掉 Dokploy 这个 app 的 Auto Deploy**，并在仓库配 `DOKPLOY_WEBHOOK_STICKER` secret。否则 push 一到 Dokploy 就部署（不等 CI 构建完），`pull :latest` 拉到的是**上一次**的镜像。正确触发是 workflow 的 `deploy` job（`needs: build`，构建完才 `curl` webhook）。详见 infra 的 [`13-registry-ci.md`](../../../kun-galgame-infra/docs/deploy/13-registry-ci.md) §13.4。

### ⚠️ 关于 `prisma` 与镜像体积（务必理解）

- **`prisma` CLI 不是本项目依赖**（不在 `package.json`）。原因：Prisma 7 的 `@prisma/client` 把 `prisma` 声明为**可选 peer**；只要 `prisma` 出现在 workspace，lockfile 就会把它解析进来，连带 `studio-core / effect / pglite / engines / typescript …` 一棵 **~230MB** 的树被拖进**运行时镜像**。
- 因此构建时用 **`pnpm dlx prisma@7.8.0 generate`**（CLI 从临时缓存跑，不进 `node_modules`），运行时只保留 `@prisma/client + @prisma/adapter-pg + pg`。
- 效果：运行时 `node_modules` 约 **178MB**（而非 ~405MB）。
- **升级 `@prisma/client` 时**，同步把 Dockerfile 里 `dlx prisma@7.8.0` 的版本号一起改。
- `pnpm prisma:push` / `pnpm prisma:generate` 这两个 script 写的是 `pnpm prisma …`，**只有在本机存在 prisma 时才可用**；不存在时请直接用 `pnpm dlx prisma@7.8.0 …`（见 §4）。

### 镜像大小

当前镜像约 **1.3GB**，其中 **~345MB 是 `static/` 里的表情包图片**（被打进 `kun-love-ren/client/`）。等图片迁到对象存储后，镜像会回落到几百 MB。功能上不受影响，仅首版构建/拉取慢一些。

---

## 3 · 环境变量

运行时配置全部走 `$env/dynamic/private`（运行期注入，**不烤进镜像**）。在 Dokploy 应用的 **Environment** 面板填写：

| 变量                              | 生产值                                                                                        | 说明                                                                                |
| --------------------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `KUN_DATABASE_URL`                | `postgresql://postgres:${POSTGRES_PASSWORD}@postgres:5432/kungalgame_sticker?sslmode=disable` | 枢纽 Postgres + 新库；密码与 infra 一致                                             |
| `KUN_OAUTH_SERVER_URL`            | `https://oauth.kungal.com/api/v1`                                                             | **单 base**：浏览器跳 `/oauth/authorize` 与 SSR 取 token 都用它，所以必须是公网域名 |
| `KUN_OAUTH_WEB_URL`               | `https://oauth.kungal.com`                                                                    | 统一注册跳转（浏览器侧）                                                            |
| `KUN_OAUTH_REDIRECT_URI`          | `https://sticker.kungal.com/auth/callback`                                                    | 必须与已注册的 redirect_uri 完全一致                                                |
| `KUN_OAUTH_CLIENT_ID`             | `c5cd7b074804ba134934eb6c175a8f4d`                                                            | 非密，可内联                                                                        |
| `KUN_OAUTH_CLIENT_SECRET`         | （Dokploy 面板，勿入库）                                                                      | confidential secret                                                                 |
| `ORIGIN`                          | `https://sticker.kungal.com`                                                                  | adapter-node 在 Traefik 后做 CSRF origin 校验所需（否则 POST 表单如登出会 403）     |
| `PROTOCOL_HEADER` / `HOST_HEADER` | `x-forwarded-proto` / `x-forwarded-host`                                                      | Traefik 默认带，辅助还原绝对 URL                                                    |
| `PUBLIC_KUN_STICKER_THEME`        | `kun-sticker-theme`                                                                           | 非密；主题 cookie 名，**构建期**烤进客户端（compose 里作为 build arg）              |
| `NODE_ENV`/`HOST`/`PORT`          | `production`/`0.0.0.0`/`3000`                                                                 | 已在 Dockerfile 内置                                                                |

> 这些非密/结构性值，[`docker-compose.prod.yml`](../../docker-compose.prod.yml) 已替你写死；真正要在 Dokploy 面板填的只有 **`POSTGRES_PASSWORD`** 和 **`KUN_OAUTH_CLIENT_SECRET`** 两个密钥（compose 用 `${VAR:?}` 引用）。

---

## 4 · 一次性迁移（对**活集群**单独跑）

infra 的 `initdb` 建库脚本和 seed **只在首次初始化时跑过**，对已上线的集群**不会再执行**。所以下面三件事要手动对**正在运行的** Postgres 跑一次。

### 4.1 建库

```bash
# 在 Dokploy 的 infra 应用 Terminal（容器网络内，服务名 postgres）
docker compose -f docker-compose.prod.yml exec postgres \
  psql -U postgres -d kun_galgame_infra -c 'CREATE DATABASE kungalgame_sticker;'
```

> 建议同时把 `kungalgame_sticker` 补进 infra 的 `docker/initdb.d/01-create-databases.sh`，**仅为将来从零重建可复现**——对今天的活集群无效。

### 4.2 建表（schema）

本项目用 `prisma db push`（无 migrations 目录）。在**有仓库代码**的机器上（CI 或本地），指向新库跑一次：

```bash
KUN_DATABASE_URL='postgresql://postgres:<PASSWORD>@<POSTGRES_HOST>:5432/kungalgame_sticker' \
  pnpm dlx prisma@7.8.0 db push
```

> 运行时镜像里**没有 prisma CLI**，所以 push 不在运行容器里跑，而是用 `dlx` 从仓库/CI 跑。`<POSTGRES_HOST>` 在容器网络里是 `postgres`，从宿主则用映射端口。

### 4.3 灌数据

把现有 `kungalgame_sticker` 库的数据导入新库（schema 已由 4.2 建好，这里只迁数据）：

```bash
pg_dump --data-only --no-owner --table=sticker \
  'postgresql://<OLD_USER>:<OLD_PW>@<OLD_HOST>:5432/kungalgame_sticker' \
| psql 'postgresql://postgres:<PASSWORD>@<NEW_HOST>:5432/kungalgame_sticker'
```

校验：

```bash
psql 'postgresql://postgres:<PASSWORD>@<NEW_HOST>:5432/kungalgame_sticker' \
  -c 'SELECT count(*) FROM sticker;'
```

---

## 5 · OAuth client

client 已注册（`c5cd7b074804ba134934eb6c175a8f4d`），`redirect_uris` 已含生产与 dev 回调。无需再改库；只要把 **secret 填进 Dokploy 面板** 的 `KUN_OAUTH_CLIENT_SECRET` 即可。

> 单 base 说明：本站 `KUN_OAUTH_SERVER_URL` 同时用于浏览器 `/oauth/authorize` 跳转和 SSR 取 token，故必须是公网 `https://oauth.kungal.com/api/v1`（不能用内部 `oauth:9277`）。SSR 调用会出网到 Traefik 再回来，正确但非内部直连——可接受。

---

## 6 · Dokploy 部署步骤

1. **新建 Compose 应用** `kun-galgame-sticker`，源指向本仓库；compose 文件用 `docker-compose.prod.yml`（已是 `image:` + `pull_policy: always`，Dokploy 只拉不 build）。
2. **填 Environment**：`POSTGRES_PASSWORD`、`KUN_OAUTH_CLIENT_SECRET`（其余已在 compose 内联）。
3. **接好 CI 触发**（与 infra/forum/patch 一致，两步缺一不可）：
   - 复制本 app 的 Dokploy **部署 Webhook URL** → 填进仓库 Actions secret **`DOKPLOY_WEBHOOK_STICKER`**。
   - **关掉本 app 的 Auto Deploy**（消除 push 早触发的赛跑，见 §2 警告）。
   - GHCR：本镜像随仓库公开即可免凭证拉；私有则在 Dokploy Settings → Registry 配 `read:packages` 的 PAT。
4. **跑 §4 的一次性迁移**（建库 / push / 灌数据）。
5. **首次部署**：`push` 到 `svelte-kit`（或 Actions 手动 `workflow_dispatch`）→ CI 构建推 GHCR → `deploy` job 触发 webhook → Dokploy 拉 `:latest` 起 `web`，等 healthy。
6. **配域名**：在应用的 **Domains** 加 `sticker.kungal.com`，路径 `/`，目标 `web` 内部端口 `3000`，Dokploy 自动注入 Traefik labels + 签证书。
7. **验证**（见 §7）。

> 共享网络：compose 已把 `default` 网络设为 external 的 `dokploy-network`。不要发布宿主端口（`expose: 3000` 即可，Traefik 内部回源）。
>
> **回滚**：把 compose 的 `image:` 临时 pin 到某个 `ghcr.io/kunmoe/sticker-web:<git-sha>` 再 redeploy。

---

## 7 · 验证 / 烟雾测试

```bash
curl -I https://sticker.kungal.com/                 # 200，有效证书
curl -s https://sticker.kungal.com/sticker/1 | grep -o '<img'   # DB 走通：渲染出表情包卡片
curl -I https://sticker.kungal.com/sitemap.xml      # 200（运行时生成，非预渲染）
```

- 登录链路：点站内登录 → 跳 `https://oauth.kungal.com/oauth/authorize` → 授权后回 `/auth/callback` → 落地。
- 容器日志应有 `Listening on http://0.0.0.0:3000`，无 prisma / 连接错误。

---

## 8 · 本地 Docker 自测（可选）

```bash
# 构建
docker build -t sticker:local .

# 临时 Postgres + 建表 + 跑容器
docker network create stickernet
docker run -d --name pg --network stickernet -p 55432:5432 \
  -e POSTGRES_PASSWORD=test -e POSTGRES_DB=kungalgame_sticker postgres:18-alpine
KUN_DATABASE_URL='postgresql://postgres:test@localhost:55432/kungalgame_sticker' pnpm dlx prisma@7.8.0 db push

docker run --rm --network stickernet -p 3001:3000 \
  -e KUN_DATABASE_URL='postgresql://postgres:test@pg:5432/kungalgame_sticker?sslmode=disable' \
  -e KUN_OAUTH_SERVER_URL='http://dummy/api/v1' -e KUN_OAUTH_WEB_URL='http://dummy' \
  -e KUN_OAUTH_REDIRECT_URI='http://localhost:3001/auth/callback' \
  -e KUN_OAUTH_CLIENT_ID='dummy' -e KUN_OAUTH_CLIENT_SECRET='dummy' \
  -e ORIGIN='http://localhost:3001' \
  sticker:local
# curl localhost:3001/ 和 /sticker/<已灌数据的 sid>
```

清理：`docker rm -f pg && docker network rm stickernet`。

---

## 9 · 运维 / 取舍

- **日志 / 重部署 / 回滚**：用 Dokploy 面板；预构建镜像模式下回滚 = 切回上一个 GHCR tag。
- **Schema 变更**：改 `prisma/schema.prisma` 后，对目标库重新 `pnpm dlx prisma@7.8.0 db push`（本项目无 migrations，是 push 流）。
- **证书 / 反代**：Traefik 托管，勿叠加 Caddy/nginx/CF Tunnel。
- **图片体积**：见 §2；迁对象存储后 `static/` 瘦身，镜像随之变小，是后续独立事项。

---

## 关联文件

- [`Dockerfile`](../../Dockerfile) · [`.dockerignore`](../../.dockerignore) · [`docker-compose.prod.yml`](../../docker-compose.prod.yml)
- [`.env.example`](../../.env.example) — 完整环境变量样板
- infra 部署文档：`../../../kun-galgame-infra/docs/deploy/`（Dokploy、网络、CI、备份）
