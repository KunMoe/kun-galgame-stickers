> [📖 文档索引](./README.md) · 上一节：[01 — Galgame 核心 CRUD](./01-galgame.md) · 下一节：[03 — 链接 / 别名 / 贡献者](./03-relations.md)

> 🔴 **下游强制范围**：本节的修订历史与 PR 编辑操作，**kungal 与 moyu 各自必须完整实现一份**（后端透传 Bearer 代理 + 前端 UI，功能与 wiki 对齐，不得只做子集）。已不再是「wiki-only，下游不做」。详见 [00-handbook §15](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。

## 版本历史 (Wiki)

每次编辑（创建、更新、PR 合并、回滚）都会创建一个 revision，存储 galgame 的完整状态快照。

### GET /galgame/:gid/revisions

版本列表。

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| limit | int | 20 | 每页数量 |
| include_minor | bool | false | 是否包含小修改 |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 3,
        "galgame_id": 1,
        "revision": 3,
        "user_id": 2,
        "action": "merged",
        "note": "更新简介",
        "is_minor": false,
        "reverted_to": null,
        "created": "2026-01-03T00:00:00Z"
      },
      {
        "id": 2,
        "revision": 2,
        "action": "updated",
        "note": "",
        "is_minor": true
      },
      {
        "id": 1,
        "revision": 1,
        "action": "created",
        "note": ""
      }
    ],
    "total": 3
  }
}
```

`action` 取值：`created`, `updated`, `merged`, `reverted`, `declined`

> **PR3/PR5 注意 — revision.note 可能由系统追写**：当 `reverted` 或 `merged` 应用的旧快照里有引用已不存在的 image_hash（image_service 端 TTL 清理掉了），后端会自动把这些 hash 从 covers/screenshots 里剔除并在 `note` 后追加 `\n[系统] reverted 时检测到 N 张图片在 image_service 已不可用，已自动从快照中剔除` 之类的说明。下游展示 revision/PR 详情时直接渲染 `note` 即可，不需要额外解析。

---

### GET /galgame/:gid/revisions/:rev

查看特定版本的完整快照。

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "galgame_id": 1,
    "revision": 1,
    "user_id": 1,
    "action": "created",
    "snapshot": {
      "vndb_id": "v12345",
      "name_zh_cn": "标题",
      "aliases": ["别名1"],
      "tag_ids": [1, 2],
      "official_ids": [1],
      "engine_ids": [],
      "links": [{"name": "VNDB", "link": "https://vndb.org/v12345"}],
      "release_date": "2019-08-16",
      "release_date_tba": false,
      "covers": [
        {"image_hash": "abcd1234...ef", "sort_order": 0, "sexual": 0, "violence": 0, "source": "user", "source_key": ""}
      ],
      "screenshots": [
        {"image_hash": "fedcba98...12", "sort_order": 0, "caption": "CG 01", "sexual": 0, "violence": 0, "source": "", "source_key": ""}
      ]
    },
    "created": "..."
  }
}
```

---

### GET /galgame/:gid/revisions/:rev/diff

计算该版本与前一版本的差异（实时计算，不存储）。

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "changed_keys": {
      "name_zh_cn": true,
      "tag_ids": true
    },
    "old": {
      "name_zh_cn": "旧标题",
      "tag_ids": [1, 2]
    },
    "new": {
      "name_zh_cn": "新标题",
      "tag_ids": [1, 2, 3]
    },
    "names": {
      "tags": { "1": "校园", "2": "治愈", "3": "RPG" },
      "officials": {},
      "engines": {},
      "series": {}
    }
  }
}
```

`old` 和 `new` 是完整的 snapshot 对象，前端可以只展示 `changed_keys` 中标记的字段。对于大文本字段（intro_*），前端可以用 diff 库展示行级差异。

> 🆕 **2026-05-22 (K-PR)**：响应新增 `names` 字段。包含本次 diff 涉及的所有 tag / official / engine / series ID 到**显示名称**的映射。让前端直接渲染"tag added: 校园, 治愈"而不是"tag added: 1, 2"，避免 N+1 后续请求。
>
> **被删除的实体**：如果某个 ID 在 snapshot 里存在但 DB 里已被 soft-delete / 硬删，对应 key **不会出现在 names 里**。前端应该 fallback 渲染成 `已删除 #<id>`。
>
> **性能**：union(old.ids + new.ids) 去重后做 4 次 `WHERE id IN (...)` PK 查询（每次 ~1ms）。典型 galgame ≤30 tags + ≤5 officials + ≤2 engines + ≤1 series，每次 diff 请求增加 ~5ms DB 时间 + ~5KB 响应体。可忽略。

---

### POST /galgame/:gid/revert

回滚到指定版本。**需要认证**。仅创建者或 admin 可操作。

回滚会创建一个新 revision（action=reverted），不会删除历史记录。

**请求体**：

```json
{
  "revision": 1
}
```

---

## PR (编辑请求)

非创建者/非 admin 通过 PR 提交编辑。PR 支持字段级自动 rebase。

### GET /galgame/:gid/prs

PR 列表。

**查询参数**：

| 参数 | 类型 | 默认值 |
|------|------|--------|
| page | int | 1 |
| limit | int | 20 |

---

### GET /galgame/:gid/prs/:id

PR 详情，包含与 base revision 的差异。

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "pr": {
      "id": 1,
      "galgame_id": 1,
      "user_id": 2,
      "status": 0,
      "note": "修改标题",
      "base_revision": 1,
      "snapshot": { ... },
      "completed_by": null,
      "revision_id": null,
      "created": "..."
    },
    "changed_keys": {
      "name_zh_cn": true
    },
    "names": {
      "tags": { "1": "校园", "3": "RPG" },
      "officials": { "5": "Key" },
      "engines": {},
      "series": {}
    }
  }
}
```

`status`：`0` = pending, `1` = merged, `2` = declined

> 🆕 **2026-05-22 (K-PR)**：响应同样新增 `names` 字段，覆盖 base + pr snapshot 涉及的 tag / official / engine / series ID。语义和性能详见 [GET /galgame/:gid/revisions/:rev/diff](#get-galgamegidrevisionsrevdiff) 末尾 callout。

---

### POST /galgame/:gid/prs

提交 PR。**需要认证**。

提交时只需提供要修改的字段，未提供的字段保持当前值。

**支持两种 Content-Type**：
- `application/json` — 普通 PR
- `multipart/form-data` — PR 提案里直接附 banner 文件，reviewer 看 diff 时可直接看到新图缩略图。详见
  [Banner 上传](#banner-上传通过-create--update--pr-端点的-multipart-模式)

**请求体**（JSON 模式）：

```json
{
  "name_zh_cn": "新标题",
  "tag_ids": [1, 2, 3],
  "note": "修改标题和标签"
}
```

支持的字段与创建/更新 galgame 相同，另外支持：

| 字段 | 类型 | 说明 |
|------|------|------|
| aliases | string[] | 别名数组（替换全部） |
| tag_ids | int[] | 标签 ID 数组（替换全部） |
| official_ids | int[] | 开发商 ID 数组（替换全部） |
| engine_ids | int[] | 引擎 ID 数组（替换全部） |
| links | object[] | 链接数组 `[{name, link}]`（替换全部） |
| covers | CoverInput[] | image_service hash 数组（替换全部），`sort_order=0` = 钉住封面（同 01-galgame.md PUT 端点说明） |
| screenshots | ScreenshotInput[] | image_service hash 数组（替换全部），无"钉住"约束 |
| release_date | string | `YYYY-MM-DD` 字符串或 `""`（未知） |
| release_date_tba | bool | 是否官方已宣布日期未定 |
| note | string | PR 说明 |

---

### PUT /galgame/:gid/prs/:id/merge

合并 PR。**需要认证**。仅 galgame 创建者或 admin 可操作。

合并时如果 PR 的 base_revision 落后于最新版本，系统会自动检查字段冲突：
- **无冲突**：自动 rebase（PR 的改动应用到最新版本上）
- **有冲突**：返回错误，列出冲突字段名

**冲突响应示例**：

```json
{
  "code": 10,
  "message": "字段冲突: name_zh_cn 已被其他编辑修改，请基于最新版本重新提交"
}
```

---

### PUT /galgame/:gid/prs/:id/decline

拒绝 PR。**需要认证**。仅 galgame 创建者或 admin 可操作。

---

## 标签 / 厂商 / 引擎 / 系列 修订（PR4 新增）

Galgame 的 4 种 taxonomy 实体（tag / official / engine / series）现也各自具备版本历史与回滚能力，端点形态与本节的 galgame 修订高度对齐：

- 端点：`GET /tag/:id/revisions`、`GET /tag/:id/revisions/:rev`、`POST /tag/:id/revert`（official / engine / series 同形）
- 修订表统一在 `taxonomy_revision`（polymorphic `entity` + `target_id`），与 `galgame_revision` 物理分离但语义对齐。
- Series 的 `galgame_ids` 变更不进 `taxonomy_revision`，而是**为受影响的 galgame 各自写一条 `galgame_revision`**（owner 视角能看到自己作品被加入/移出 series）；Series 的 name/desc 仍走 taxonomy_revision。
- 完整规约见 [04-taxonomy.md §修订与回滚](./04-taxonomy.md#修订与回滚-pr4-新增4-实体同款)。

---

下一节：[03 — 链接 / 别名 / 贡献者](./03-relations.md)
