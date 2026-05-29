> [📖 文档索引](./README.md) · 上一节：[06 — 管理统计](./06-admin.md) · 下一节：[08 — 消息](./08-messages.md)

## 用户投稿与审核

让 kungal / moyu 的普通用户也能创建新 galgame。架构设计参见
[docs/galgame_wiki/06-submission-and-review-design.md](../../galgame_wiki/06-submission-and-review-design.md)。

### 工作流概览

```
用户点 "发布 galgame"
  → 搜索 (GET /galgame/search?include_pending=true)
  ├── 命中已发布 (status=0)：直接 INSERT 本地 galgame_stats，跳详情页
  ├── 命中 VNDB 草稿 (status=2)：POST /galgame/:gid/claim → 翻转为 0 → INSERT stats
  ├── 命中自己的待审 (status=3)：提示"已提交，等待审核"
  └── 都没命中：POST /galgame/submit → 创建 status=3 草稿 → INSERT stats

审核 (admin 在 wiki 后台操作)
  → PUT /admin/galgame/:gid/status { status: 0 } / { status: 4, reason }
  → 写 message 通知提交者

kungal/moyu cron 每日
  → GET /galgame/messages/feed?since_id=X (Basic Auth)
  → 同步本地 galgame_stats.wiki_status_snapshot
```

### Status 取值

| status | 含义 |
|---|---|
| 0 | 已发布（全站可见） |
| 1 | 封禁 |
| 2 | VNDB 草稿（系统同步，可被任意用户认领） |
| 3 | 用户提交，待审核 |
| 4 | 审核拒绝（用户可修改后重交） |

---

### POST /galgame/submit

普通用户提交新 galgame 申请。**需要认证**（Bearer access_token，任意角色）。

**请求体**：

```json
{
  "vndb_id": "v17",
  "name_en_us": "Title",
  "name_ja_jp": "タイトル",
  "name_zh_cn": "标题",
  "name_zh_tw": "標題",
  "banner": "",
  "intro_zh_cn": "...",
  "content_limit": "sfw",
  "original_language": "ja-jp",
  "age_limit": "r18",
  "series_id": null,
  "aliases": "别名1,别名2",
  "tag_ids": [1, 2],
  "official_ids": [3],
  "engine_ids": [],
  "release_date": "2019-08-16",
  "release_date_tba": false,
  "covers": [
    {"image_hash": "abcd1234...ef", "sort_order": 0, "sexual": 0, "violence": 0, "source": "user", "source_key": ""}
  ],
  "screenshots": []
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| vndb_id | 否 | **可空**——原创 / 独立作品没有 VNDB 编号时留空。非空时格式 `v\d+`，且必须全局唯一 |
| name_* | 否 | 至少填一个语言名（实践要求；后端不强制） |
| banner | 否 | 旧 URL fallback；新封面通过 `covers[sort_order=0]` 或 multipart `file` 表达（PR5 起 `banner_image_hash` 字段已退役） |
| covers / screenshots | 否 | image_service hash 数组，见 [03-relations.md 封面/截图](./03-relations.md#封面--截图pr2-新增) |
| release_date / release_date_tba | 否 | PR1 新字段；详见 [01-galgame.md](./01-galgame.md#put-galgamegid) |
| 其他 | 否 | 同 [POST /galgame](./01-galgame.md#post-galgame) |

支持 `multipart/form-data` 直接带 banner 文件（同 [POST /galgame 的 multipart 模式](./01-galgame.md#banner-上传通过-create--update--pr-端点的-multipart-模式)）。

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "id": 10000,
    "status": 3,
    "vndb_id": "v17",
    "name_zh_cn": "标题",
    ...完整 galgame 字段...
  }
}
```

**错误响应**：

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | 7 | 参数验证失败 |
| 400 | 20003 | vndb_id 格式非法（非空但不匹配 `v\d+`） |
| 400 | 20004 | vndb_id 已存在 |
| 429 | 20009 | 今日提交配额已用尽（默认 5 条/天） |

---

### POST /galgame/:gid/claim

认领一个 VNDB 草稿并直接发布。**需要认证**（任意角色）。

**前置**：目标 galgame `status == 2`。

**Behaviour**：

1. 把 galgame.status 翻转为 0
2. galgame.user_id 改为认领者
3. 加 contributor
4. 写 revision (action='claimed')
5. 写 message (type='claimed', target=NULL)

**请求体**：空对象 `{}`。

**成功响应**：返回认领并发布后的完整 galgame（status=0）。

**错误响应**：

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | 20001 | galgame 不存在 |
| 400 | 20006 | galgame 当前状态不是 2（不是 VNDB 草稿，不能认领） |

---

### PATCH /galgame/:gid

修改自己的待审或被拒草稿。**需要认证**。仅 `user_id == 当前用户 AND status IN (3, 4)` 时允许。

修改后 status=4 → 自动翻回 status=3（重入审核队列）。

**请求体**：同 [PUT /galgame/:gid](./01-galgame.md#put-galgamegid)（所有字段可选）。

**Behaviour**：

- UPDATE 字段
- 若原 status=4，UPDATE status=3
- 写 revision (action='edited_pending')
- 写 message (type='edited_pending', target=NULL)

**成功响应**：返回更新后的 galgame。

**错误响应**：

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | 20001 | galgame 不存在 |
| 403 | 20007 | 不是提交者本人 |
| 400 | 20008 | galgame 当前 status 不在 {3, 4}（已发布或封禁的不能走 PATCH，必须走 PUT） |

> 区分：
> - **PUT /galgame/:gid** — 已发布条目的直接编辑（creator 或 admin 可调，写 'updated' revision）
> - **PATCH /galgame/:gid** — 待审稿的修订（仅提交者可调，写 'edited_pending' revision，会触发重审）

---

### DELETE /galgame/:gid

撤回自己的草稿。**需要认证**。

**前置**：`user_id == 当前用户 AND status IN (3, 4)`。

**Behaviour**：硬删（CASCADE 清掉关联表）。**已发布的 galgame 不能走这个端点**，要走
admin status=1 (ban)。

**响应**：`{ "code": 0 }`

**错误响应**：

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | 20001 | galgame 不存在 |
| 403 | 20007 | 不是提交者本人 |
| 400 | 20008 | 当前 status 不在 {3, 4} |

---

### GET /galgame/mine

列出当前用户的所有提交。**需要认证**。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|---|---|---|---|
| status | int (csv) | `3,4` | 过滤状态；不传则默认列出 pending + declined |
| page | int | 1 | |
| limit | int | 20 | max 50 |

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 10000,
        "status": 4,
        "name_zh_cn": "标题",
        "name_ja_jp": "...",
        "name_en_us": "...",
        "name_zh_tw": "...",
        "banner": "",
        "effective_banner_hash": "abcd...",
        "content_limit": "sfw",
        "vndb_id": "v17",
        "created": "2026-05-12T07:00:00Z",
        "updated": "2026-05-12T08:30:00Z",
        "decline_reason": "VNDB ID 错误，请核对后重新提交"
      },
      {
        "id": 10001,
        "status": 3,
        "name_zh_cn": "另一个",
        "vndb_id": "",
        "created": "2026-05-12T09:00:00Z",
        "updated": "2026-05-12T09:00:00Z"
      }
    ],
    "total": 2
  }
}
```

`decline_reason` 仅 status=4 时出现，从该 galgame 最近一条 admin `declined` 消息的 payload 提取。status=3 / 0 / 其他时该字段省略（JSON `omitempty`）。让"我的提交"页可以一次拉齐"被拒了+原因"，不需要再调 `/messages/mine` 关联。

---

### GET /galgame/search 增量参数

```
GET /galgame/search?q=...&include_pending=true
Authorization: Bearer <access_token>
```

| 行为 |
|---|
| 默认 `include_pending=false`：行为不变，只返 status=0 |
| `include_pending=true` 且带 JWT：响应中**额外**返一段 `pending` 列表，含 `status IN (3,4) AND user_id = JWT.uid` 命中的条目 |
| `include_pending=true` 但无 JWT：忽略参数，按 false 走 |

**响应结构变化**：

```json
{
  "code": 0,
  "data": {
    "items": [...status=0 的命中],
    "pending": [...自己的 status=3/4 的命中],
    "total": 42,
    "facets": {...},
    "processing_time_ms": 3
  }
}
```

前端使用建议：把 `pending` 单独渲染为"等待审核中"区块。

---

### GET /galgame/batch 增量行为

不需要传额外参数。**有 Bearer Authorization 时**，自动把"调用者作为提交者的
status=3/4 条目"也纳入返回；无 Bearer 时行为完全不变（只返 status=0）。

```
GET /api/galgame/batch?ids=1,2,3
Authorization: Bearer <access_token>     ← 可选
```

→ 返回的 items 中 status 字段可能是 0 / 3 / 4，调用方可自行判断是否展示。

---

### Admin 审核（已存在端点行为扩展）

```
PUT /api/admin/galgame/:gid/status      认证：Bearer + admin/moderator
Body: { "status": 0 | 1 | 4, "reason": "..." }
```

新 status 值含义：

| 目标 status | 行为 |
|---|---|
| 0 | 从 status=3 → approved（写 revision('approved') + message('approved', target=owner））；从 status=1 → unbanned（写 revision('unbanned') + message('unbanned', target=owner)）；其它源 status 当 approved 处理 |
| 4 | 仅源 status=3 时允许；写 revision('declined') + message('declined', target=owner) |
| 1 | 封禁（任意源 status）；写 revision('banned') + message('banned', target=owner) |

> `status=2` **不允许**作为目标——admin 不应该把人为内容退回 VNDB 草稿态。

> `status=3` 作为目标也**不允许**——admin 不主动新建待审，待审是用户提交产生的。

---

### 调用方 SDK 建议（kungal/moyu 后端）

```go
type GalgameClient struct {
    // ...
}

// 用户身份接口（透传 access_token）
func (c *GalgameClient) Submit(ctx, token string, req SubmitRequest) (*Galgame, error)
func (c *GalgameClient) Claim(ctx, token string, gid int) (*Galgame, error)
func (c *GalgameClient) PatchDraft(ctx, token string, gid int, req PatchRequest) (*Galgame, error)
func (c *GalgameClient) DeleteDraft(ctx, token string, gid int) error
func (c *GalgameClient) ListMine(ctx, token string, req ListMineRequest) (*Page, error)
func (c *GalgameClient) SearchWithPending(ctx, token string, q string) (*SearchResult, error)

// 服务身份接口（OAuth Client Basic Auth）
func (c *GalgameClient) MessageFeed(ctx, sinceID int64, limit int) ([]Message, error)
```

每个 user-facing 方法都把用户的 access_token 透传给 wiki，让 wiki 用 JWT.uid 判定身份。
不要在 kungal 后端"代为决定身份"。

---

### 调用方 cron 同步本地 status

每日凌晨：

```sql
-- kungal 本地
SELECT MAX(wiki_message_last_id) FROM cron_state WHERE name = 'wiki_msg_sync';
```

调 `GET /galgame/messages/feed?since_id=<last>&limit=1000` 拉增量。对每条：

```go
switch msg.Type {
case "approved", "unbanned":
    UPDATE galgame_stats SET wiki_status_snapshot = 0 WHERE galgame_id = msg.GalgameID
case "declined":
    UPDATE galgame_stats SET wiki_status_snapshot = 4 WHERE galgame_id = msg.GalgameID
case "banned":
    UPDATE galgame_stats SET wiki_status_snapshot = 1 WHERE galgame_id = msg.GalgameID
}
```

#### 为什么 cron 只处理这 4 种 type

`/messages/feed` 按 `target_user_id IS NOT NULL` 过滤，刚好对应这 4 种 admin 触发的事件。**kungal/moyu 自己触发**的操作（submit / claim / patch / delete）不需要 cron 同步——这些请求是 kungal 后端发出去的，wiki 的同步返回里就带了最新 `status`，按返回值更新本地即可：

```go
// kungal 后端 — 用户提交
result, _ := wiki.Submit(ctx, token, req)
db.Exec(`INSERT INTO galgame_stats(galgame_id, wiki_status_snapshot) VALUES (?, ?)`,
    result.ID, result.Status) // status=3

// kungal 后端 — 用户认领草稿
result, _ := wiki.Claim(ctx, token, gid)
db.Exec(`INSERT INTO galgame_stats(galgame_id, wiki_status_snapshot) VALUES (?, ?)
         ON CONFLICT (galgame_id) DO UPDATE SET wiki_status_snapshot = EXCLUDED.wiki_status_snapshot`,
    result.ID, result.Status) // status=0

// kungal 后端 — 用户编辑草稿（可能从 4 翻回 3）
result, _ := wiki.PatchDraft(ctx, token, gid, req)
db.Exec(`UPDATE galgame_stats SET wiki_status_snapshot = ? WHERE galgame_id = ?`,
    result.Status, result.ID) // status=3

// kungal 后端 — 用户撤回草稿
_ = wiki.DeleteDraft(ctx, token, gid)
db.Exec(`DELETE FROM galgame_stats WHERE galgame_id = ?`, gid)
```

cron 是"被动事件流"，处理 admin 主动操作；用户操作走"主动 RPC + 同步本地"——两条路职责清晰互不重叠。

#### 处理 `galgame` 字段为 null 的"幽灵消息"

如果某用户撤回了一份已经被 approve 过、又被 ban 过的草稿（或类似不常见路径），消息表里的事件仍会保留（设计选择），但 enrich 时 `galgame` 字段为 null。cron 要 gracefully skip：

```go
for _, msg := range messages {
    if msg.Galgame == nil {
        // galgame 已不存在 — 清掉本地 stats 行（如果还有）
        db.Exec(`DELETE FROM galgame_stats WHERE galgame_id = ?`, msg.GalgameID)
        continue
    }
    switch msg.Type { ... }
}
```

记下 `max(id)` 写回 cron_state。

> 即便 cron 漏跑，前端打开详情时也可以"机会主义校验"——拉 `/galgame/:gid` 看
> 实际 status，发现与本地缓存不一致就刷新一行。但这不是必需的。

---

下一节：[08 — 消息](./08-messages.md)
