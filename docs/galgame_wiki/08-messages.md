> [📖 文档索引](./README.md) · 上一节：[07 — 投稿](./07-submission.md) · 下一节：[附录](./99-appendix.md)

## 消息系统

wiki 起一张 `galgame_message` 表，沉淀投稿审核流程产生的所有事件。kungal / moyu 作为
消费方读取，admin web UI 直接读它显示队列。

设计文档：[docs/galgame_wiki/06-submission-and-review-design.md §5.7](../../galgame_wiki/06-submission-and-review-design.md)。

### 消息类型

| type | 触发 | actor | target | payload |
|---|---|---|---|---|
| `submitted` | 用户调 POST /galgame/submit | submitter | NULL | `{ "vndb_id": "..." }` |
| `claimed` | 用户调 POST /galgame/:gid/claim | claimer | NULL | `{ "from_status": 2 }` |
| `edited_pending` | 用户调 PATCH /galgame/:gid | submitter | NULL | `{ "field_count": 3 }` |
| `approved` | admin 调 PUT /admin/galgame/:gid/status status=0（源 status=3） | admin | 当前 owner | `{ "approved_by": <admin uid>, "note": "" }` |
| `declined` | admin 调 PUT /admin/galgame/:gid/status status=4 | admin | 当前 owner | `{ "declined_by": <admin uid>, "reason": "..." }` |
| `banned` | admin 调 PUT /admin/galgame/:gid/status status=1 | admin | 当前 owner | `{ "banned_by": <admin uid>, "reason": "" }` |
| `unbanned` | admin 调 PUT /admin/galgame/:gid/status status=0（源 status=1） | admin | 当前 owner | `{ "unbanned_by": <admin uid>, "note": "" }` |

> **没 target 的消息只给 admin 队列看**（`submitted` / `claimed` / `edited_pending`，因为没有用户需要被推送）。带 target 的（`approved` / `declined` / `banned` / `unbanned`）既会出现在用户的 `/messages/mine`，也会出现在 cron 的 `/messages/feed` 里。
>
> banned / unbanned 之所以也带 target，是因为 kungal/moyu 的 cron 需要这些事件来同步本地 `wiki_status_snapshot` 列。否则封禁过的作品在本地永远是 published 状态，渲染列表时会出现"指向不存在 galgame"的死链。

### 通用响应字段

每条 message 内嵌一个 galgame brief，避免消费端二次 batch：

```json
{
  "id": 42,
  "type": "approved",
  "galgame_id": 10000,
  "galgame": {
    "id": 10000,
    "name_zh_cn": "...",
    "name_ja_jp": "...",
    "effective_banner_hash": "...",
    "status": 0
  },
  "actor_user_id": 1,
  "target_user_id": 5,
  "payload": { "approved_by": 1 },
  "created_at": "2026-05-12T07:00:00Z"
}
```

`galgame` 字段是即时 JOIN 出来的，**始终反映 galgame 当前状态**（如果 galgame 被硬删了，
返回 NULL，消费端要做空判断）。

---

### GET /galgame/messages/mine

拉给当前用户的通知。**需要认证**（Bearer access_token）。

WHERE `target_user_id = JWT.uid AND id > since_id`，按 id 倒序。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| since_id | int64 | 0 | 只返 id > since_id 的（用于增量拉） |
| limit | int | 20 | max 100 |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "items": [...message 列表，id 倒序...],
    "total": 5
  }
}
```

**用法**：前端打开通知中心时调，配合本地 last_read_message_id 计算未读数。

---

### GET /galgame/messages/feed

服务到服务批量拉。**认证**：OAuth Client Basic Auth（同 [/users/batch](../oauth/03-cross-service.md#get-usersbatch)）。

WHERE `target_user_id IS NOT NULL AND id > since_id`，按 id **正序**（保证 cron 单调推进）。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| since_id | int64 | 0 | 上次拉到的最大 id |
| limit | int | 1000 | max 5000 |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "items": [...message 列表，id 正序...],
    "has_more": true
  }
}
```

`has_more`：true 表示 limit 卡满了，应该继续翻页（用 `items[last].id` 当下次 since_id）。

**用法**：kungal / moyu 每日 cron。详见 [07-submission.md §调用方 cron 同步本地 status](./07-submission.md#调用方-cron-同步本地-status)。

> **为什么 feed 排序与 mine 相反**：
> - mine：前端 UI 通常想看最新的 → 倒序
> - feed：cron 想线性消费完所有事件 → 正序 + cursor

---

### GET /admin/galgame/messages

admin 队列。**认证**：Bearer + admin/moderator role。

只返"未处理"的——JOIN galgame 表过滤掉 status 已经不是 3 的提交。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| type | string (csv) | `submitted,edited_pending` | 过滤类型 |
| page | int | 1 | |
| limit | int | 20 | max 100 |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 42,
        "type": "submitted",
        "galgame_id": 10000,
        "galgame": {
          "id": 10000,
          "name_zh_cn": "...",
          "effective_banner_hash": "...",
          "status": 3,
          "user_id": 5
        },
        "actor_user_id": 5,
        "actor": { "id": 5, "name": "kun", "avatar": "..." },
        "target_user_id": null,
        "payload": { "vndb_id": "v17" },
        "created_at": "2026-05-12T07:00:00Z"
      }
    ],
    "total": 23
  }
}
```

注意 admin 视图额外返了 `actor` brief（用户名 + 头像），方便后台 UI 显示"谁提交的"。

---

### 已读状态

**wiki 不维护已读**。每个消费端各自存。

**kungal / moyu 本地表建议**：

```sql
CREATE TABLE wiki_message_read_state (
    user_id              INT PRIMARY KEY,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

前端使用：

```typescript
// 打开消息面板
const state = await $fetch('/api/wiki/messages/read-state')
const messages = await $fetch('/api/wiki/messages/mine?since_id=0&limit=20')
const unreadCount = messages.filter(m => m.id > state.last_read_message_id).length

// 用户点"标记全部已读"
await $fetch('/api/wiki/messages/read-state', {
  method: 'PUT',
  body: { last_read_message_id: messages[0].id }  // 第一条最新
})
```

kungal / moyu 后端实现这两个端点（`/api/wiki/messages/read-state` GET / PUT），把状态存
自己库。**不要把已读状态写到 wiki**——跨站不一致。

---

### 错误响应

| HTTP | code | 触发 |
|---|---|---|
| 400 | 7 | since_id 不是非负整数 / limit 超界 |
| 401 | 10001 / 15001 / 15009 | 认证失败（按端点） |
| 403 | 10001 | admin endpoint 角色不够 |

---

### 数据生命周期

wiki 不主动清理消息。预计每年累计几千到几万行，长期可忽略存储成本。

如果未来量级显著上升：
- 加一个 cron 把 6 个月前已被消费的消息归档/删除
- 或对 `submitted` 类型加 TTL（galgame 状态已变就可删）

第一期不实现。

---

下一节：[附录](./99-appendix.md)
