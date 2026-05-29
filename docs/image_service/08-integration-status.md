# 08 — 调用方接入现状快照

> **截止日期**：2026-05-26
> **目的**：给出 image_service 在各上游项目中**实际被使用到什么程度**的真实图景，避免"我以为 X 接了 / Y 没接"这种判断错位。
> **维护方式**：本文件是 point-in-time 快照，非长期文档。重大接入变更后请回头重抓 file:line 引用并刷新表格。

## TL;DR

| 业务域 | 接入状态 | 说明 |
|---|---|---|
| galgame banner 上传（创建/编辑/PR/rewrite） | ✅ 全链路 | wiki 主接入点；kungal/moyu 都已切到 multipart → wiki |
| galgame cover / screenshot 上传 | ✅ 已接 | kungal 走 `/image/galgame` 代理；moyu 走 `/upload/image-service` 代理；都拿 hash 写 covers[]/screenshots[] |
| 头像上传 | ✅ 已接 | kungal/moyu 都代理到 OAuth `/auth/me/avatar` |
| 图片保活（refping） | ✅ 已接 | oauth/wiki 侧 `galgame_image_refping` 扫 cover/screenshot/revision/PR snapshot |
| **头像「展示」** | ⚠️ **不完整** | `KunAvatar` 不读 `avatar_image_hash`；kungal DTO 几乎都没透传该字段 |
| **JSON 写入时的 hash 探活** | ⚠️ **缺失** | wiki 接受 64 位假 hash 写入，跑到展示时才发现死图 |
| kungal 论坛 topic / Markdown 图片 | ❌ 未接 | 仍走旧 S3 + 自家 daily-quota |
| 文档 banner / 网站 icon / 广告图 / 友链 / 静态 MDX 文章图 | ❌ 未接 | 静态资源，按设计不需要接 |
| patch 资源文件（压缩包/补丁包） | ❌ 不需要接 | image_service 只管图片 |

## 已接入：详细引用

### A. galgame banner / cover / screenshot 主链路

**oauth-admin (wiki) 是 galgame 图片的核心接入点。** 所有 galgame 图片的最终去向都是 image_service，但具体调用路径分两段：

#### A1. wiki 自身的 multipart 接入

multipart 请求里如果带 `file`，wiki 用 `preset=galgame_banner` 上传到 image_service，把返回的 hash 注入 `galgame_cover(sort_order=0)`（=当前 banner）。

- 多部分解析 + image_service 调用：[`apps/api/internal/platform/galgame/handler/multipart_helper.go:28`](../../apps/api/internal/platform/galgame/handler/multipart_helper.go#L28)
- promote 成 sort_order=0 cover：[`apps/api/internal/platform/galgame/service/galgame_service.go:221`](../../apps/api/internal/platform/galgame/service/galgame_service.go#L221)

适用接口：`POST /galgame`（创建）、`PUT /galgame/:id`（编辑）、`POST /galgame/:id/pr`（PR）、`POST /galgame/submit`（公开提交，被 moyu 直连）。

#### A2. kungal 侧的代理 + 编辑器

kungal 不只在「创建时」用 image_service，编辑器的 cover/screenshot 拖拽上传也走它：

- 后端代理路由：[`kun-galgame-nuxt4/apps/api/internal/app/router.go:234`](../../../kun-galgame-nuxt4/apps/api/internal/app/router.go#L234) — `POST /image/galgame` 转发到 image_service
- 前端 cover 编辑器：[`kun-galgame-nuxt4/apps/web/app/components/edit/galgame/pr/Covers.vue:56`](../../../kun-galgame-nuxt4/apps/web/app/components/edit/galgame/pr/Covers.vue#L56) — 拿 hash 后写入 `covers[]`
- 前端 screenshot 编辑器：[`kun-galgame-nuxt4/apps/web/app/components/edit/galgame/pr/Screenshots.vue:48`](../../../kun-galgame-nuxt4/apps/web/app/components/edit/galgame/pr/Screenshots.vue#L48) — 拿 hash 后写入 `screenshots[]`

**preset 选择注**：kungal 的 screenshot 编辑器**复用 `galgame_banner` preset**（不是自己的 `galgame_screenshot`，因为 image_presets.yaml 里没有这个 preset）。main pipeline 的 1920×1080 fit 对 screenshot 也足够，多出来的 `mini` 变体不被 gallery 渲染但无害。见 `Screenshots.vue:48` 的 AUDIT FIX 注释。

#### A3. moyu 侧的代理 + 编辑器

moyu 走两条不同链路：
- **banner**（创建/rewrite/PR 的封面）→ multipart 直连 wiki，wiki 自己上传到 image_service
- **screenshot** → moyu 自己的 `/upload/image-service` 代理（不经过 wiki）

引用：
- moyu 创建 galgame 的 banner multipart：[`apps/web/app/pages/edit/create.vue:246`](../../../kun-galgame-patch-next/apps/web/app/pages/edit/create.vue#L246) → `POST /galgame/submit`
- moyu rewrite 的 banner multipart：[`apps/web/app/pages/edit/rewrite.vue:325`](../../../kun-galgame-patch-next/apps/web/app/pages/edit/rewrite.vue#L325) → `PUT /galgame/:id`
- moyu screenshot 编辑器：[`apps/web/app/components/galgame/edit/ScreenshotsEditor.vue:33`](../../../kun-galgame-patch-next/apps/web/app/components/galgame/edit/ScreenshotsEditor.vue#L33) — 调 `ge.uploadImageService(f, 'topic')`
- moyu 后端代理：[`apps/api/internal/app/router.go:313`](../../../kun-galgame-patch-next/apps/api/internal/app/router.go#L313) — `POST /upload/image-service`
- 代理实现：[`apps/api/internal/common/upload/handler.go:109`](../../../kun-galgame-patch-next/apps/api/internal/common/upload/handler.go#L109)

**preset 不一致警告**：moyu screenshot 用 `'topic'` preset（不产 mini 变体），kungal screenshot 用 `'galgame_banner'`（产但不用 mini）。两边对"截图"这一同类资产用了不同 preset，应该统一 — 后续要么加一个 `galgame_screenshot` preset 显式表达，要么两边都迁到同一个 preset。

### B. 头像上传

kungal 和 moyu 都不自己上传头像，都代理到 OAuth：

- OAuth 侧统一入口：`POST /auth/me/avatar`（multipart）→ image_service `preset=avatar`
- 上游代理转发后从 OAuth 拿 hash 并写入自家 user 表的 `avatar_image_hash` 列

### C. 图片保活（reference-ping）

image_service 用 `last_referenced_at` + TTL 做软清理（见 README 决策 0），调用方需要定期 ping 自己用到的 hash 集。但 **galgame 系列的保活不是 kungal/moyu 做的，是 oauth/wiki 侧的 cron job 做的**：

- [`apps/api/internal/jobs/galgame_image_refping.go:51`](../../apps/api/internal/jobs/galgame_image_refping.go#L51)

它扫的 hash 来源：
1. 当前 `galgame_cover.image_hash`
2. 当前 `galgame_screenshot.image_hash`
3. 全部 `galgame_revision.snapshot` 中出现过的 hash（历史 revert / diff 不能让 TTL 复活失败）
4. 全部 `galgame_pr.snapshot` 中的 hash（待合并的 PR）

> ⚠️ 这意味着：上游项目自己 DB 里如果还存了 galgame 图片的 hash（比如 moyu patch 的 `cover_image_hash`），**oauth/wiki 的 refping 不会扫到它**。如果上游有这种独立持有的 hash，必须自己加一个独立的 refping job。

## 未接入 / 不完整：详细引用

### D. 头像「展示」缺口（⚠️ 真实问题）

**问题**：上传链路通了，展示链路没接完。

#### D1. `KunAvatar` 组件不识别 `avatar_image_hash`

[`packages/ui/app/components/kun/avatar/Avatar.vue`](../../packages/ui/app/components/kun/avatar/Avatar.vue) 的 `userAvatarSrc` 实现：

```ts
const userAvatarSrc = computed(() => {
  const user = props.user
  if (user?.avatar) {
    return props.size === 'original' || props.size === 'original-sm'
      ? user.avatar
      : user.avatar.replace(/\.webp$/, '-100.webp')
  }
  return getRandomSticker(user?.name ?? '').value
})
```

- 只读 `user.avatar`（legacy CDN URL 字符串），完全忽略 `avatar_image_hash`
- `-100.webp` 后缀是旧 CDN 命名约定，不是 image_service 的 variant URL pattern（image_service 用 `_{variant}` 而不是 `-{variant}`）
- 结果：即使后端把 `avatar_image_hash` 透传上来，前端也读不出 mini 头像，只能拿旧 legacy URL

#### D2. DTO 透传不齐

- **moyu 已透传**：chat / admin / auth DTO 都带 `AvatarImageHash` 字段（[`apps/api/internal/auth/dto/dto.go:32`](../../../kun-galgame-patch-next/apps/api/internal/auth/dto/dto.go#L32) 等多处）
- **kungal 完全没透传**：grep `kun-galgame-nuxt4/apps/api/internal/` 查不到任何 `avatar_image_hash` / `AvatarImageHash` 字段，全部 DTO 仍只返回 `avatar` URL 字符串

#### D3. `resolveAvatarUrl()` 是死代码

`apps/web/shared/utils/resolveImage.ts` 和 `apps/wiki/app/shared/utils/resolveImage.ts` 都定义了 `resolveAvatarUrl(user, opts)`，但**没有任何头像渲染组件使用它** — 因为通用 `KunAvatar` 走自己的 legacy URL 拼接。

**修复路径**（建议作为单独 PR）：
1. `KunAvatar` 加 `avatarImageHash` prop（或者类型扩展 `props.user`）
2. `userAvatarSrc` 优先级：`avatar_image_hash` → `avatar` URL → sticker fallback
3. kungal DTO 补 `AvatarImageHash` 字段透传
4. 全站搜 `<KunAvatar :user=` 确保所有调用点的 user 对象都带 hash

### E. JSON 写入路径的 hash 验证缺口（⚠️ 真实问题）

[`apps/api/internal/platform/galgame/dto/galgame_dto.go:63`](../../apps/api/internal/platform/galgame/dto/galgame_dto.go#L63) 和 `:74`：

```go
ImageHash string `json:"image_hash" validate:"required,len=64"`
```

只校验长度 = 64。

- **multipart 路径**是可信的，因为 hash 是 image_service 自己返回的
- **JSON 路径**任何 API 调用者可以提交 `"a"*64` 当 hash，wiki 接受写入 `covers[]/screenshots[]`，展示时才发现 404 死图

**修复路径**：在 `CreateGalgameRequest` / `UpdateGalgameRequest` 的 service 层验证阶段，对 `Covers[].ImageHash` 和 `Screenshots[].ImageHash` 调用 `imageclient.HeadByHash(ctx, hash)`（或新增一个 batch endpoint `POST /image/exists` 一次性查多个 hash 减少 N 次往返）。失败 → reject 整个写入。

### F. 已知不接（按设计）

- **kungal 论坛 topic / Markdown 图片**：[`kun-galgame-nuxt4/apps/api/internal/image/service/image_service.go:45`](../../../kun-galgame-nuxt4/apps/api/internal/image/service/image_service.go#L45) `UploadTopicImage` 把图片重编码为 PNG 直传 S3，不走 image_service。  
  原因：legacy 历史负担、用户日配额逻辑深度耦合自家 DB；切换需要单独迁移计划。
- **patch 资源文件（zip/补丁包等）**：image_service 设计只管图片（webp/jpeg/png/gif），不接通用文件存储。
- **静态资源**（doc banner、网站 icon、广告图、友链、静态 MDX 文章图）：这些是 build-time 资产，不需要走运行时图片服务。

## Follow-up 短板清单（按优先级）

1. **【P0】头像展示链路收口**（D 节）— KunAvatar 加 hash 支持 + kungal DTO 透传补齐
2. **【P1】JSON 写入路径 hash 探活**（E 节）— 避免死图
3. **【P2】screenshot preset 统一**（A2/A3）— 加 `galgame_screenshot` preset 或让 kungal/moyu 用同一个名字
4. **【P3】kungal topic 图片迁移到 image_service**（F）— 大工程，单独立项

## 历史 / 相关文档

- [01-design.md](./01-design.md) — 总体架构与决策
- [04-migration-plan.md](./04-migration-plan.md) — 从旧体系迁移的分阶段计划
- [06-integration-guide.md](./06-integration-guide.md) — 调用方视角接入 checklist
