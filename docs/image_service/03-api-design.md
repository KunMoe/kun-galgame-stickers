# 03 — API 设计

## 基础约定

- **Base URL**：`https://image.api.example.com`（生产）/ `http://127.0.0.1:9278`（开发）
- **Content-Type**：
  - 上传：`multipart/form-data`
  - 其他：`application/json`
- **鉴权**：`Authorization: Bearer <access_token>`
  - V1：**仅支持**后端 OAuth Client Credentials
  - V2（按需）：前端用户 JWT 直传
- **Scope 要求**：
  - 上传：`image:upload`
  - 元信息查询：`image:read`
  - 管理（V3）：`image:admin`

## 错误响应

> ⚠️ 本节已对齐实现（2026-05 核对）。早期草案写的是嵌套
> `{"error":{"code":"quota_exceeded"}}`（字符串 code）——**那是过期设计，
> 从未落地**。实际服务端走全系统统一的扁平信封 `pkg/response`，`code` 是
> **整数**。`pkg/imageclient` 也按整数 `code` 解析/分类，二者一致。

统一格式（与 OAuth / galgame 等所有本仓服务同一套 `{code,message,data}`）：

```json
{
  "code": 80008,
  "message": "当日配额已用完",
  "details": {
    "quota_count": 10000,
    "quota_bytes": 10737418240,
    "used_count": 10000,
    "used_bytes": 10737418240,
    "reset_at": "2026-04-24T00:00:00Z"
  }
}
```

- 成功：`{"code":0,"message":"...","data":{...}}`。
- 普通错误：`{"code":<非0整数>,"message":"..."}`（无 `details`）。
- 配额超限是唯一带 `details` 的错误（`handler.go` 特例，字段如上）。
- HTTP 状态码与 `code` 并存，调用方应以**整数 `code`** 为准做分类。

| HTTP | code | 常量 | 场景 |
|------|------|------|------|
| 400 | 80014 | `ErrImageBadRequest` | 请求格式错误（缺 file/参数等） |
| 400 | 80011 | `ErrImagePresetNotFound` | 未知 preset |
| 400 | 80009 | `ErrImageMIMEDenied` | 该 preset 不接受此格式 |
| 400 | 80010 | `ErrImageDecodeFailed` | 图片解码失败 |
| 401 | 80001 | `ErrImageUnauthorized` | 缺失/无效凭证 |
| 401 | 80002 / 80003 | `ErrImageBadClient` / `ErrImageBadSecret` | client id / secret 错误 |
| 403 | 80004 / 80005 | `ErrImageSiteDisabled` / `ErrImageSiteUnconfigured` | 站点未开启/缺配置 |
| 403 | 80006 | `ErrImagePresetDenied` | 站点不允许此 preset |
| 413 | 80007 | `ErrImageFileTooLarge` | 超过站点上限 |
| 422 | 60002 | `ErrModerationRejected` | 审核拒绝（仅启用审核时） |
| 429 | 80008 | `ErrImageQuotaExceeded` | 超出站点日配额（带 `details`） |
| 500 | 80012 | `ErrImageStoreFailed` | 存储/处理失败 |
| 503 | 80015 | `ErrImageUploadDisabled` | 上传功能未开放 |

完整码表见 `apps/api/pkg/errors/codes.go`（`ErrImage*` 80001-80015、
`ErrModeration*` 60001-60003）。

**注意**：是否启用审核由部署开关决定；未启用时不会出现 `60002`。

---

## 1. 上传图片

### `POST /image/upload`

上传一张图片，同步处理（压缩主图 + 按 preset 生成变体）并返回永久 URL。

**请求**：

```http
POST /image/upload HTTP/1.1
Host: image.api.example.com
Authorization: Bearer <token>
Content-Type: multipart/form-data; boundary=----xxx

------xxx
Content-Disposition: form-data; name="file"; filename="photo.png"
Content-Type: image/png

<binary data>
------xxx
Content-Disposition: form-data; name="preset"

avatar
------xxx--
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file` | file | ✅ | 图片二进制，支持 `image/jpeg` `image/png` `image/webp` `image/gif`（首帧） |
| `preset` | string | ✅ | 处理预设名（`avatar` / `galgame_banner` / `topic`），必须在本站 `image_allowed_presets` 中 |

> **注意**：V1 **不接受** `keep_original` 参数（主图永远是 `webp@82 fit 1920×1080`）。未来如需保留原图，新加 preset 或字段。

#### Body 格式：只支持 `multipart/form-data`

V1 **仅接受** `multipart/form-data`，不支持 raw body + `Content-Type: image/*` + header 传 preset 的变体形式。即使调用方是 Server-to-Server 转发（拿到前端 FormData 再转到 image_service），也请保持 multipart。

理由：
- 只有一种 content-type 约定可以让服务端代码和测试大幅简化
- multipart 解析 1 次的成本几乎可忽略（libvips decode 远比它贵）
- 统一 error handling、统一日志格式

如未来 S2S 转发出现性能瓶颈（目前不存在），再评估加 raw body 通道。

**成功响应**：

```json
{
  "hash": "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ef",
  "url": "https://cdn.example.com/img/ab/cd/abcd...ef.webp",
  "width": 512,
  "height": 512,
  "size_bytes": 45678,
  "variant_urls": {
    "256": "https://cdn.example.com/img/ab/cd/abcd...ef_256.webp",
    "100": "https://cdn.example.com/img/ab/cd/abcd...ef_100.webp"
  },
  "deduplicated": false
}
```

**字段说明**：
- `hash` — 内容 hash，调用方存入自己库（`users.avatar_image_hash`）作为外键
- `url` — 主图 CDN 永久 URL，可直接用于 `<img src>`
- `variant_urls` — 按本次 preset 生成的变体清单。**字段里列出的都是实际落盘的静态 URL**，调用方直接使用
  - `preset=avatar` → `{ "256": "...", "100": "..." }`
  - `preset=galgame_banner` → `{ "mini": "..." }`
  - `preset=topic` → `{}`（空对象）
- `deduplicated` — 是否命中已存在图（调用方可用于统计；本次 preset 所需的变体如缺失，仍会补生成）

#### Go 类型定义（权威）

`apps/api/pkg/imageclient/client.go` 中的 `UploadResult` 是字段名权威来源，调用方直接 import 用：

```go
type UploadResult struct {
    Hash         string            `json:"hash"`
    URL          string            `json:"url"`
    VariantURLs  map[string]string `json:"variant_urls"`
    Width        int               `json:"width"`
    Height       int               `json:"height"`
    SizeBytes    int64             `json:"size_bytes"`
    Deduplicated bool              `json:"deduplicated"`
}

type ReferencePingResult struct {
    Updated  int64    `json:"updated"`
    NotFound []string `json:"not_found"`
}

// Sentinel errors
var (
    ErrQuotaExceeded      = errors.New("imageclient: quota exceeded")
    ErrModerationRejected = errors.New("imageclient: rejected by moderation")
    ErrUnauthorized       = errors.New("imageclient: unauthorized")
)
```

注意 `URL` / `VariantURLs` / `SizeBytes` 是 Go 命名（首字母大写驼峰）；JSON 字段是 snake_case（见 tag）。

**去重 + 补变体场景**：

假设同一个 hash 先被 A 站以 `preset=topic` 上传（只有主图），之后 B 站以 `preset=avatar` 再次上传：

- `images` 表行已存在
- 但 `variants` 列 `{}` 不含 `100`
- 服务从对象存储下载主图 → 生成 `100×100` 变体 → PUT → `UPDATE images SET variants = ARRAY_APPEND(variants, '100')`
- 返回：`deduplicated: true` + 完整的 `variant_urls`

**调用方接入例子（TypeScript 前端 via 调用方后端代传）**：

```ts
// 调用方后端
const token = await getOAuthToken()  // Client Credentials，缓存
const fd = new FormData()
fd.append('file', fileStream)
fd.append('preset', 'avatar')

const res = await fetch('https://image.api.example.com/image/upload', {
  method: 'POST',
  headers: { Authorization: `Bearer ${token}` },
  body: fd
})
const { hash, url, variant_urls } = await res.json()

await db.users.update({ where: { id: userId }, data: { avatar_image_hash: hash } })
return { avatar: url, avatar_thumb: variant_urls['100'] }
```

---

## 2. 查询元信息

### `GET /image/:hash`

查询一张图片的元信息。

**请求**：
```http
GET /image/abcd1234...ef HTTP/1.1
Authorization: Bearer <token>
```

**响应**：
```json
{
  "hash": "abcd...ef",
  "url": "https://cdn.example.com/img/ab/cd/abcd...ef.webp",
  "variant_urls": {
    "100": "https://cdn.example.com/img/ab/cd/abcd...ef_100.webp",
    "mini": "https://cdn.example.com/img/ab/cd/abcd...ef_mini.webp"
  },
  "width": 512,
  "height": 512,
  "size_bytes": 45678,
  "mime": "image/webp",
  "review_status": "approved",
  "created_at": "2026-04-23T10:20:30Z",
  "sites": ["kungal", "moyu"]
}
```

- `variant_urls` 包含**当前已生成的全部变体**，不再按 preset 过滤
- `sites` 来自 `image_site_usage`，显示哪些站上传过此 hash（仅 `image:read` scope 的 admin 可见；普通调用方只看到自己站）

**404** 如果 hash 不存在或已物理删除。

---

## 3. 批量续期引用

### `POST /image/reference-ping`

调用方周期性上报"我当前还在引用这些图"，图片服务更新其 `last_referenced_at`。

**请求**：
```json
{
  "hashes": [
    "abcd...ef",
    "1234...aa",
    "5678...bb"
  ]
}
```

最多 1000 个 hash / 请求。

**响应**：
```json
{
  "updated": 998,
  "not_found": ["5678...bb"]
}
```

- `updated` — 成功续期的条目数
- `not_found` — 图片服务没有这些 hash（可能已被清理；调用方可清理自己的外键）

**建议频率**：每天一次即可。新上传的图自动会刷 `last_referenced_at = NOW()`，无需单独 ping。

**调用方实现建议**：

```sql
-- kungal 每天凌晨
SELECT DISTINCT avatar_image_hash FROM users WHERE avatar_image_hash IS NOT NULL
UNION
SELECT DISTINCT banner_image_hash FROM galgame WHERE banner_image_hash IS NOT NULL
-- ... 其他业务实体的 hash 外键
```

按 1000 / 批发到 `/image/reference-ping`。

---

## 4. SDK（V2 提供）

派生变体 URL 不走后端 API，由调用方自行拼。图片服务提供 SDK 工具：

### Go SDK

```go
import "api/pkg/imageclient"

main := imageclient.MainURL(hash)
// → https://cdn.example.com/img/ab/cd/abcd...ef.webp

thumb := imageclient.VariantURL(hash, "100")
// → https://cdn.example.com/img/ab/cd/abcd...ef_100.webp
```

### TypeScript SDK

```ts
import { imageMainUrl, imageVariantUrl } from '@kun/image-client'

const main = imageMainUrl(hash)
const thumb = imageVariantUrl(hash, '100')
```

SDK 内部只做字符串拼接（因为 URL 是 content-addressed 且无签名），极简。

> **没有 HMAC 签名**。V1 所有图片 URL 都是公开可预测的。如需私有化由 CDN 层做访问控制。

---

## 5. 管理端点（V3）

以下接口 **V3 才实现**，V1 / V2 不上，需要 `image:admin` scope。

### `GET /admin/image/list`

**查询参数**：
- `site` — 过滤站点（过滤 `image_site_usage`）
- `review_status` — `pending` / `rejected` / `manual_review`
- `from` / `to` — 时间范围
- `page` / `limit`

### `PATCH /admin/image/:hash/review`

手动调整审核状态（V3）。

```json
{
  "status": "approved" | "rejected" | "manual_review",
  "reason": "误杀，人工放行"
}
```

### `GET /admin/stats`

**响应**：
```json
{
  "upload_count": 12345,
  "unique_images": 10234,
  "deduplicated_count": 2111,
  "total_bytes": 123456789012,
  "by_site": {
    "kungal": { "count": 8000, "unique": 6500 },
    "moyu":   { "count": 3000, "unique": 2500 },
    "galgame_wiki": { "count": 1345, "unique": 1234 }
  },
  "by_preset": {
    "avatar": 4000,
    "galgame_banner": 345,
    "topic": 8000
  }
}
```

> V2 可以先上 `GET /stats`（无 admin scope 需求）的简化版给运维看。

---

## 6. 健康检查

### `GET /healthz`

```json
{
  "status": "ok",
  "postgres": "ok",
  "storage": "ok",
  "redis": "ok"
}
```

任意依赖不健康则 HTTP 503。

### `GET /metrics`

Prometheus 指标端点（标准 Go runtime + 自定义业务指标）。

> **仅内网 / VPC 暴露**。不对公网放行（会泄露站点上传量、去重率等敏感业务数据）。部署时在反代/防火墙层面拒绝外部访问。

自定义指标：
- `image_upload_total{site,preset,result}`
- `image_upload_duration_seconds{site,preset}`
- `image_processing_duration_seconds{op}` — `decode` / `resize` / `encode` / `variant_gen` / `store`
- `image_dedup_hits_total{site}`
- `image_storage_bytes_total`（周期采样）
- `image_quota_remaining{site,type}` — `count` / `bytes`

---

## 端点汇总

| Method | Path | Scope | 里程碑 |
|--------|------|-------|--------|
| POST | `/image/upload` | `image:upload` | V1 |
| GET | `/image/:hash` | `image:read` | V1 |
| POST | `/image/reference-ping` | `image:upload` | V1 |
| GET | `/stats` | `image:read` | V2 |
| GET | `/admin/image/list` | `image:admin` | V3 |
| PATCH | `/admin/image/:hash/review` | `image:admin` | V3 |
| GET | `/admin/stats` | `image:admin` | V3 |
| GET | `/healthz` | — | V1 |
| GET | `/metrics` | — | V1 |

## 没有的端点

显式说明 **V1 故意不提供**：

- ❌ `DELETE /image/:hash` — 调用方不主动删图（决策 0）。仅合规场景走 V3 的 admin-only `DELETE /admin/image/:hash?force=true`
- ❌ `POST /image/upload-ticket` — V1 无前端直传，不需要签发临时凭证
- ❌ `GET /image/:hash/variants/*` — 变体是固定静态 URL，不需要查询端点
- ❌ Raw body 上传（`POST` + `Content-Type: image/*` + header `X-Preset`）——见上文 §1 body 格式说明

## CORS

V1 **不对外开放 CORS**（只接受后端调用）。V2 开放前端直传时再配白名单，从 `oauth_client.redirect_uris` 派生。

下一篇：[04 — 迁移计划](./04-migration-plan.md)
