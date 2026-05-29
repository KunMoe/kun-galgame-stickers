> [📖 文档索引](./README.md) · 上一节：[05 — 搜索](./05-search.md) · 下一节：[附录 — 错误码与端点总览](./99-appendix.md)

## 管理统计 (Admin)

### GET /admin/stats

Wiki 管理统计接口，返回各实体的总量和每日新增计数。

**查询参数**：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| days | int | 否 | 30 | 查询最近 N 天 (1-365) |

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "totals": {
      "galgame_tag": 1234,
      "galgame_official": 567,
      "galgame_engine": 89,
      "galgame_series": 234,
      "galgame_link": 4567,
      "galgame_pr": 123,
      "galgame_revision": 890
    },
    "daily": [
      {
        "date": "2026-04-10",
        "galgame_tag": 3,
        "galgame_official": 1,
        "galgame_engine": 0,
        "galgame_series": 2,
        "galgame_link": 15,
        "galgame_pr": 4,
        "galgame_revision": 12
      }
    ]
  }
}
```

| 字段 | 说明 |
|------|------|
| totals | 各表全量 COUNT |
| daily | 按日期升序排列，每天一行，没有数据的日期不返回 |
| date | 格式 YYYY-MM-DD，与 `date_trunc('day', created)::date::text` 一致 |

统计的 7 个维度：

| 字段 key | 对应表 |
|----------|--------|
| galgame_tag | galgame_tag |
| galgame_official | galgame_official |
| galgame_engine | galgame_engine |
| galgame_series | galgame_series |
| galgame_link | galgame_link |
| galgame_pr | galgame_pr |
| galgame_revision | galgame_revision |

### GET /admin/galgame

管理视角的 galgame 列表（**可跨 status 查询**，区别于公开的 `/galgame` 只返回 `status=0`）。**需要认证**。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| status | int | — | 不传即不过滤；可传 `0`（已发布）/ `1`（封禁）/ `2`（草稿）|
| search | string | — | ILIKE 匹配 vndb_id + 4 语言 name |
| page | int | 1 | |
| limit | int | 20 | max 100 |

**响应**：`{ items: [...galgame], total: int }`

### GET /admin/galgame/:gid

管理视角的 galgame 详情（任意 status，preload 全部关联）。**需要认证**。返回 `Galgame` 对象含 `Alias`, `Tag.Tag`, `Official.Official`, `Engine.Engine`, `Series`。

### PUT /admin/galgame/:gid/status

修改 galgame 状态（发布 / 封禁 / 拒绝）。**需要认证**（admin/moderator）。

**请求体**：

```json
{ "status": 0, "reason": "可选审核备注" }
```

| 目标 status | 含义 | 副作用 |
|---|---|---|
| `0` | 已发布 | 若源 status=3 → 写 message(type='approved', target=submitter)；同时一条 revision |
| `1` | 封禁 | 写 message(type='banned', target=NULL) |
| `4` | 审核拒绝 | 仅源 status=3 时允许；写 message(type='declined', target=submitter, payload.reason) |

`status=2` / `status=3` **不允许**作为目标 —— 2 是 sync-vndb 专属域，3 是用户提交产生的。

**响应**：`{ "code": 0, "data": { "id": <gid>, "status": <new> } }`

完整审核流程见 [07 — 投稿](./07-submission.md)。

---


---

下一节：[附录](./99-appendix.md)
