> [📖 文档索引](./README.md) · 上一节：[03 — 链接 / 别名 / 贡献者](./03-relations.md) · 下一节：[05 — 搜索](./05-search.md)

> 🔴 **下游强制范围**：本节 tag / official / engine / series 的增删改查（含本次新增的 `POST /tag` `POST /official` `POST /engine` 与各 `DELETE`），**kungal 与 moyu 各自必须完整实现一份**（后端透传 Bearer 代理 + 前端「选已有/没有就新建」UI，功能与 wiki 对齐，不得只做子集）。详见 [00-handbook §15](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。

## 标签 (Tag)

### GET /tag

标签列表（分页，按关联 galgame 数量排序）。

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| page | int | 1 | 页码 |
| limit | int | 50 | 每页数量（max 100） |

> 🆕 **2026-05-22 (K-PR)**：`galgame_count` 字段（已发布作品数）现在**也会同步出现在 [`GET /galgame/:gid`](./01-galgame.md#get-galgamegid) 详情响应嵌入的 `tag.tag` 对象上**。两处计数口径完全一致（同款 `LEFT JOIN ... COUNT(*) WHERE status = 0` 子查询）。下游可在 galgame 详情页直接渲染"标签 +N" badge，**不再需要为每个嵌入 tag 单独发 `GET /tag/:name` 请求**。同款扩展也加到了 `official` 和 `engine`，见下文。

### GET /tag/search

搜索标签。**由 Meilisearch 驱动**，替代原 DB LIKE 实现。详见 [搜索 (Search)](#搜索-search) 章节。

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| q | string | `""` | 搜索词；空时按 `galgame_count` 倒序返回热门 tag |
| category | string | — | `content` / `sexual` / `technical` |
| limit | int | 50 | max 100 |

**响应**：
```json
{
  "items": [
    { "id": 45, "name": "校园", "aliases": ["学园"], "category": "content", "galgame_count": 850 }
  ],
  "total": 1,
  "processing_time_ms": 4
}
```

### GET /tag/multi?tag_ids=1,2,3

多标签筛选，返回同时拥有所有指定标签的 galgame。

**查询参数**：`page`, `limit`, `tag_ids`（数组）

### GET /tag/:name

标签详情 + 关联的 galgame 列表。

> ⚠️ **`:name` 路径段仅用于 URL 美观 / 分享**（如 `/tag/校园?tag_id=42`），实际的查询条件是 `tag_id` query 参数。后端不读 `:name`，传任意字符串都会按 `tag_id` 查找。这与 Wikipedia 的 `/wiki/Article_Name?oldid=N` 设计一致。

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tag_id | int | 是 | tag 主键，**实际查询字段** |
| page | int | 否 | 页码 |
| limit | int | 否 | 每页数量 |
| sort_field | string | 否 | `created` / `resource_update_time` / `view` |
| sort_order | string | 否 | `asc` / `desc` |
| content_limit | string | 否 | `sfw` / `nsfw` —— 仅返回对应分级 galgame，`total` 同步反映过滤后数量 |

### PUT /tag

更新标签。**需要认证（admin/moderator）**。

```json
{
  "tag_id": 1,
  "name": "新名称",
  "category": "content",
  "description": "描述",
  "alias": ["别名1", "别名2"]
}
```

事务内替换全部别名。

### POST /tag

新建标签。**需要认证（任意登录用户）** —— 这是为了让 kungal/moyu 用户给「VNDB 没有的原创 / 同人作品」补一个 wiki 里尚不存在的 tag（与 `POST /series` 同权限模型）。

```json
{
  "name": "标签名",
  "category": "content",
  "description": "可选描述",
  "alias": ["别名1", "别名2"]
}
```

- `category` 必填，取值 `content` / `sexual` / `technical`。
- 同名已存在 → `400`，`message` 提示「已存在同名 tag」（name 全局唯一）。前端遇此应改用 `GET /tag/search` 选用既有 tag。
- 成功返回新建的 tag 实体（含 `id`），随后写入 Meilisearch。

### DELETE /tag/:id

删除标签。**需要 role > 1（admin/moderator；普通用户 role=1 一律 403）**。

安全默认 + 强制清除两段式（同 `DELETE /admin/image/:hash?force=true` 约定）：

- **默认（不带 `?force=true`）**：若该 tag 仍被任意 galgame 引用 → **拒绝**，返回
  `code:7`，message 含被引用的 galgame 数。避免一删就静默把它从 N 个作品上摘掉。
- **`?force=true`**：一键「清除全部引用 → 硬删」。事务内级联删除该 tag 的别名
  （`galgame_tag_alias`）与全部 `galgame_tag_relation` 关联，再硬删 tag 行，并从
  Meilisearch 移除，无悬挂引用。
- 引用数为 0 时，带不带 `force` 都直接删（无引用可清）。

成功响应：

```json
{ "deleted": true, "forced": true, "purged_relations": 2, "purged_aliases": 0 }
```

`forced` 表示本次是「带 force 且确实清理了引用」。`purged_relations` / `purged_aliases`
为本次清掉的行数（审计用）。

---

## 开发商 (Official)

### GET /official

开发商列表。**查询参数**：`page`, `limit`

> 🆕 **2026-05-22 (K-PR)**：`galgame_count` 字段（已发布作品数）现在**也会同步出现在 [`GET /galgame/:gid`](./01-galgame.md#get-galgamegid) 详情响应嵌入的 `official` / `engine` 对象上**。两处计数口径完全一致（同款 `LEFT JOIN ... COUNT(*) WHERE status = 0` 子查询）。下游不再需要为渲染"会社名 +N 部作品" badge 而对每个嵌入的 official 单独发一次 `GET /official/:name` 请求。

### GET /official/search

搜索会社。**由 Meilisearch 驱动**，详见 [搜索 (Search)](#搜索-search)。

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| q | string | `""` | 搜索词；空时按 `galgame_count` 倒序 |
| category | string | — | `company` / `individual` / `amateur` |
| lang | string | — | 按主语言过滤（`ja`, `en`, `zh-Hans` 等） |
| limit | int | 50 | max 100 |

### GET /official/:name

详情 + 关联 galgame。

> ⚠️ **`:name` 路径段仅用于 URL 美观**，实际查询字段是 `official_id` query 参数（同 [GET /tag/:name](#get-tagname) 的设计）。

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| official_id | int | 是 | official 主键，**实际查询字段** |
| page | int | 否 | 页码 |
| limit | int | 否 | 每页数量 |
| sort_field | string | 否 | `created` / `resource_update_time` / `view` |
| sort_order | string | 否 | `asc` / `desc` |
| content_limit | string | 否 | `sfw` / `nsfw`，只返回对应分级 galgame，`total` 同步反映过滤后数量 |

### PUT /official

更新。**需要认证（admin/moderator）**。

```json
{
  "official_id": 1,
  "name": "新名称",
  "link": "https://...",
  "category": "company",
  "lang": "ja",
  "description": "描述",
  "alias": ["别名1"]
}
```

### POST /official

新建开发商 / 会社。**需要认证（任意登录用户）**，同 `POST /tag` 的用途与权限模型。

```json
{
  "name": "会社名",
  "category": "company",
  "original": "原文名（可选，日文等）",
  "link": "https://...（可选）",
  "lang": "ja（可选）",
  "description": "可选描述",
  "alias": ["别名1"]
}
```

- `category` 必填，取值 `company` / `individual` / `amateur`。
- 同名已存在 → `400`「已存在同名 official」（name 全局唯一）。
- 成功返回新建实体，写入 Meilisearch。

### DELETE /official/:id

删除开发商。**需要 role > 1（admin/moderator）**。与 [`DELETE /tag/:id`](#delete-tagid) 完全同款两段式：默认被引用则**拒绝**（返回引用数），`?force=true` 才一键清除别名（`galgame_official_alias`）+ `galgame_official_relation` 后硬删并移出 Meilisearch；成功返回 `{deleted,forced,purged_relations,purged_aliases}`。

---

## 引擎 (Engine)

### GET /engine

全量列表（数据量小，不分页）。

> 🆕 **2026-05-22 (K-PR)**：`galgame_count` 同时也会出现在 [`GET /galgame/:gid`](./01-galgame.md#get-galgamegid) 详情响应里每个 `engine.engine` 对象上，见上面 "GET /official" 同款说明。

### GET /engine/:name

详情 + 关联 galgame。

> ⚠️ **`:name` 路径段仅用于 URL 美观**，实际查询字段是 `engine_id` query 参数（同 [GET /tag/:name](#get-tagname) 的设计）。

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| engine_id | int | 是 | engine 主键，**实际查询字段** |
| page | int | 否 | 页码 |
| limit | int | 否 | 每页数量 |
| content_limit | string | 否 | `sfw` / `nsfw`，只返回对应分级 galgame，`total` 同步反映过滤后数量 |

### PUT /engine

更新。**需要认证（admin/moderator）**。

```json
{
  "engine_id": 1,
  "name": "新名称",
  "description": "描述",
  "alias": ["别名1"]
}
```

引擎的 `alias` 以 JSONB 数组存储（与 tag/official 的关联表不同）。

### POST /engine

新建引擎。**需要认证（任意登录用户）**，同 `POST /tag` 的用途与权限模型。

```json
{
  "name": "引擎名",
  "description": "可选描述",
  "alias": ["别名1"]
}
```

- 同名已存在 → `400`「已存在同名 engine」（name 全局唯一）。
- 引擎**不进 Meilisearch**（无 `/engine/search`），故无搜索写回。
- 成功返回新建实体。

### DELETE /engine/:id

删除引擎。**需要 role > 1（admin/moderator）**。同款两段式：默认被引用则**拒绝**（返回引用数），`?force=true` 才一键清除 `galgame_engine_relation` 后硬删（引擎别名为行内 JSONB，随行删除；引擎不进 Meili，无索引移除）；成功返回 `{deleted,forced,purged_relations}`（无 `purged_aliases`，引擎无别名表）。

---

## 系列 (Series)

### GET /series

系列列表（含前 5 个 galgame 预览）。**查询参数**：`page`, `limit`

### GET /series/search?keywords=xxx

搜索 galgame（按名称、VNDB ID、标签、别名），用于系列分配。

返回最多 20 条。

### GET /series/:id

系列详情 + 全部 galgame。

### POST /series

创建系列。**需要认证**。

```json
{
  "name": "系列名",
  "description": "描述",
  "galgame_ids": [1, 2, 3]
}
```

### POST /series/modal

按 ID 批量获取 galgame（模态框用）。**需要认证**。

```json
{
  "ids": [1, 2, 3]
}
```

返回结果按输入 ID 顺序排列。

### PUT /series/:id

更新系列。**需要认证**。

```json
{
  "name": "新名称",
  "galgame_ids": [1, 2, 4]
}
```

`galgame_ids` 会**替换**系列中的所有 galgame。

### DELETE /series/:id

删除系列。**需要认证（admin/moderator）**。关联的 galgame 的 `series_id` 会被置为 `null`。

---

## 修订与回滚（PR4 新增，4 实体同款）

> 🟢 **ADDITIVE — taxonomy 编辑现已全量审计 + 可 revert**：
> 每次 tag/official/engine/series 的 create / update / delete 都会写一条 `taxonomy_revision` 行（多态单表：`entity` 列区分）。下面 4 实体 × 3 端点 = 12 条新路由，shape **完全一致**。
> 详见 [00-handbook §15 ADDITIVE 段](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖) + [`docs/galgame_wiki/01-revision-system-design.md §14`](../../galgame_wiki/01-revision-system-design.md)。

### GET /{tag,official,engine,series}/:id/revisions

列出该实体的修订历史（newest first，分页）。**公开（与 galgame `/revisions` 一致）**。

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| page | int | 1 | 页码 |
| limit | int | 20 | 每页（max 50） |

**响应**：
```json
{
  "items": [
    {
      "id": 42,
      "entity": "tag",
      "target_id": 17,
      "revision": 3,
      "action": "updated",
      "user_id": 9,
      "user_role": 3,
      "snapshot": { "name": "校园", "category": "content", "description": "...", "aliases": ["学园"] },
      "changed_fields": ["description"],
      "ref_count": 0,
      "affected_galgame_ids": [],
      "note": "",
      "created": "2026-05-20T03:21:09Z"
    }
  ],
  "total": 5
}
```

字段说明：
- `entity`：`tag` / `official` / `engine` / `series` 之一；与路径前缀一致
- `revision`：per-(entity, target_id) 自增序号，从 1 开始
- `action`：`created` / `updated` / `deleted` / `reverted`
- `snapshot` 形态因 entity 而异（TagSnapshot / OfficialSnapshot / EngineSnapshot / SeriesSnapshot）
- `changed_fields`：每次编辑改动的字段名列表
  - `created` 列出 Snapshot 的全部字段名（如 tag 的 `["name","category","description","aliases"]`）
  - `updated` 只列出实际 diff 的字段
  - `deleted` 是空数组（整行消失，不是某些字段被改）
  - `reverted` 列出与"当前状态" diff 的字段
- `ref_count` + `affected_galgame_ids`：**仅 `deleted` 行有值**，被删除时该实体被多少 galgame 引用 + 受影响的 gid 列表
- `user_role`：编辑时该用户的角色快照（admin=3 / moderator=2 / user=1 / 0），为将来字段级 RBAC 留位

### GET /{tag,official,engine,series}/:id/revisions/:rev

加载某一条特定修订记录。**公开**。响应同上单条。

### POST /{tag,official,engine,series}/:id/revert

回滚到目标修订。**需要 admin/moderator**。

**请求体**（与 galgame revert 同 shape）：
```json
{ "revision": 3 }
```

**响应**：
```json
{ "reverted_to": 3 }
```

回滚行为：
- 若实体行还存在 → 直接 `Apply*Snapshot` 把目标 snapshot 重放到 DB
- 若实体行已被删除（最近 action='deleted') → INSERT 主行（用目标 snapshot 的 Name 等字段）+ 重建 aliases；产生一条 action='reverted' 的新 revision
- **不**自动恢复 galgame_*_relation 关联（避免悄悄改 galgame 状态）。前端 UI 可以读 deleted revision 的 `affected_galgame_ids` 给 admin 列一份"建议恢复"，admin 勾选哪些就到对应 galgame 上手动加回 tag_ids/official_ids/engine_ids/series_id（每个对应一条 galgame_revision）

### 跨实体编辑联动（重要）

`taxonomy delete force-purge` 不只产生一条 taxonomy_revision，还会**为每个 affected galgame 写一条 `galgame_revision`**（changed_fields 标记 `['tag_ids' | 'official_ids' | 'engine_ids' | 'series_id']`），保证 galgame 修订历史里这个 tag/official/engine 的"消失"也可查询。详见 [02-revisions-and-prs.md](./02-revisions-and-prs.md)。

### Series 的特殊性

`PUT /series/:id` 既改 series 标量字段（Name/Description）又改 galgame_id 成员（`galgame_ids` 数组）。**这两类编辑产生不同的 revision**：
- **改 Name/Description** → 一条 `taxonomy_revision (entity='series', action='updated')`
- **改 galgame_ids（成员变更）** → **每个受影响 galgame 一条 `galgame_revision`**（changed_fields=`['series_id']`），**不**落 taxonomy_revision

理由：membership 是 `galgame.series_id` 这个 FK 持有的，编辑这个 FK 是 galgame 行的编辑，不是 series 行的编辑。从 series 的修订历史看不到成员变化，从某 galgame 的修订历史能看到它何时加入/离开了什么 series。

---

下一节：[05 — 搜索](./05-search.md)
