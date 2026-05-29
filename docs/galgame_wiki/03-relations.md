> [📖 文档索引](./README.md) · 上一节：[02 — 版本历史 + PR](./02-revisions-and-prs.md) · 下一节：[04 — 分类轴 (Tag/Official/Engine/Series)](./04-taxonomy.md)

> 🔴 **下游强制范围**：本节的链接 / 别名 / 贡献者增删操作，**kungal 与 moyu 各自必须完整实现一份**（后端透传 Bearer 代理 + 前端 UI，功能与 wiki 对齐）。详见 [00-handbook §15](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。

## 链接

### GET /galgame/:gid/links

链接列表。返回**纯数组**（不分页，非 `{items, total}` 形态）：

```json
{
  "code": 0,
  "data": [
    { "id": 1, "galgame_id": 1, "name": "官网", "link": "https://...", "created": "...", "updated": "..." }
  ]
}
```

### POST /galgame/:gid/links

添加链接。**需要认证**。自动创建 revision。

```json
{
  "name": "官网",
  "link": "https://example.com"
}
```

### DELETE /galgame/:gid/links

删除链接。**需要认证**。自动创建 revision。

```json
{
  "id": 1
}
```

---

## 别名

### GET /galgame/:gid/aliases

别名列表。返回**纯数组**（不分页）：

```json
{
  "code": 0,
  "data": [
    { "id": 1, "galgame_id": 1, "name": "Fate/EXTRA", "created": "...", "updated": "..." }
  ]
}
```

### POST /galgame/:gid/aliases

添加别名。**需要认证**。自动创建 revision。

```json
{
  "name": "新别名"
}
```

### DELETE /galgame/:gid/aliases

删除别名。**需要认证**。自动创建 revision。

```json
{
  "id": 1
}
```

---

## 贡献者

### GET /galgame/:gid/contributors

贡献者列表（含用户信息）。

**成功响应**：

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "galgame_id": 1,
      "user_id": 1,
      "created": "...",
      "user": {
        "id": 1,
        "name": "KUN",
        "avatar": "https://..."
      }
    }
  ]
}
```

### DELETE /galgame/:gid/contributors/:id

删除贡献者。**需要认证**。仅 galgame 创建者或 admin 可操作。

---

## 封面 / 截图（PR2 新增）

封面 (`galgame_cover`) 与截图 (`galgame_screenshot`) 是 PR2 引入的新关联表，shape 一致。**没有独立的 CRUD 端点** —— 整套 CRUD 都内化在 `POST /galgame` / `PUT /galgame/:gid` / PR 提交端点上的 `covers` / `screenshots` 字段里（详见 [01-galgame.md PUT /galgame/:gid](./01-galgame.md#put-galgamegid)）。

**为什么没有独立端点**：覆盖/截图是 galgame 内容的一部分，与简介、标签同属一次事务。一图改动一条 revision 不利于历史可读；做成 batch 字段后，"改标题 + 改封面"是 1 条 revision 而不是 2 条。

| 字段 | covers | screenshots |
|------|--------|-------------|
| 主键 | `(galgame_id, image_hash)` | `(galgame_id, image_hash)` |
| `sort_order=0` 唯一 | ✅（DB 强制部分唯一索引 `idx_galgame_cover_pinned WHERE sort_order=0`） | ❌（无约束） |
| 派生展示字段 | `galgame.effective_banner_hash`（= `sort_order=0` 那张的 `image_hash`） | 无（前端按 `sort_order` 排序展示） |
| 同 PR2 之前的 banner | 取代 `banner_image_hash`（PR5 退役） | 取代旧 album / preview 链接 |

**字段 shape**（提交时的 CoverInput / ScreenshotInput，详见 `apps/api/internal/platform/galgame/model/snapshot.go` 的 `SnapshotCover` / `SnapshotScreenshot`）：

```json
{
  "image_hash": "abcd1234...ef",
  "sort_order": 0,
  "sexual": 0,
  "violence": 0,
  "source": "user",
  "source_key": ""
}
```

screenshots 额外有 `caption` 字段（短描述，可空）。

**presence 语义**：与 `tag_ids` 一致 —— 不传 = 保持不变；传 `[]` = 清空；传非空数组 = 权威全量替换。

**multipart 上传**：见 [01-galgame.md Banner 上传段](./01-galgame.md#banner-上传通过-create--update--pr-端点的-multipart-模式)；后端把上传得到的 hash 通过 `PromoteCoverHash` 合并进 covers（提升或新增 `sort_order=0`）。screenshots 暂无 multipart 直传，需要前端先经 image_service 拿 hash 再随 `screenshots` 字段提交。

---

下一节：[04 — 分类轴](./04-taxonomy.md)
