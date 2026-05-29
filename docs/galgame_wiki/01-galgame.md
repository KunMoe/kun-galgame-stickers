> [📖 文档索引](./README.md) · 下一节：[02 — 版本历史 + PR](./02-revisions-and-prs.md)

## Galgame 核心 CRUD

### GET /galgame

列表（分页 + 搜索 + 排序）。

**查询参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 24 | 每页数量 (1-50) |
| sort_field | string | 否 | created | 排序字段: `created`, `updated`, `view`, `resource_update_time`, `release_date` |
| sort_order | string | 否 | desc | 排序方向: `asc`, `desc` |
| search | string | 否 | | 搜索关键词（匹配四语言名称） |
| content_limit | string | 否 | **sfw** | NSFW 过滤。`sfw` / `nsfw` / `all`。**省略 = sfw**（safe-by-default）。详见 [00-handbook §NSFW](./00-handbook-for-downstream.md#nsfw-content_limit-协议) |
| released_from | string | 否 | | 发售日期下限（含）。支持 `YYYY`（按年）或 `YYYY-MM`（按月）。详见 [00-handbook §17 日期筛选](./00-handbook-for-downstream.md#17-发售日期-release_date-筛选协议) |
| released_to | string | 否 | | 发售日期上限（含）。同上格式。 |
| released_months | int (csv) | 否 | | 不连续月份集合（1–12），叠加在年份区间上的 AND 过滤。如 `3,7,12`。详见 [00-handbook §17.10](./00-handbook-for-downstream.md#1710-不连续月份筛选released_months2026-05-28-新增) |

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "items": [
      {
        "id": 1,
        "vndb_id": "v12345",
        "name_en_us": "Title",
        "name_ja_jp": "タイトル",
        "name_zh_cn": "标题",
        "name_zh_tw": "標題",
        "banner": "https://...",
        "effective_banner_hash": "abcd1234...ef",
        "release_date": "2019-08-16",
        "release_date_tba": false,
        "content_limit": "sfw",
        "view": 100,
        "created": "2026-01-01T00:00:00Z",
        "tag": [...],
        "official": [...],
        "cover": [...],
        "screenshot": [...]
      }
    ],
    "total": 42
  }
}
```

---

### GET /galgame/batch

批量获取 galgame 轻量信息（跨服务展示用，不加载关联数据）。

> ⚠️ **batch 是轻量 DTO，字段是白名单的**。返回**只有**下方响应示例里那些字段（id / vndb_id / name_* / banner / effective_banner_hash / content_limit / status / user_id / resource_update_time / original_language / age_limit）。
>
> **不包含**：`release_date` / `release_date_tba` / `intro_*` / `tag` / `official` / `engine` / `series` / `cover` / `screenshot` / `alias` 等。要这些字段请用 **`GET /galgame/:gid`**（单条详情）、**`GET /galgame`**（列表）或 **`GET /galgame/search`**（搜索）—— 这三个端点都带 `release_date`。
>
> 典型踩坑：想按发售日期做本地镜像/筛选的下游，**不能**从 batch 拿 `release_date`（它压根不在 batch 里），必须走 `/galgame/:gid` 或列表/搜索端点。

**查询参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| ids | int[] | 是 | — | galgame ID 数组，最多 100 个 |
| content_limit | string | 否 | **（不过滤）** | NSFW 过滤。`sfw` / `nsfw` / `all`。**与 list/search 不同：batch 默认不过滤** — 调用方已经明确知道想要哪些 ID（如 patch.galgame_id、收藏列表），过滤会"静默丢失"调用方显式列出的条目。需要过滤时显式传 `content_limit=sfw`。详见 [00-handbook §NSFW](./00-handbook-for-downstream.md#nsfw-content_limit-协议) |

**请求示例**：`GET /galgame/batch?ids=1,2,3`

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": 1,
      "vndb_id": "v12345",
      "name_en_us": "Title",
      "name_ja_jp": "タイトル",
      "name_zh_cn": "标题",
      "name_zh_tw": "標題",
      "banner": "https://image.kungal.com/...",
      "effective_banner_hash": "abcd...ef",
      "content_limit": "sfw",
      "status": 0,
      "user_id": 1,
      "resource_update_time": "2026-01-01T00:00:00Z",
      "original_language": "ja-jp",
      "age_limit": "all"
    }
  ]
}
```

| 字段 | 备注 |
|---|---|
| `status` | `0` 已发布；`3` / `4` 仅当请求带 Bearer 且调用者是该条 galgame 的 submitter 时才会返回（pending / declined 草稿）。匿名 / 非 owner 调用看不到非 0 条目 |
| `effective_banner_hash` | image_service 哈希（= `covers[sort_order=0].image_hash`，PR5 退役 banner_image_hash 后唯一的 image_service banner 引用）；前端 `resolveBannerUrl` 优先用此字段拼 CDN URL，缺失时 fallback 到 `banner` 老 URL |

不存在或对调用者不可见的 ID 会被过滤，不会报错。返回数组长度可能小于请求的 ID 数量。

**带 Bearer 的语义**：与 OAuth 终端用户 access_token 一起调用，wiki 解 JWT 得 `id`，返回结果中额外包含调用者的 status=3/4 条目（参见 [07-submission.md GET /galgame/batch 增量行为](./07-submission.md#get-galgamebatch-增量行为)）。无 Bearer 时只返 status=0。

---

### GET /galgame/user/:id/stats

获取用户的 Galgame 统计数据。

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 用户 ID |

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "galgame_created": 10,
    "galgame_created_today": 1,
    "galgame_contributed": 15,
    "revision_count": 42,
    "pr_submitted": 5,
    "pr_merged": 3,
    "pr_declined": 1,
    "pr_pending": 1
  }
}
```

| 字段 | 说明 |
|------|------|
| galgame_created | 用户创建的 galgame 总数（不含被封禁的） |
| galgame_created_today | 用户今日创建的 galgame 数量 |
| galgame_contributed | 用户参与贡献的 galgame 数量（含创建和编辑） |
| revision_count | 用户产生的版本记录总数 |
| pr_submitted | 用户提交的 PR 总数 |
| pr_merged | 已合并的 PR 数量 |
| pr_declined | 已拒绝的 PR 数量 |
| pr_pending | 待处理的 PR 数量 |

用户不存在时返回全零数据，不报错。

---

### GET /galgame/check

检查 VNDB ID 是否已存在 + 返回对应整数 `galgame_id`。

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| vndb_id | string | 是 | VNDB ID (如 `v12345`) |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "exists": true,
    "galgame_id": 1
  }
}
```

#### Recipe — 按 VNDB ID 取完整 galgame 信息

本服务**不提供**直接 `GET /galgame/by-vndb/:vndb_id` 端点（设计上 galgame 主键是整数 `id`）。要按 VNDB ID 取完整信息，标准做法是两步：

```bash
# Step 1: 拿到整数 galgame_id
GET /galgame/check?vndb_id=v17
# → { "exists": true, "galgame_id": 8329 }

# Step 2: 取完整信息（含 alias / tag / official / series 等 relations）
GET /galgame/8329
# → { "galgame": { ... 全字段 + relations ... }, "users": { ... } }
```

伪代码：

```go
// Go
exists, gid := api.CheckVNDB(ctx, "v17")
if !exists { return nil }
galgame := api.GetGalgame(ctx, gid)
```

```ts
// TypeScript
const r1 = await api.get<{exists: boolean; galgame_id: number}>('/galgame/check?vndb_id=v17')
if (!r1.data.exists) return null
const r2 = await api.get<{galgame: Galgame}>(`/galgame/${r1.data.galgame_id}`)
return r2.data.galgame
```

> 单次请求的"按 vndb_id 搜索"也可以用 `GET /galgame/search?q=v17&limit=1`（vndb_id 在 Meilisearch searchableAttributes 里且禁用 typo），但**这是 search 语义**——依赖 Meilisearch 在线 + 索引同步状态，不如 `/check` 端点强一致。常规跨服务集成走上面的 2 步 recipe。

---

### GET /galgame/:gid

获取详情（含全部关联数据 + 用户信息）。

**查询参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| content_limit | string | 否 | **（不过滤）** | NSFW 过滤。`sfw` / `nsfw` / `all`。**与 list/search 不同：详情默认不过滤** — 直接 URL 访问（书签、深链）是有意为之；如果想让详情访问也按 list 同样的过滤口径，前端调用时显式带上该参数。不匹配的条目返回 404（与 status 过滤同形）。详见 [00-handbook §NSFW](./00-handbook-for-downstream.md#nsfw-content_limit-协议) |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "galgame": {
      "id": 1,
      "vndb_id": "v12345",
      "name_en_us": "...",
      "name_ja_jp": "...",
      "name_zh_cn": "...",
      "name_zh_tw": "...",
      "banner": "...",
      "intro_en_us": "...",
      "intro_ja_jp": "...",
      "intro_zh_cn": "...",
      "intro_zh_tw": "...",
      "content_limit": "sfw",
      "original_language": "ja-jp",
      "age_limit": "r18",
      "view": 100,
      "user_id": 1,
      "series_id": null,
      "alias": [{"id": 1, "name": "别名"}],
      "tag": [
        {
          "galgame_id": 1,
          "tag_id": 2,
          "spoiler_level": 0,
          "tag": {
            "id": 2,
            "name": "RPG",
            "category": "content",
            "alias": [{"name": "角色扮演"}],
            "galgame_count": 1234
          }
        }
      ],
      "official": [
        {
          "galgame_id": 1,
          "official_id": 3,
          "official": {
            "id": 3,
            "name": "开发商",
            "link": "https://...",
            "category": "company",
            "lang": "ja-jp",
            "alias": [{"name": "别名"}],
            "galgame_count": 12
          }
        }
      ],
      "engine": [
        {
          "galgame_id": 1,
          "engine_id": 5,
          "engine": {
            "id": 5,
            "name": "KiriKiri",
            "alias": ["KiriKiri2", "KAG"],
            "galgame_count": 87
          }
        }
      ],
      "link": [{"id": 1, "name": "VNDB", "link": "https://vndb.org/v12345"}],
      "contributor": [{"id": 1, "user_id": 1}],
      "created": "...",
      "updated": "..."
    },
    "users": {
      "1": {"id": 1, "name": "KUN", "avatar": "https://..."}
    }
  }
}
```

`users` 是一个 user_id → 用户信息的 map，包含 galgame 创建者和所有贡献者的信息。

> 🆕 **2026-05-22 (K-PR)**：每个 `tag.tag`、`official.official` 和 `engine.engine` 对象**新增 `galgame_count` 字段**，表示该 tag / 开发商 / 引擎下已发布（`status = 0`）的 galgame 数。这与独立的 `GET /tag`、`GET /official` 和 `GET /engine` 列表端点使用的计数口径一致（同款 `LEFT JOIN ... COUNT(*)` 子查询，drafts 不计入）。下游可直接在 galgame 详情页面渲染"标签名 +N / 会社名 +N / 引擎名 +N" badge，**不再需要为每个嵌入实体各发一次单独的 `GET /tag/:id`、`GET /official/:id`、`GET /engine/:id` 请求**。
>
> 字段为 `int`，**没有 `omitempty`**：值为 0 是"该实体当前 0 部已发布作品"的有效语义，不应被序列化省略。

---

### POST /galgame

创建 Galgame。**需要认证 + admin/moderator 角色**。

> ⚠️ **此端点已锁到 admin/moderator**。它是 admin 直接发布的旁路：创建 `status=0` 条目，跳过审核队列。
>
> **普通用户必须走 [POST /galgame/submit](./07-submission.md#post-galgamesubmit)**——创建 `status=3` 待审稿，无需 VNDB ID，进入审核队列。
>
> 非 admin/moderator 调此端点返回 403。kungal/moyu 的发布 UI 应仅暴露 `/submit`。

创建时自动：创建 revision 1、添加创建者为贡献者、添加 VNDB 链接。

**支持两种 Content-Type**：
- `application/json` — 不上传 banner 文件时使用（请求体见下）
- `multipart/form-data` — 创建时直接带 banner 文件，详见
  [Banner 上传](#banner-上传通过-create--update--pr-端点的-multipart-模式)

**请求体**（JSON 模式）：

```json
{
  "vndb_id": "v12345",
  "name_en_us": "Title",
  "name_ja_jp": "タイトル",
  "name_zh_cn": "标题",
  "name_zh_tw": "標題",
  "banner": "https://...",
  "release_date": "2019-08-16",
  "release_date_tba": false,
  "intro_en_us": "...",
  "intro_ja_jp": "...",
  "intro_zh_cn": "...",
  "intro_zh_tw": "...",
  "content_limit": "sfw",
  "original_language": "ja-jp",
  "age_limit": "r18",
  "series_id": null,
  "aliases": "别名1,别名2",
  "tag_ids": [1, 2, 3],
  "official_ids": [1],
  "engine_ids": [1],
  "covers": [
    {"image_hash": "abcd1234...ef", "sort_order": 0, "sexual": 0, "violence": 0, "source": "user", "source_key": ""}
  ],
  "screenshots": []
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| vndb_id | 是 | 格式 `v\d+`，必须唯一 |
| banner | 否 | 老的 URL 字符串字段；image_service 接入前的旧路径，迁移期保留作 fallback |
| covers | 否 | image_service 哈希数组，PR2 起的新字段；`sort_order=0` 的那张是钉住封面（DB 强制每作品至多一张）。详见下面 PUT 端点说明 |
| screenshots | 否 | image_service 哈希数组，与 covers 同 shape，无"钉住"约束 |
| aliases | 否 | 逗号分隔的别名字符串 |
| tag_ids | 否 | 标签 ID 数组 |
| official_ids | 否 | 开发商 ID 数组 |
| engine_ids | 否 | 引擎 ID 数组 |
| content_limit | 否 | `sfw` (默认) 或 `nsfw` |
| age_limit | 否 | `r18` (默认) 或 `all` |
| release_date | 否 | `YYYY-MM-DD` 字符串；`""` 表示未知（PR1 取代旧 `released` 字符串） |
| release_date_tba | 否 | bool；`true` 表示官方已宣布但日期未定（与 `release_date` 独立） |

> **banner 优先级**：前端读取时优先派生只读字段 `effective_banner_hash`（= `covers[sort_order=0].image_hash`，拼 image_service URL），缺失时回退 `banner` 老 URL。PR5 起不再有 `banner_image_hash` 字段，banner 由 `covers` 表达。multipart 上传见下方 Banner 上传段（hash 经 `PromoteCoverHash` 由服务合并进 covers）。

---

### PUT /galgame/:gid

更新 Galgame。**需要认证**。仅创建者或 admin 可操作。

每次更新自动创建新 revision。**所有字段（含 `covers` / `screenshots`）的变化都会进入 revision 快照与 PR diff**（covers/screenshots 的 image_hash 集合按集合语义 diff，参见 `apps/api/internal/platform/galgame/model/galgame_revision.go` 中的 ChangedKeys）。

**支持两种 Content-Type**：
- `application/json` — 不修改 banner 或修改时只改 hash 字段
- `multipart/form-data` — 同时上传新 banner 文件，详见
  [Banner 上传](#banner-上传通过-create--update--pr-端点的-multipart-模式)

**请求体**（JSON 模式，所有字段可选）：

```json
{
  "name_zh_cn": "新标题",
  "intro_zh_cn": "新简介",
  "release_date": "2019-08-16",
  "release_date_tba": false,
  "aliases": ["别名A", "别名B"],
  "links": [{"name": "官网", "link": "https://example.com"}],
  "tag_ids": [1, 2, 3],
  "official_ids": [1],
  "engine_ids": [],
  "covers": [
    {"image_hash": "abcd1234...ef", "sort_order": 0, "sexual": 0, "violence": 0, "source": "user", "source_key": ""}
  ],
  "screenshots": [
    {"image_hash": "fedcba98...12", "sort_order": 0, "caption": "CG 01", "sexual": 0, "violence": 0, "source": "", "source_key": ""}
  ],
  "is_minor": false
}
```

> **PR5 BREAKING — `banner_image_hash` 字段已彻底移除。** Banner 现在唯一通过 `covers[sort_order=0]` 表达；详见
> [00-handbook §15 PR5 BREAKING 段](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。

`covers` / `screenshots` 字段说明：
- 都按 `image_hash` 引用 image_service（先在 image_service 上传得 hash，再在本端点提交）。
- `covers` 中 `sort_order=0` 的那张是**钉住的封面**（DB 强制每作品至多一张），管理员"换封面" = 同一请求里把旧的 `sort_order` 改成非 0、新的设为 0。
- `screenshots` 没有"钉住"约束，`sort_order` 只是画廊展示顺序。
- presence 语义同 `tag_ids`：不传 = 保持原集合不变；传 `[]` = 清空全部；传非空数组 = 权威全量替换（**必须回传该作当前全量**，不要只回传新增/删除的那几条）。
- 响应里有派生字段 `effective_banner_hash`（= covers 里 `sort_order=0` 的那张的 `image_hash`；无则 null）。**前端封面展示通过 `resolveBannerUrl` helper，只看 `effective_banner_hash → banner` 两级。**

> ⚠️ **多值字段（`tag_ids` / `official_ids` / `engine_ids` / `aliases` / `links`）= presence 语义全量替换，必须看懂**：
> - **不传该字段** → 该 galgame 的对应集合**保持不变**（只改名字时绝不会清空 tag/别名）。
> - **传数组（含空 `[]`）** → 该字段是**权威全量集合**：服务端"清空旧的 → 按此重建"。`[]` = 显式清空全部。
> - 因此下游（kungal/moyu）编辑表单**必须回传该 galgame 当前的全量集合**（在原集合上增/删后整体回传），**不要只回传"新增的"那几个**——会被当成"替换成只剩这几个"。
> - 与标量字段一致：传了就改、不传就不动。整个编辑是**一次事务、一条 revision**（原子；集合语义、顺序无关，进 revision 快照与 PR diff）。
> - **`release_date` / `release_date_tba`**（取代旧 `released` 字符串）现可经此端点编辑，各自走 presence 语义：`release_date` 用 `*string`（`null`/省略 = 保持，`""` = 清空为未知，`"YYYY-MM-DD"` = 设置）；`release_date_tba` 用 `*bool`。两者独立——可同时给值表达"预计 X 年某月 + TBA"。详见 §00-handbook BREAKING 段。
>
> `aliases` / `links` 现已是本端点的一等字段（推荐整表单一次性提交）。`/galgame/:gid/aliases|links` 的增删端点保留为便捷糖（同样每次产生 revision），但一次性表单保存请走本端点以获得原子单条 revision。`bid`/Bangumi ID 为保留字段，暂不可编辑（sync 托管）。

`is_minor` 为 `true` 时标记为小修改，在版本历史中可被过滤。

---

### Banner 上传：通过 Create / Update / PR 端点的 multipart 模式

**没有独立的"上传 banner"端点**。banner 文件作为可选 `file` 表单字段一并随
`POST /galgame`、`PUT /galgame/:gid`、`POST /galgame/:gid/prs` 的 multipart 请求提交，
后端会先把文件转给 image_service 拿到 hash，再通过服务端瞬态字段 `PromoteCoverHash`
把它合并进 `covers`（若同 hash 已存在则提升为 `sort_order=0`，否则插入为新的 pinned cover；
原有的 `sort_order=0` 自动让位），跟其他字段一起进入同一次 revision / PR diff。

> 设计动机：图片上传与 article 编辑在业务上是同一次动作，应当原子。
> 不再有"上传成功但忘了点保存留下 orphan 文件"的情况——文件在浏览器内存里
> 暂存，没点保存就丢弃，从源头避免 orphan。

**两种 Content-Type 等价**，前端按需选用：

#### A. application/json — 不上传文件时使用（与以前完全相同）

```http
PUT /api/v1/galgame/:gid
Content-Type: application/json

{ ... fields including optional covers/screenshots arrays ... }
```

#### B. multipart/form-data — 需要上传 banner 文件时使用

```http
PUT /api/v1/galgame/:gid
Content-Type: multipart/form-data; boundary=...

--boundary
Content-Disposition: form-data; name="data"

{"name_zh_cn": "新标题", ...other fields}
--boundary
Content-Disposition: form-data; name="file"; filename="banner.png"
Content-Type: image/png

<binary>
--boundary--
```

| 字段 | 必填 | 说明 |
|------|------|------|
| data | 是 | JSON 字符串，等同于 JSON 模式下的 body |
| file | 否 | 图片文件（image/jpeg / png / webp）；上传后后端把 hash 作为 `PromoteCoverHash` 合并进 `covers`，提升为 `sort_order=0` 钉住封面（详见上方 covers 字段语义） |

**错误码**（multipart 模式下额外可能出现的）：透传 image_service 的状态码与
错误码（如 `80008` 配额超限、`80015` 上传暂未开放、`60002` 审核拒绝），调用方
按需展示给用户。

**该 multipart 模式同样适用于：**
- `POST /galgame`（创建时直接带 banner 文件，避免"先创建再编辑改 banner"两步）
- `POST /galgame/:gid/prs`（PR 提案里直接附 banner 文件，reviewer 看 diff 时能看到新图缩略图）

---


---

下一节：[02 — 版本历史 + PR](./02-revisions-and-prs.md)
