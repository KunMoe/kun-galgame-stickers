# Galgame 创建与 Wiki 联动 — 下游接入手册（kungal / moyu）

> 此文档面向 **kungal / moyu 团队**，把"用户在站点发布一个 galgame"的端到端流程、wiki 提供的能力、下游需要实现的工作量、所有商议决策一次性梳理清楚。
>
> 设计文档：[docs/galgame_wiki/06-submission-and-review-design.md](../../galgame_wiki/06-submission-and-review-design.md)
> 调用方 API：[07-submission.md](./07-submission.md) · [08-messages.md](./08-messages.md) · [01-galgame.md](./01-galgame.md) · [05-search.md](./05-search.md)

---

## 1. 总体目标与角色

**Wiki 是 galgame 元数据的唯一可信源（SoT）**。kungal / moyu 不再独立维护 galgame 主表，所有 galgame 的"身份"由 wiki 颁发整数 `id`，所有展示字段（name / banner / intro / tag / official 等）每次渲染从 wiki 拉取。

| 角色 | 数据所有 | 接口角色 |
|---|---|---|
| **Wiki**（`:9280`） | galgame、tag、official、engine、series、revision、PR、**galgame_message** | 服务端 |
| **kungal / moyu**（各自后端） | 本地交互数据：`galgame_stats`、`galgame_like`、`galgame_comment`、`galgame_resource`、`galgame_rating` | 上游消费者 |
| **OAuth**（`:9277`） | users、roles、oauth_client | 身份提供方 |

**三库 user_id 已全局对齐**（migrate-users 完成），不需要 ID 映射；移交过来的 user_id 直接是 OAuth 全局 user_id。

> **接口角色补充（强制）**：kungal / moyu 不只是「投稿 + 渲染」的消费者，**还必须各自完整承载 galgame 的编辑面**（PR、修订历史、关系、分类轴的增删改），后端代理 + 前端 UI 一份不少。详见 [§15](#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。wiki 仍是数据 SoT 与服务端逻辑实现方，但「下游只读、编辑归 wiki」的旧定位已作废。

---

## 2. status 状态机（5 档）

| status | 含义 | 谁能在 wiki 看 | 触发来源 |
|---|---|---|---|
| 0 | 已发布 | 所有人 | admin 直接 POST、claim 草稿、approved 审核 |
| 1 | 封禁 | admin | admin `PUT /admin/galgame/:gid/status` |
| 2 | VNDB 草稿（系统建） | admin + 任何用户走 claim 入口 | `sync-vndb` 每日 cron |
| **3** | **用户提交，待审核** | admin + 提交者本人 | `POST /galgame/submit` |
| **4** | **审核拒绝** | admin + 提交者本人 | admin decline 审核结果 |

完整转换表见 [06-submission-and-review-design.md §3](../../galgame_wiki/06-submission-and-review-design.md#3-status-状态机扩展)。

---

## 3. 五个已敲定的决策

实施前请通读，避免实现时再回头讨论。

### 决策 1 — 萌萌点时机：**推迟到通过审核**

| 路径 | 萌萌点 |
|---|---|
| 用户调 `POST /galgame/submit` | **0**（提交不奖励） |
| 用户调 `POST /galgame/:gid/claim` | **+3**（claim 直接 published） |
| moyu cron 收到 `approved` 消息 | **+3**（在 cron 事务里写一行 user.moemoepoint += 3） |
| `declined` / `delete draft` | **不退**（一开始就没给） |

**理由**：submit 即时 +3 会激励刷垃圾投稿；只奖励"通过审核"才是正确的质量信号。

**Cron 频率**：approved/declined/banned/unbanned 同步建议 **5–15 分钟一次**（不是每日），否则用户等"被通过"的萌萌点会延迟一整天，体验差。增量拉取（带 `since_id`），单次成本极低。

### 决策 2 — 本地 `wiki_status_snapshot` 列：**不加**

`galgame_stats` 保持纯计数器表，**不加 status 影子列**。理由链：

- 公开列表：wiki batch 匿名只返 status=0，**天然过滤**，本地不需要再过滤一层
- "我的提交"页：直接代理 `GET /galgame/mine`，本地不查
- 被封禁后清理：cron 收到 `banned` 消息 → `DELETE FROM galgame_stats WHERE galgame_id = ?`

**galgame_stats 何时 INSERT**：**懒加载**——只在首次互动（点赞、评分、评论、加资源）时 `INSERT ... ON CONFLICT DO UPDATE`。**提交时不 INSERT**，直到用户在本站对该作品产生第一次实际交互。

### 决策 3 — Basic Auth 凭据：**复用现有 OAuth Client 凭证**

Wiki 的 `oauth_client` 表跟 OAuth 服务**共享同一张表**（在 `kun_oauth_admin` 库里，wiki 只读连进去）。

moyu 现有调 `OAuth /users/batch` 的 `client_id / client_secret`，**直接拿来调 wiki `/galgame/messages/feed`**——零新配置。

### 决策 4 — 何时带 Bearer

| 场景 | 是否带 Bearer | 原因 |
|---|---|---|
| 首页 / 默认列表 / 默认搜索 | **不带** | 用户期望首页是"公共已发布"视图，不要混入自己的 pending |
| "发布 galgame" 向导的搜索框 | **带 + `?include_pending=true`** | 帮用户发现"你已提过这个" |
| `/user/:id/galgames`（任意人作品页） | **不带** | 只看 status=0 |
| "我的提交"页 | **不调 search/batch**，直接代理 `GET /galgame/mine` | 一条 RPC 拿齐 |
| 详情页 | **不带**（详情端点对任意 status 开放，owner 也能看） | wiki 详情逻辑已处理 |

### 决策 5 — `POST /galgame` 已锁到 admin/moderator

普通用户**只能**走 `POST /galgame/submit`（创建 status=3，进入审核队列）。`POST /galgame` 返回 403。

moyu 的"发布 galgame" 路径**只暴露 submit**。如果之前后端有调 `POST /galgame` 的代码，删掉或改 submit。

---

## 4. 四种发布场景的完整数据流

### 场景 A：用户搜索时命中**已发布**条目

```
moyu 用户输入"Fate"
  ↓
moyu 前端 → moyu 后端 /api/galgame/search-for-publish?q=Fate
  ↓ (带用户 access_token)
moyu 后端 → wiki GET /galgame/search?q=Fate&include_pending=true
  ↓ (Authorization: Bearer <access_token>)
wiki 返回 { items: [...status=0], pending: [...自己的 3/4] }
  ↓
用户在"已发布"列表里点选《Fate/stay night》(id=8329)
  ↓
moyu 后端 → 跳详情页 /galgame/8329
  ↓
moyu 详情页加载 → 调 wiki /galgame/batch?ids=8329 拿展示字段 + 调本地 galgame_stats(8329)
  ↓
首次互动时（如点赞）→ INSERT galgame_stats(galgame_id=8329, like_count=1)
```

**关键点**：
- moyu 后端不在 wiki 触发任何写操作
- 不 INSERT galgame_stats（懒加载）
- 不奖励萌萌点（用户没"创建"任何东西）

### 场景 B：用户搜索时命中 **VNDB 草稿**（status=2）

```
moyu 用户输入"v17"
  ↓
moyu 前端 → moyu 后端 → wiki search
  ↓
wiki 返回 status=2 草稿（VNDB 同步过来的 Fate/stay night）
  ↓
前端在"VNDB 草稿（一键认领发布）"区块显示
  ↓
用户点击「认领并发布」
  ↓
moyu 后端 → wiki POST /galgame/8329/claim (透传 access_token)
  ↓
wiki 事务:
  - status 2 → 0
  - user_id = 用户的 uid
  - 加 contributor (uid)
  - 写 revision (action='claimed')
  - 写 message (type='claimed', target=NULL)  ← admin 知道有人认领了
  ↓
返回 status=0 的 galgame 对象
  ↓
moyu 后端事务:
  - INSERT galgame_stats(galgame_id=8329, all zeros) ← 这里需要 INSERT，因为 claim 即发布
  - UPDATE user.moemoepoint += 3 (决策 1：claim 即时奖励)
  ↓
跳详情页
```

**关键点**：
- claim 即时奖励 +3（claim 流程没有"审核"环节）
- INSERT galgame_stats（用户首次在本站"拥有"此 galgame）
- 不需要等 cron——同步在 RPC 返回时就处理本地副作用

### 场景 C：用户搜索时命中**自己的 pending**（status=3 或 4）

```
moyu 用户输入"我的新作"
  ↓
moyu 后端 → wiki search?include_pending=true (带 Bearer)
  ↓
wiki 返回 { items: [...], pending: [{id: 10000, status: 3, user_id=<我>, ...}] }
  ↓
前端在"等待审核中"区块显示
  ↓
用户点击 → 跳到 moyu 的"我的提交"页 /me/submissions
  ↓
moyu 后端 → wiki GET /galgame/mine (带 Bearer)
  ↓
wiki 返回 { items: [{id: 10000, status: 3, ...}, {id: 9999, status: 4, decline_reason: "..."}] }
  ↓
前端展示，用户看到 pending + 被拒原因
  ↓
如果状态是 4，用户点"编辑后重新提交"
  ↓
moyu 后端 → wiki PATCH /galgame/9999 (带 Bearer + body)
  ↓
wiki 事务:
  - UPDATE galgame fields
  - status 4 → 3 (自动翻回)
  - revision (action='edited_pending')
  - message (type='edited_pending', target=NULL)
  ↓
返回 status=3 的 galgame
  ↓
moyu 后端不需要做本地 update（galgame_stats 行可能根本不存在）
```

**关键点**：
- 编辑 declined 草稿自动翻回 pending
- moyu 完全代理 `/galgame/mine`，不在本地维护"我的提交"列表

### 场景 D：用户搜索时**完全没命中**（要新建）

```
moyu 用户搜不到 → 点击"提交新作"
  ↓
弹出表单（name 4 语言、intro、tag、official、可选 vndb_id、banner 文件）
  ↓
moyu 后端 → wiki POST /galgame/submit (带 Bearer，支持 multipart)
  ↓
wiki 事务:
  - 检查每日配额（默认 5/天，超出返回 20009）
  - vndb_id 不空时检查唯一性
  - INSERT galgame status=3 user_id=uid
  - INSERT aliases / tag / official / engine relations
  - INSERT contributor (uid)
  - INSERT VNDB link (if vndb_id present)
  - INSERT revision 1 (action='created')
  - INSERT message (type='submitted', actor=uid, target=NULL) ← 进 admin 队列
  ↓
返回 status=3 的 galgame { id: 10000, ... }
  ↓
moyu 后端**不做**本地 INSERT 任何东西（决策 2：懒加载，提交时不建 stats）
  ↓
moyu 后端返回响应给前端
  ↓
前端跳转到 /galgame/10000 详情页（或 /me/submissions"等待审核"提示）
```

**审核分支**（异步发生）：

```
admin 在 wiki 后台审核队列看到这条
  ├── 通过 → PUT /admin/galgame/10000/status {status: 0, reason: "可选备注"}
  │   ↓
  │   wiki 事务:
  │     - status 3 → 0
  │     - revision (action='approved')
  │     - message (type='approved', actor=admin_uid, target=submitter_uid)
  │   ↓
  │   moyu cron 每 5-15 分钟拉一次 /galgame/messages/feed
  │   ↓
  │   收到 type='approved'：
  │     - UPDATE user.moemoepoint += 3 (决策 1：approved 才奖励)
  │     - 发本地消息通知"您提交的《X》已通过审核"
  │     - 不需要建 galgame_stats（懒加载）
  │
  └── 拒绝 → PUT /admin/galgame/10000/status {status: 4, reason: "VNDB ID 错"}
      ↓
      wiki 事务:
        - status 3 → 4
        - revision (action='declined')
        - message (type='declined', actor=admin_uid, target=submitter_uid, payload.reason)
      ↓
      moyu cron 收到 type='declined'：
        - 不扣萌萌点（一开始没给）
        - 发本地消息通知"您提交的《X》被拒，原因：..."
      ↓
      用户去"我的提交"页看到，可编辑重新提交（回到场景 C 的 PATCH 分支）
```

---

## 5. moyu 后端 SDK 方法清单

参考 `docs/integration/galgame_wiki/07-submission.md` 的 SDK 草案。建议在 moyu 后端实现一个 `internal/galgame/wiki/client.go`，包含：

### 5.1 用户身份方法（透传 access_token）

| 方法 | 调用 | 用途 |
|---|---|---|
| `Submit(ctx, token, req)` | POST `/galgame/submit` | 用户提交新作 |
| `Claim(ctx, token, gid)` | POST `/galgame/:gid/claim` | 用户认领 VNDB 草稿 |
| `PatchDraft(ctx, token, gid, req)` | PATCH `/galgame/:gid` | 用户编辑自己的 pending/declined |
| `DeleteDraft(ctx, token, gid)` | DELETE `/galgame/:gid` | 用户撤回自己的草稿 |
| `ListMine(ctx, token, req)` | GET `/galgame/mine` | "我的提交"页（带 decline_reason） |
| `SearchWithPending(ctx, token, q)` | GET `/galgame/search?include_pending=true` | 发布向导搜索（含自己的 pending） |
| `SearchPublic(ctx, q)` | GET `/galgame/search` | 公开搜索（不带 Bearer） |
| `Batch(ctx, token?, ids)` | GET `/galgame/batch` | 列表渲染拉 brief；带 Bearer 时含自己 pending |
| `GetDetail(ctx, gid)` | GET `/galgame/:gid` | 详情页元数据 |
| `MyNotifications(ctx, token, sinceID)` | GET `/galgame/messages/mine` | 消息中心 wiki 消息流 |

### 5.1b galgame 编辑面方法（强制，kungal 与 moyu 各实现一份 — 见 §15）

写操作一律透传用户 `access_token`；读操作公开端点可不带（鉴权语义以 wiki 为准）。

| 方法 | 调用 | 用途 |
|---|---|---|
| `UpdateGalgame(ctx, token, gid, req)` | PUT `/galgame/:gid` | 已发布条目直接编辑 |
| `ListRevisions(ctx, gid, page, limit)` | GET `/galgame/:gid/revisions` | 修订历史列表 |
| `GetRevision(ctx, gid, rev)` | GET `/galgame/:gid/revisions/:rev` | 单条修订快照 |
| `GetRevisionDiff(ctx, gid, rev)` | GET `/galgame/:gid/revisions/:rev/diff` | 修订 diff |
| `Revert(ctx, token, gid, req)` | POST `/galgame/:gid/revert` | 回滚到某修订 |
| `SubmitPR(ctx, token, gid, req)` | POST `/galgame/:gid/prs` | 提交编辑请求（PR） |
| `ListPRs(ctx, gid)` | GET `/galgame/:gid/prs` | PR 列表 |
| `GetPR(ctx, gid, id)` | GET `/galgame/:gid/prs/:id` | PR 详情/diff |
| `MergePR(ctx, token, gid, id)` | PUT `/galgame/:gid/prs/:id/merge` | 合并 PR |
| `DeclinePR(ctx, token, gid, id, req)` | PUT `/galgame/:gid/prs/:id/decline` | 拒绝 PR |
| `ListLinks/CreateLink/DeleteLink(ctx, [token,] gid, ...)` | GET/POST/DELETE `/galgame/:gid/links` | 链接增删查 |
| `ListAliases/CreateAlias/DeleteAlias(ctx, [token,] gid, ...)` | GET/POST/DELETE `/galgame/:gid/aliases` | 别名增删查 |
| `ListContributors(ctx, gid)` / `DeleteContributor(ctx, token, gid, id)` | GET / DELETE `/galgame/:gid/contributors[/:id]` | 贡献者查/删 |
| `Tag{List,Search,Get,Create,Update,Delete}` | GET/POST/PUT/DELETE `/tag*` | tag 分类轴全 CRUD |
| `Official{List,Search,Get,Create,Update,Delete}` | GET/POST/PUT/DELETE `/official*` | official 全 CRUD |
| `Engine{List,Get,Create,Update,Delete}` | GET/POST/PUT/DELETE `/engine*` | engine 全 CRUD |
| `Series{List,Search,Get,Create,Modal,Update,Delete}` | GET/POST/PUT/DELETE `/series*` | series 全 CRUD |

### 5.2 服务到服务方法（Basic Auth）

| 方法 | 调用 | 用途 |
|---|---|---|
| `MessageFeed(ctx, sinceID, limit)` | GET `/galgame/messages/feed` | cron 拉新消息（approved/declined/banned/unbanned） |

### 5.3 调用样板

```go
// 用户身份调用
type WikiClient struct {
    baseURL    string  // e.g. https://galgame.kungal.com/api
    httpClient *http.Client
    // basic auth credentials (只给 MessageFeed 用)
    clientID, clientSecret string
}

func (c *WikiClient) Submit(ctx context.Context, token string, req SubmitRequest) (*Galgame, error) {
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/galgame/submit", toJSON(req))
    httpReq.Header.Set("Authorization", "Bearer "+token)
    httpReq.Header.Set("Content-Type", "application/json")
    return c.do(httpReq)
}

// 服务身份调用
func (c *WikiClient) MessageFeed(ctx context.Context, sinceID int64, limit int) ([]Message, bool, error) {
    url := fmt.Sprintf("%s/galgame/messages/feed?since_id=%d&limit=%d", c.baseURL, sinceID, limit)
    httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    creds := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
    httpReq.Header.Set("Authorization", "Basic "+creds)
    // ...
    return items, hasMore, nil
}
```

---

## 6. moyu 数据库 migration

只有一项：**不需要任何 ALTER 给 galgame_stats**（决策 2：不加 snapshot 列）。

需要：

```sql
-- 1. 加一张 cron 游标表（如果还没有的话，给所有 cron 共享）
CREATE TABLE IF NOT EXISTS cron_state (
    name        VARCHAR(64) PRIMARY KEY,
    last_id     BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. wiki 消息已读状态（前端"未读数"用，可选）
CREATE TABLE IF NOT EXISTS wiki_message_read_state (
    user_id              INT PRIMARY KEY,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`galgame_stats` 保持现状（如果还没建，仍按 `08-galgame-service-architecture.md` 的 schema 建，但不加 wiki_status_snapshot 列）。

---

## 7. cron 实现

每 **5–15 分钟**跑一次（不是每日）。

```go
// internal/galgame/cron/wiki_sync.go
func (c *WikiSyncCron) Run(ctx context.Context) error {
    // 1. 读上次游标
    var sinceID int64
    db.Raw("SELECT last_id FROM cron_state WHERE name = 'wiki_msg_sync'").Scan(&sinceID)

    for {
        // 2. 拉增量
        msgs, hasMore, err := c.wiki.MessageFeed(ctx, sinceID, 1000)
        if err != nil {
            return err
        }
        if len(msgs) == 0 {
            break
        }

        // 3. 在一个事务里处理这批
        err = db.Transaction(func(tx *gorm.DB) error {
            for _, m := range msgs {
                if m.Galgame == nil {
                    // 幽灵消息——作品被删了
                    tx.Exec(`DELETE FROM galgame_stats WHERE galgame_id = ?`, m.GalgameID)
                    continue
                }
                switch m.Type {
                case "approved", "unbanned":
                    // 萌萌点 +3 给提交者（仅 approved；unbanned 不奖励）
                    if m.Type == "approved" && m.TargetUserID != nil {
                        tx.Exec(`UPDATE "user" SET moemoepoint = moemoepoint + 3 WHERE id = ?`, *m.TargetUserID)
                    }
                    // 发本地消息通知用户
                    writeLocalNotification(tx, m.TargetUserID, "approved",
                        fmt.Sprintf("您提交的《%s》已通过审核", displayName(m.Galgame)))
                case "declined":
                    reason := payloadString(m.Payload, "reason")
                    writeLocalNotification(tx, m.TargetUserID, "declined",
                        fmt.Sprintf("您提交的《%s》被拒：%s", displayName(m.Galgame), reason))
                case "banned":
                    // 作品被封 → 清掉本地 stats（避免列表里点进去 404）
                    tx.Exec(`DELETE FROM galgame_stats WHERE galgame_id = ?`, m.GalgameID)
                    writeLocalNotification(tx, m.TargetUserID, "banned",
                        fmt.Sprintf("您的作品《%s》已被封禁", displayName(m.Galgame)))
                }
                sinceID = m.ID
            }
            // 4. 更新游标
            return tx.Exec(`
                INSERT INTO cron_state(name, last_id) VALUES ('wiki_msg_sync', ?)
                ON CONFLICT (name) DO UPDATE SET last_id = EXCLUDED.last_id, updated_at = NOW()
            `, sinceID).Error
        })
        if err != nil {
            return err
        }

        if !hasMore {
            break
        }
    }
    return nil
}
```

**关键不变量**：
- 整个批次在**一个事务**里。萌萌点累加、本地通知写入、游标推进必须原子
- 出错时事务回滚 → 游标不前进 → 下次 cron 重新处理这批（消息处理必须**幂等**——萌萌点 idempotent 不容易，下面 §8 讲）

---

## 8. 萌萌点幂等性（important）

cron 可能重跑同一批消息（崩溃恢复、人为重启）。如果 cron 直接 `moemoepoint += 3`，重跑就会重复奖励。

**解法 A（推荐，简单）**：用 `wiki_message_id` 做处理日志

```sql
CREATE TABLE wiki_message_processed (
    message_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

每条消息处理前 `INSERT ... ON CONFLICT DO NOTHING RETURNING message_id`——成功插入才往下走奖励/通知逻辑，重复 message_id 跳过。

```go
result := tx.Exec(`INSERT INTO wiki_message_processed(message_id) VALUES (?) ON CONFLICT DO NOTHING`, m.ID)
if result.RowsAffected == 0 {
    continue  // 已经处理过
}
// safely award moemoepoint, write notification, etc.
```

**解法 B（更轻量但更难做对）**：游标 `last_id` 当成唯一去重器，cron 必须保证"游标推进"和"业务副作用"在同一事务内。前面 §7 的事务代码就是这种思路——但要小心**事务超时**和"批处理一半失败，部分消息已被业务消费但游标没推"的边界情况。

推荐 **A + B 一起用**：A 是绝对的幂等护甲，B 是常态推进。

---

## 9. 用户操作时的本地副作用速查

cron 处理 admin 触发的事件；**用户自己触发**的事件不依赖 cron，moyu 后端在 RPC 返回时同步处理：

| 用户操作 | wiki 端结果 | moyu 后端在响应回来时做的事 |
|---|---|---|
| Submit | 创建 status=3 galgame | **不做事**（galgame_stats 懒加载） |
| Claim | status 2→0 | INSERT galgame_stats(zeros) + moemoepoint += 3 |
| PatchDraft | 改字段 + 4→3 翻转 | **不做事** |
| DeleteDraft | 硬删 galgame + relations | DELETE galgame_stats(galgame_id) — 如果有的话 |
| 首次互动（like/comment） | (与 wiki 无关) | `INSERT ... ON CONFLICT DO UPDATE` galgame_stats |

---

## 10. 前端 UX 流程

### 10.1 发布 galgame 向导（3 步）

```
Step 1: 输入名字搜索
  ├── 命中已发布 → "选择此条目并直接发布"
  ├── 命中 VNDB 草稿 → "认领并发布"
  ├── 命中自己 pending → "查看审核状态"
  └── 都没命中 → "都不是？提交新作"

Step 2: 提交新作表单
  - name 4 语言（建议至少填 1 个）
  - intro 4 语言（可选）
  - banner（image_service 上传，可选；不传 fallback default）
  - tag / official / engine（多选）
  - aliases（逗号分隔）
  - vndb_id（可选）

Step 3: 提交后
  - 成功 → 跳到详情页或"我的提交"页
  - 显示"等待审核中"状态
```

### 10.2 "我的提交"页

```typescript
// pages/me/submissions.vue
const data = await $fetch('/api/wiki/galgame/mine?status=3,4')
// 直接代理 wiki /galgame/mine
// data.items 形如：
// [
//   { id: 10000, status: 3, name_zh_cn: "...", effective_banner_hash: "..." },
//   { id: 9999,  status: 4, decline_reason: "VNDB ID 错", ... }
// ]
```

每条:
- status=3 → "审核中" 灰底
- status=4 → "已拒绝" 红底 + 显示 `decline_reason` + "重新编辑" 按钮（调 PATCH 端点）

### 10.3 消息中心

合并两个源：
```typescript
const [localMsgs, wikiMsgs] = await Promise.all([
  $fetch('/api/message'),                 // moyu 本地（reply/like/PR 等）
  $fetch('/api/wiki/messages/mine')       // moyu 后端透传到 wiki
])
const merged = [...localMsgs, ...wikiMsgs].sort(byCreatedAtDesc)
```

未读数：本地用 `wiki_message_read_state.last_read_message_id` 做基线，count wiki messages 中 `id > last_read_id` 的数量。

### 10.4 详情页

按 wiki id 渲染，**不区分 status**——但要处理 batch 返回为空的情况：
- batch 返回有 → 渲染
- batch 返回空（galgame 不存在 / 被 ban / 不是 owner 的 pending）→ 404 页

---

## 11. 错误处理速查

| HTTP | code | 含义 | 前端建议 |
|---|---|---|---|
| 400 | 20003 | vndb_id 格式错 | 表单校验提示 |
| 400 | 20004 | vndb_id 已存在 | 提示"该 VNDB 作品已有人提交"，引导走搜索 |
| 400 | 20006 | 草稿不可认领 | 提示"该草稿已被他人认领" → 刷新搜索 |
| 400 | 20008 | 草稿状态不对 | 内部错误（不该发生） |
| 403 | 20005 | 无权操作 | "你不是创建者 / 非 admin" |
| 403 | 20007 | 仅提交者可编辑 | "你不是这条草稿的提交者" |
| 429 | 20009 | 配额耗尽 | "您今日投稿已达上限（5 条），明日再来" |

完整列表见 [99-appendix.md](./99-appendix.md)。

---

## 12. 工作量估算

| 阶段 | 内容 | 人天 |
|---|---|---|
| 1. DB migration | `cron_state` + `wiki_message_read_state` 两张表 | 0.5 |
| 2. SDK client | WikiClient 11 个方法 | 2 |
| 3. cron 实现 | MessageFeed 拉取 + 幂等表 + 萌萌点 + 本地通知 | 1 |
| 4. 后端 API 代理 | 5 个用户操作端点 + 1 个 messages/mine 代理 + 1 个 search-for-publish + 1 个 /me/submissions | 1.5 |
| 5. 前端发布向导 | 3 步搜索-选择-提交流程 + multipart banner | 3 |
| 6. 前端"我的提交"页 | 列表 + 编辑/撤回操作 | 2 |
| 7. 前端消息中心融合 | wiki 消息流 + 已读状态 | 1 |
| 8. 联调 + 灰度 | 内部账号试跑、灰度 1% 流量、监控 | 2 |
| **合计** | | **13 人天 / 约 3 周** |

可以分两期上：
- **期一**：场景 A + B（已发布选用 + VNDB 草稿认领）—— 8 人天
- **期二**：场景 D（新建提交 + 审核）+ 我的提交页 + cron —— 5 人天

> ⚠️ **上表仅覆盖投稿流，不含 §15 的编辑面**（PR / 修订历史 / 关系 / 分类轴增删改）。§15 是后追加的强制范围，需**额外**估算并排期——且 **kungal 与 moyu 各算一份**（两站不共享前端，后端代理也各写一份）。新增 `期三`：galgame 编辑面全量代理 + UI（两站各一套），按端点数量另行评估。

---

## 13. 上线 checklist

### 后端
- [ ] WikiClient 完成，所有 11 个方法各写一个单元测试
- [ ] cron 跑通：dev 环境模拟一条 approved 消息 → 验证萌萌点 +3 + 本地通知写入 + 游标推进
- [ ] 幂等性测试：同一条 message_id 两次进 cron，萌萌点只 +3 一次
- [ ] 5 个用户操作端点透传 access_token 正确（不要"自己解 JWT 改 user_id"）
- [ ] 错误码透传：wiki 返回 20009（配额）→ 前端能看到 429 + 友好文案

### 前端
- [ ] 发布向导覆盖 4 种命中分支
- [ ] "我的提交"页正确显示 decline_reason
- [ ] 消息中心合并 wiki + 本地两个源
- [ ] 列表/首页/详情**不带 Bearer** 调 batch（避免泄露自己的 pending 给其他人）
- [ ] 发布向导的搜索框**带 Bearer + include_pending=true**

### 联调
- [ ] 用 curl 模拟 cron 拉 feed，确认 Basic Auth 用现有 client_id/secret 通过
- [ ] 提交 → admin 在 wiki 后台 approve → 等下一个 cron tick → 验证 moyu 用户萌萌点真的 +3
- [ ] 提交 → admin decline + reason → 用户在 /me/submissions 看到 reason → 编辑后重新提交 → admin 看队列里又出现

### 灰度
- [ ] 1% 流量 24h 观察提交成功率、cron 失败率
- [ ] 监控 wiki 端 `/messages/feed` 调用频次（每 5–15 min × 站点数 × N batch）
- [ ] 监控 moyu cron 的 `wiki_message_processed` 表增长（应等于 feed 拉到的 message 数）

---

## 14. 参考文档

| 文档 | 用途 |
|---|---|
| [06-submission-and-review-design.md](../../galgame_wiki/06-submission-and-review-design.md) | wiki 侧主设计：status 状态机、schema、可见性矩阵 |
| [07-submission.md](./07-submission.md) | submit/claim/patch/delete/mine 端点详细说明 |
| [08-messages.md](./08-messages.md) | /messages/mine、/messages/feed、admin queue 端点 |
| [01-galgame.md](./01-galgame.md) | batch/detail 等公共端点；含 multipart banner 上传 |
| [05-search.md](./05-search.md) | 搜索端点 + `include_pending=true` |
| [99-appendix.md](./99-appendix.md) | 错误码、端点总览、Meilisearch 运维 |

---

## 15. kungal / moyu 必须各自完整实现的 galgame 编辑面（强制，全覆盖）

> **方向变更（强制）**：galgame 的编辑类操作——PR 编辑、修订历史操作、关系（links/aliases/contributors）增删、分类轴（tag/official/engine/series）增删改——**不再是「wiki-only，下游不做」**。wiki 仍是数据唯一可信源（SoT）与服务端逻辑实现方，但**面向用户的完整功能必须在 kungal 和 moyu 各自落地一份**：每一站都要做后端代理（透传用户 access_token）+ 前端 UI，**功能与 wiki 端一一对齐，不得删减、不得只做子集**。kungal 一份、moyu 一份，二者覆盖范围相同。

### 15.1 强制清单（每一项 kungal 与 moyu 都要各做一份）

| 域 | wiki 端点 | 鉴权 | 下游必须做 |
|---|---|---|---|
| **已发布条目直接编辑** | `PUT /galgame/:gid` | Bearer（创建者/admin） | 后端代理 + 前端编辑表单 |
| **修订历史 — 列表** | `GET /galgame/:gid/revisions` | 公开 | 后端代理 + 历史列表 UI |
| **修订历史 — 单条** | `GET /galgame/:gid/revisions/:rev` | 公开 | 后端代理 + 快照查看 UI |
| **修订历史 — diff** | `GET /galgame/:gid/revisions/:rev/diff` | 公开 | 后端代理 + diff 视图 |
| **修订历史 — 回滚** | `POST /galgame/:gid/revert` | Bearer | 后端代理 + 回滚操作入口 |
| **PR — 提交编辑请求** | `POST /galgame/:gid/prs` | Bearer | 后端代理 + PR 提交表单 |
| **PR — 列表** | `GET /galgame/:gid/prs` | 公开 | 后端代理 + PR 列表 UI |
| **PR — 详情** | `GET /galgame/:gid/prs/:id` | 公开 | 后端代理 + PR 详情/diff UI |
| **PR — 合并** | `PUT /galgame/:gid/prs/:id/merge` | Bearer | 后端代理 + 合并操作 |
| **PR — 拒绝** | `PUT /galgame/:gid/prs/:id/decline` | Bearer | 后端代理 + 拒绝操作 |
| **关系 — 链接** | `GET/POST/DELETE /galgame/:gid/links` | 写需 Bearer | 后端代理 + 链接编辑 UI |
| **关系 — 别名** | `GET/POST/DELETE /galgame/:gid/aliases` | 写需 Bearer | 后端代理 + 别名编辑 UI |
| **关系 — 贡献者** | `GET /galgame/:gid/contributors`、`DELETE /galgame/:gid/contributors/:id` | 写需 Bearer | 后端代理 + 贡献者管理 UI |
| **分类轴 — tag** | `GET /tag*`、`POST /tag`、`PUT /tag`、`DELETE /tag/:id` | 创建=任意登录用户；改/删=admin/mod | 后端代理 + tag 选择/新建/编辑 UI |
| **分类轴 — official** | `GET /official*`、`POST /official`、`PUT /official`、`DELETE /official/:id` | 同 tag | 后端代理 + UI |
| **分类轴 — engine** | `GET /engine*`、`POST /engine`、`PUT /engine`、`DELETE /engine/:id` | 同 tag | 后端代理 + UI |
| **分类轴 — series** | `GET /series*`、`POST /series`、`POST /series/modal`、`PUT /series/:id`、`DELETE /series/:id` | 创建=任意登录用户；删=admin/mod | 后端代理 + 系列管理 UI |

> `POST /tag` `POST /official` `POST /engine` 为本次新增（详见 [04-taxonomy.md](./04-taxonomy.md)）：任意登录用户可为「VNDB 没有的原创/同人作品」新建尚不存在的 tag/会社/引擎；改/删仍限 admin/moderator（role > 1，普通用户 role=1 一律 403）。下游发布/编辑向导**必须**接入这套「选已有 + 没有就新建」的交互，kungal 与 moyu 各一份。

> 🟢 **ADDITIVE — taxonomy 编辑现已全量审计 + 可 revert（U3）**：
> - **每次** tag / official / engine / series 的 create / update / delete 都会写一条 `taxonomy_revision` 行（多态单表：`entity ∈ {tag,official,engine,series}`、`action ∈ {created,updated,deleted,reverted}`、`snapshot jsonb`、`changed_fields []string`、`user_id` + `user_role` 快照）。
> - **force-delete 时**：除 taxonomy_revision (action='deleted') 外，**还为每个被解除引用的 galgame 写一条 galgame_revision**（`changed_fields=['tag_ids' | 'official_ids' | 'engine_ids' | 'series_id']`），所以 tag/official 等"消失"在 galgame 历史里也有迹可循。
> - **affected_galgame_ids 持久化**：`taxonomy_revision.deleted` 行的 `affected_galgame_ids jsonb` 列保留被影响的 gid 列表，未来"撤销删除"UI 可以读它列出"这 N 部作品之前用过此 tag，要恢复哪些？"。
> - **Series 特殊**：membership（`galgame.series_id`）改动**不**走 series 的 taxonomy_revision，**只**写 galgame_revision（每个受影响 gid 一条，`changed_fields=['series_id']`）。Series 的 taxonomy_revision 仅记 Name/Description。详见 [09-final-upgrade-plan.md §7.5](../../galgame_wiki/09-final-upgrade-plan.md#75-series-revision-完整流程最特殊)。
> - **新端点（每个实体一套）**：
>   - `GET /tag/:id/revisions` / `GET /official/:id/revisions` / `GET /engine/:id/revisions` / `GET /series/:id/revisions` — 分页历史列表（newest first）
>   - `GET /tag/:id/revisions/:rev` 等 — 单一历史 snapshot 详情
>   - `POST /tag/:id/revert {revision: N}` 等 — 回滚到目标版本（admin/moderator；自动 resurrect 被删除的实体行；写 action='reverted' 行；**不**自动恢复 galgame_*_relation 引用——下游 UI 应展示 `affected_galgame_ids` 让 admin 勾选要恢复的关联）
> - **下游 DTO breaking（小，可选）**：`UpdateTagRequest.Alias` / `UpdateOfficialRequest.Alias` / `UpdateEngineRequest.Alias` / `UpdateSeriesRequest.GalgameIDs` 全部改为指针类型 `*[]string` / `*[]int`（presence 语义：不传 = 保持不变；传 `[]` = 清空；传非空 = 权威全量替换）。`UpdateOfficialRequest` 同时新增了 `Original *string` 字段（之前缺失，无法编辑日文原文名）。kungal/moyu 编辑表单需相应调整。
>
> `DELETE /tag|official|engine/:id` 为**安全两段式**（同 `DELETE /admin/image/:hash?force=true` 约定）：默认若仍被任意 galgame 引用则**拒绝**（返回引用数，避免静默把分类从 N 个作品上摘掉）；`?force=true` 才一键「清除全部引用 → 硬删」，返回 `{deleted,forced,purged_relations[,purged_aliases]}` 审计摘要。下游若做分类管理 UI，删除按钮需走「先尝试普通 DELETE → 命中拒绝则二次确认后带 `?force=true`」两步交互。
>
> 🔴 **编辑某个 galgame 的多值字段（`tag_ids/official_ids/engine_ids/aliases/links`）= 经 `PUT /galgame/:gid`，presence 全量替换语义（务必看懂，否则编辑不生效或误清空）**：
> - 这些字段与标量字段一样进 revision/快照/PR-diff（集合语义，顺序无关），一次编辑 = **一条原子 revision**。
> - **不传该字段** = 该集合**保持不变**（只改名字时不会动 tag/别名）。
> - **传数组（含空 `[]`）** = **权威全量集合**，服务端"清空旧的 → 按此重建"；`[]` = 清空全部。
> - 因此 kungal/moyu 的 galgame 编辑表单**必须回传该 galgame 当前的全量集合**（在原集合上增删后整体回传），**绝不能只传"新增/删除的那几个"**——会被当成"替换成只剩这几个"。这是之前"kungal 改 tag/engine/official 不生效"的根因（旧实现整段忽略；现已按 snapshot overlay 根治，详见 [docs/galgame_wiki/01-revision-system-design.md §1.5/§6.1/§6.2](../../galgame_wiki/01-revision-system-design.md)）。
> - `aliases`/`links` 现已是本端点一等字段（推荐整表单一次性提交，单条原子 revision）；`/galgame/:gid/aliases|links` 增删端点保留为便捷糖。`bid`/Bangumi ID 为保留字段，sync 托管，暂不可编辑。
> - 不变量：create/submit/update/patch/merge/revert **全部走同一个 ApplySnapshot 写入路径**，`Snapshot` 每个可编辑字段都能被编辑 API 改到（`bid` 是唯一保留例外），有单测护栏防回归。

> 🔴 **BREAKING — `banner_image_hash` 字段 + 列彻底移除（PR5 一刀切）**：
> - **旧**：响应 / 请求体里的 `banner_image_hash` （image_service hash）；migration period 期间 cover 表与该列并存
> - **新**：彻底退役，**galgame_cover (sort_order=0) 是唯一的 banner image_service 引用**
> - **GET 响应**：`banner_image_hash` 从所有响应中消失（`/galgame/:gid` / 列表 / `/galgame/mine` / message brief 等）。改读派生字段 `effective_banner_hash`（PR2 已加，现在它是唯一可读源）
> - **PUT / POST `/galgame` / `/galgame/submit` / `/galgame/:gid/pr` JSON body**：移除 `banner_image_hash` 入参。改通过 `covers` 数组里的 `{image_hash, sort_order:0, ...}` 表达"钉住的封面"
> - **multipart `file` 上传（创建向导）**：行为**不变**——上传文件后仍自动成为封面。后端实现改为"将上传的 hash 推到 `covers[sort_order=0]`，把已有的 sort_order=0 cover 降到 sort_order=1"（partial-unique 约束自动保证唯一）；下游不需要变。
> - **历史 revision / PR snapshot**：wiki 侧的 `cmd/migrate-drop-banner-image-hash` 已一次性把 `banner_image_hash` jsonb 字段 flatten 进 `covers[]`（若与已有 sort_order=0 cover 不冲突则注入；冲突则保留人工选择）+ 删除该字段。下游**不需要**做任何快照迁移。
> - **refping**：wiki 自己的 daily ping job 已剔除 banner_image_hash 数据源，改从 covers/screenshots 收集；下游 zero work。
> - **kungal / moyu 同期发版**：测试环境一刀切，无 dual-emit 兼容期；SDK 类型、表单 schema、列表卡片要在同一波 PR 里改完。`resolveBannerUrl(g, ...)` 类前端 helper 现在只看 `effective_banner_hash → banner` 两级 fallback。

> 🟢 **ADDITIVE — 新增 `covers[]` / `screenshots[]` 关联字段 + 派生 `effective_banner_hash`**：
> - **新增 `covers[]`**：作品的封面候选集，按 `image_hash` 引用 image_service（无跨服务 FK；写入前下游须先把图片上传到 image_service 拿 hash）。每条 `{image_hash, sort_order, sexual, violence, source, source_key}`。**`sort_order=0` = "钉住的封面"，DB 强制约束每作品最多一张**（partial unique index）。管理员"换封面"= 在同一编辑请求里把旧 cover 的 `sort_order` 改为 1 + 新 cover 的 `sort_order` 改为 0（**绝不**在两次请求里分别做，否则中间态会被 DB 拒绝）。
> - **新增 `screenshots[]`**：作品的画廊 / CG / 截图集，与 covers **同构但独立**：同一张图可在 A 作当 cover、在 B 作当 screenshot；同一作里两张表互不干涉。每条 `{image_hash, sort_order, caption, sexual, violence, source, source_key}`。
> - **派生 `effective_banner_hash`**：响应里只读字段。规则 = covers 中 `sort_order=0` 那张的 `image_hash`，无则 `null`（PR5 起 `banner_image_hash` 已彻底退役，不再作为 fallback；展示链则继续 fallback 到 `banner` 老 URL，详见 PR5 BREAKING 段）。**前端展示封面建议读这个**——它在票选这些扩展演化中都保持稳定。
> - **presence 语义同 `tag_ids` 一样**：`PUT /galgame/:gid` 的 `covers` / `screenshots` 字段 **不传** = 该集合保持不变；**传 `[]` 或非空数组** = 权威全量替换。和 tag_ids 同款的"不全量水合就误清空"陷阱——编辑表单**必须回传当前全量集合**。Create / Submit / PR 同款字段，非指针（首次提交时给空数组即可）。
> - **`sexual` / `violence` 的当前定位**：schema 字段已落库（0..3 整型，0 = 未评定），但 v1 应用层**不基于这两个字段做 NSFW 门控**——展示门控按 `galgame.content_limit` 走 wiki 端的 `content_limit` 查询参数协议（见下方 §NSFW content_limit 协议）。下游 SDK 可以读 `sexual` / `violence` 但不强制消费。
> - **image_service 与 wiki 之间的契约**：图片字节、宽高、文件级安全审核全在 image_service；wiki 只持有 `image_hash` 引用 + 关系属性（顺序 / caption / 本作品语境分级 / 导入来源）。下游编辑画廊 UI 调 wiki 时只需传 hash，不需要把图片 metadata 镜像过来。
> - **TTL/refping**：wiki 侧（`internal/jobs.GalgameImageRefping`）每天 ping「当前 banner + 当前 cover + 当前 screenshot + 所有 revision/PR snapshot 中曾出现过的 hash」给 image_service 保活——保证 revert 到任何历史版本都不会出死图。下游**不需要**做任何 ping。
> - **旧 `banner_image_hash` 字段的状态**：PR5 已完成（U2.c）—— 字段从响应/请求/快照/列定义中彻底移除；详见上方 PR5 BREAKING 段。

> 🔴 **BREAKING — `released` 字符串字段已被替换为 `release_date` + `release_date_tba` 两个字段（无双发兼容期）**：
> - **旧**：响应 / 请求体里的 `released string`（"unknown" / "tba" / "YYYY[-MM[-DD]]"哨兵字符串），不可排序不可筛选不可结构化。
> - **新**：
>   - `release_date string|null`（"YYYY-MM-DD" 或 `null`；`null` = 未知）
>   - `release_date_tba bool`（`true` = 已宣布但日期未定）
>   - 两者**独立**：可同时给值（"预计 2026 年某月"= `release_date="2026-06-01"` + `release_date_tba=true`）。
> - **GET 响应**：`released` 已从 `/galgame/:gid` / 列表 / `/galgame/mine` / 搜索结果中消失；改读两个新字段。
> - **PUT / POST `/galgame` / `/galgame/submit` / PR**：旧 `released` 入参不再被识别；改传 `release_date` + `release_date_tba`。Update 端点对两者各自走 presence 语义（`release_date` 用 `*string`，传 `null` 或省略 = 保持不变，传 `""` = 清空为 unknown，传 `"YYYY-MM-DD"` = 设置；`release_date_tba` 用 `*bool`，省略 = 保持不变）。
> - **校验**：`release_date` 非空时必须严格匹配 `YYYY-MM-DD`（`validate:"datetime=2006-01-02"`）；"YYYY" 或 "YYYY-MM" 已不再接受（前端如有那种 UI，先补到日级再提交）。
> - **历史 revision/PR 快照**：wiki 侧已用 migration cmd 一次性改写所有 `galgame_revision.snapshot` / `galgame_pr.snapshot` 的 jsonb，把 `released` 字段就地拆为 `release_date` + `release_date_tba`，所以 revert 到任何老版本都能正确回放新结构。下游**不需要**做任何快照迁移。
> - **kungal / moyu 同期发版**：wiki 在测试环境一刀切，无 dual-emit 兼容期；旧字段在响应里**直接消失**。下游 SDK 类型、表单 schema、列表卡片、详情页要在同一波 PR 里改完。

### 15.2 落地要求

- **后端**：每个端点都要在 kungal 与 moyu 各实现一个代理方法（用户态调用一律透传用户 `access_token`，**严禁**下游自行解 JWT 改 `user_id`）。SDK 方法清单见 §5，已扩充编辑面。
- **前端**：每个操作都要有对应 UI，两站功能对齐；不允许「kungal 做了 PR、moyu 不做」这类不对称。
- **一致性**：wiki 后续新增/变更任一编辑端点，kungal 与 moyu **都要**同步补齐，保持三处行为一致（wiki 服务端 + kungal 代理/UI + moyu 代理/UI）。
- **鉴权语义**：以 wiki 端为准（创建者/admin、admin/moderator、任意登录用户），下游不得放宽或收紧。

---

## 16. 不在范围内

显式说明，避免重复确认：

- ❌ **VNDB 同步**：wiki 自己每日跑 `sync-vndb` cron 维护 status=2 草稿库存。下游不参与
- ❌ **同步用户提交的"评分/评论/资源"**：这些是 kungal/moyu 本地数据，跟 wiki 完全无关
- ❌ **kungal/moyu 之间任何直接通信**：两站不互相调用，都通过 wiki + OAuth 间接联动
- ❌ **wiki 审核队列（admin moderation）UI**：投稿审核队列（approve/decline/ban，`apps/wiki/app/pages/review/`）是 admin 专属、wiki 自有页面，下游不做。**注意**：这指的是「审核队列」，不含 §15 列出的 PR/修订/关系/分类**编辑面**——后者下游必须各做一份
- ❌ **跨站合并重复提交**：admin 在 wiki 后台手动 ban 或 decline，不做自动 dedupe

> ⚠️ 历史版本曾把「PR 流程 / 修订 / 关系 / 分类编辑」列为下游不在范围内——**该结论已作废**，现按 §15 强制要求 kungal 与 moyu 各自完整实现。

---

## 16. NSFW content_limit 协议

> **wiki 是 NSFW 门控的唯一可信源**。下游展示要 sfw / nsfw / 全部，唯一正确做法是把决策反映为 `content_limit` 查询参数发给 wiki，由 wiki SQL/Meilisearch 层过滤。**不要**在下游做客户端 filtering（wiki 端不过滤的 endpoint 即使你不展示，数据已经离开 wiki 边界）。

### 16.1 参数语义

所有接受 `content_limit` 的端点统一识别 4 种值：

| 取值 | 含义 | 等效 SQL |
|------|------|----------|
| `sfw` | 仅 SFW | `WHERE content_limit = 'sfw'` |
| `nsfw` | 仅 NSFW | `WHERE content_limit = 'nsfw'` |
| `all` | 显式包含两者 | 无 WHERE 子句 |
| 省略 / 空 / 未识别值 | 由 endpoint 自身的默认值决定 | 见 §16.2 |
| `SFW` / `Nsfw` / 大写等 | **不识别**，按"未识别值"处理 → 落到端点默认 | — |

未识别值落到端点默认（不会"穿透"成无过滤）是 safe-by-default 保证 —— 调用方拼错参数不会意外泄露 NSFW。

### 16.2 各端点默认值矩阵

| 端点 | 缺省值 | 设计动因 |
|------|--------|----------|
| `GET /galgame` (主列表) | **sfw** | 浏览语义，safe-by-default |
| `GET /galgame/search` | **sfw** | 同上 |
| `GET /tag/:name`（带 galgames） | **sfw** | 同上 |
| `GET /official/:name`（带 galgames） | **sfw** | 同上 |
| `GET /engine/:name`（带 galgames） | **sfw** | 同上 |
| `GET /tag/multi` | **sfw** | 同上 |
| `GET /series` (列表) | **sfw** | 系列卡片里预载前 5 个 galgame |
| `GET /series/:id` (详情) | **sfw** | 系列详情页里展开所有关联 galgame |
| **`GET /galgame/batch`** | **不过滤** | 调用方已知 IDs（patch.galgame_id、收藏列表、引用回链等）；默认过滤会"静默丢失"调用方显式列出的条目。需要过滤显式传 `?content_limit=sfw` |
| **`GET /galgame/:gid`** (单条详情) | **不过滤** | 与 batch 同款反对称：直接 URL 访问是有意为之；调用方想让详情访问也按 list 口径过滤就显式传同样的 `content_limit`，不匹配时返回 404 |
| `GET /galgame/mine` (我的提交) | 不应用此过滤 | 用户看自己提交的内容 |
| `GET /galgame/user/:id/stats` | 不应用此过滤 | 仅返回聚合 counts，不含 per-entry 数据；不构成内容泄漏 |

> **batch 默认不过滤是有意的反对称**：browse 默认 safe-by-default、batch 默认 explicit-only。原因是 batch 的调用方已经持有 ID（patch / 收藏 / 引用），过滤会让"我请求 5 个，wiki 只回 3 个"变成调用方的隐性问题。

### 16.3 调用示例

```http
GET /galgame                          # sfw 列表（默认）
GET /galgame?content_limit=all        # 含 NSFW 列表
GET /galgame?content_limit=nsfw       # 仅 NSFW

GET /galgame/batch?ids=1,2,3                       # 全部 3 条都返回，不论 sfw/nsfw
GET /galgame/batch?ids=1,2,3&content_limit=sfw     # 仅返回 sfw 的子集

GET /galgame/search?q=fate                          # 等价于 ?content_limit=sfw
GET /galgame/search?q=fate&content_limit=all        # 含 NSFW 全文搜索

GET /tag/魔法?content_limit=all                     # 该 tag 下所有 galgame，含 NSFW
```

### 16.4 与 `age_limit` 的关系（important）

`galgame.age_limit` (`all` / `r18`) 在 v1 **仅作为响应里的元数据**，wiki 端**不依据它做服务端过滤**。门控完全走 `content_limit`。

业务语义：`content_limit` 表达"内容是否涉性 / 暴力"，`age_limit` 表达"作品发行时的年龄分级"。理论上是两个维度，v1 退化成单维度。后续如要给 `age_limit` 加过滤逻辑，会拆一个独立 query 参数 + 文档化，不会突然让 `age_limit` 跟着 `content_limit` 一起被解释。

### 16.5 写入的 content_limit（不在此协议内）

`content_limit` 在 **写入 DTO**（POST/PUT/Submit/PR 的 body）中是该 galgame 的**属性**：枚举 `sfw` / `nsfw`，默认 `sfw`。和这里讨论的查询参数 `content_limit`（控制过滤行为）是**两件事**——同名只是巧合，前者是 row 字段，后者是 query filter。

### 16.6 与 `include_pending` 的交互

`GET /galgame/search?include_pending=true` 在主结果之外再跑一次窄查询返回当前用户的 pending（status=3/4）。**这次查询继承同样的 `content_limit` 过滤**——传 `content_limit=sfw` 时，pending 也只列 SFW 的 pending（即使是用户自己的 NSFW 草稿）。一致语义优先于"反正是自己的内容"。要拿全部 pending 显式传 `content_limit=all`。

### 16.6 下游接入要点

1. 用户在下游 UI 切换"显示 NSFW"开关 → 下游把该偏好转成查询参数 → 调 wiki 时拼到 URL 上。**不要**：客户端二次过滤、自定义 header、JSON-as-header。
2. 下游可以为不同页面用不同策略（首页主流 sfw、用户偏好页 all、收藏页 batch 不过滤等）。
3. 用户登出 / 未登录 / cookie 丢失等 fallback 场景下，下游应当不传 `content_limit`，让 wiki 按端点默认（即 sfw）兜底。

---

## 17. 发售日期 (release_date) 筛选协议

galgame 的发售日期字段（`galgame.release_date`，PG `date` 类型）覆盖率 **91%**（截至 2026-05，55 525 / 61 002 条有日期）；其中 status=2 VNDB 草稿 95%，status=0 用户已发布 70%。数据本身**月份精度真实**（只有 2.85% 落在每年 1月1日，不是"年精度伪装成月日"）。前端可以放心做按年份 / 按月份的筛选 chip。

### 17.1 协议

接受 `released_from` + `released_to` 两个查询参数，**字符串**类型，三种格式：

| 输入 | 含义 | 边界（inclusive） |
|------|------|------|
| `YYYY`（如 `2024`） | 整年 | from = Jan 1 00:00:00 UTC / to = Dec 31 23:59:59 UTC |
| `YYYY-MM`（如 `2024-03`） | 整月 | from = 月 1 日 00:00:00 UTC / to = 月末 23:59:59 UTC（自动处理 28/29/30/31 天）|
| 空 / 省略 | 不过滤该端 | 该端不施加 WHERE |

非法输入（`24` / `2024-3` 缺前导零 / `2024-13` 月份越界 / `garbage`）→ **400 ErrValidationFailed**，不会被静默忽略。

> `released_from` / `released_to` 表达的是**连续区间**。要在区间内只选**不连续的若干月份**（如"历年三月和七月发售"），叠加 `released_months` 参数，见 §17.10。

### 17.2 应用端点

| 端点 | 实现 | 索引 |
|------|------|------|
| `GET /galgame` | SQL `WHERE release_date >= ? AND release_date <= ?`（PG `date` 列）；`released_months` 追加 `EXTRACT(MONTH FROM release_date)::int IN (...)` | `galgame.release_date` btree index（model gorm tag 声明）|
| `GET /galgame/search` | Meilisearch `released_ts >= X AND released_ts <= Y`（Unix 秒）；`released_months` 追加 `released_month IN [...]` | `released_ts` / `released_month` filterable + indexer 已写入 |

年份区间两套实现都通过 `pkg/utils.ParseReleaseLowerBound` / `ParseReleaseUpperBound` 解析；月份集合通过 `pkg/utils.ParseMonthSet` 解析（见 §17.10）。

### 17.3 与 sort 的联动

`GET /galgame` 的 `sort_field` 新增 `release_date` 选项 — 按发售日期升/降序排。常见组合：
- 浏览「2024 年所有游戏，按发售时间倒序」：
  ```
  GET /galgame?released_from=2024&released_to=2024&sort_field=release_date&sort_order=desc
  ```
- 浏览「2024 年 3 月发售的」：
  ```
  GET /galgame?released_from=2024-03&released_to=2024-03
  ```

`GET /galgame/search` 的 `sort` 维持 `released_desc` / `released_asc` 选项不变。

### 17.4 NULL 处理

设了任一端边界 → `release_date IS NULL` 的行**自动排除**（PG 的 `>=` / `<=` 对 NULL 求 UNKNOWN，自动 drop）。这符合用户期望：「我筛 2024 年的」就是只看知道发售日期的 2024 年游戏。

排序时（`sort_field=release_date`）PG 默认 NULLS LAST on ASC、NULLS FIRST on DESC — 可接受，NULL 行少（status=0 才 30% 是 NULL，且大多是用户提交时漏填）。

### 17.5 向后兼容

`/galgame/search` 历史上 `released_from/to` 是 int（年）。新协议升级成 string 仍接受 `2024` 这种纯数字字符串（4 字符 = 年）→ **kungal / moyu 这种历史调用方零改动**。下游升级到月份精度只需把 `2024` 改成 `2024-03` 即可。

### 17.6 时区

`release_date` 是 PG `date` 类型，无时区。helper 解析 `YYYY-MM` 时按 UTC 计算边界 ts。日期只到日级精度 — 跨时区显示语义清晰（"2024 年 3 月发售"对全球用户都是同一组游戏，不会因时区切换跑出 1 天差异）。

### 17.7 响应里 release_date 的格式：`YYYY-MM-DD`（不是 RFC3339）

galgame 实体响应里 `release_date` 字段一律是 **`"2019-08-16"`** 这种纯日期串（或 `null` = 未知），**不带** `T00:00:00Z`。GET /galgame/:gid、GET /galgame 列表、GET /galgame/search、revision/PR snapshot 全部一致。

> 历史注记：早期 wiki 把 `release_date`（PG `date` 列）用 Go `*time.Time` 裸序列化，REST 详情/列表端点会吐 RFC3339 `"2019-08-16T00:00:00Z"`，与文档承诺的 `YYYY-MM-DD` 不符，且给 date-only 值塞了幻影 `T00:00:00Z`（消费方做时区转换会 off-by-one-day）。已用自定义 `Date` 类型修复 —— 现在三条序列化路径（REST live / snapshot / search index）统一输出 `YYYY-MM-DD`。
>
> **下游 parser 建议**：用 `YYYY-MM-DD` layout 解析即可。想对老数据/缓存鲁棒可同时容错 RFC3339，但 canonical wiki 现在保证纯日期。

### 17.8 batch 端点不含 release_date（重要）

`GET /galgame/batch` 是轻量 DTO，**不返回 `release_date`**（也不返回 intro / tag / cover / screenshot 等）。要 `release_date` 用 `GET /galgame/:gid`、`GET /galgame`、`GET /galgame/search`。字段白名单见 [01-galgame.md GET /galgame/batch](./01-galgame.md#get-galgamebatch)。

想"本地镜像 release_date 再按发售日期排序/筛选"的下游（典型 moyu patch 表）务必从 `/galgame/:gid` 或列表/搜索端点取，**不要**指望 batch。

### 17.9 所有 timestamp 字段：统一 UTC RFC3339（`...Z`）

galgame 实体 + DTO 里的所有**时间戳**字段（`created` / `updated` / `resource_update_time` / `created_at` / `completed_time` 等）一律序列化为 **UTC RFC3339 + 字面 Z**，如 `"2026-05-10T16:24:28Z"`。**不会**出现带本地偏移的形式（`...+08:00` / `...-07:00`）。

### 17.10 不连续月份筛选：`released_months`（2026-05-28 新增）

`released_from` / `released_to` 只能表达**连续区间**。要在一段连续年份内只挑**不连续的若干月份**（典型："历年三月发售的"、"每年春季 3/4/5 月"、"2024 年的 3 月和 7 月"），用 `released_months` 参数。

**协议**：逗号分隔的月份号（1–12），是叠加在年份区间上的**正交 AND 过滤**。

| 参数 | 示例 | 含义 |
|------|------|------|
| `released_months` | `3,7,12` | 只保留发售月份 ∈ {3, 7, 12} 的游戏 |

- 解析：去空格、去重、升序（`12, 3, 3` → `{3,12}`）；空 / 省略 = 不施加月份过滤。
- 非法输入（`0` / `13` / `-1` / `abc` / `3,15`）→ **400 ErrValidationFailed**。
- **两端点都支持**：`GET /galgame`（SQL）与 `GET /galgame/search`（Meilisearch）。
- 与年份区间是 **AND** 关系，自由组合：

```
# 历年（不限年）每年三月发售的
GET /galgame?released_months=3

# 2020–2024 这五年里，3 月和 7 月发售的
GET /galgame?released_from=2020&released_to=2024&released_months=3,7

# 2024 一整年里，只看 3 月和 7 月（等价于年区间 = 单年 + 月集合）
GET /galgame?released_from=2024&released_to=2024&released_months=3,7

# 搜索端同款
GET /galgame/search?q=魔法&released_from=2020&released_to=2024&released_months=3,7
```

**能表达 / 不能表达**：覆盖"连续年份 + 任意月份子集"。**唯一表达不了**的是"逐年不同的月份集合"（如 2020 要 3 月、2021 要 7 月）—— 这种极小众需求不在本协议范围。

**实现 / 性能**：
- `GET /galgame`：SQL `EXTRACT(MONTH FROM release_date)::int IN (...)`。非 sargable，但只对年份区间 btree 扫描**已缩小**的候选集做 recheck。实测 EXPLAIN：`Bitmap Index Scan on idx_galgame_release_date`（走年份区间索引）+ `Filter: EXTRACT(month ...) = ANY(...)`（廉价 recheck），无回归。
- `GET /galgame/search`：新增 filterable 字段 `released_month`（1–12，indexer 与 `released_year` 一并写入），过滤串 `released_month IN [3, 7]`。**升级搜索端需跑一次 `go run ./cmd/reindex-search --index=galgames` 重建索引**以回填 `released_month`（SQL 端无需迁移）。
- **NULL**：`release_date IS NULL` 的行在月份过滤下也自动 drop（`EXTRACT(NULL)` → NULL → 不 IN），与 §17.4 年份区间的 NULL 语义一致。

> 历史注记：早期这些字段用 Go 裸 `time.Time` 序列化。DB 驱动按**进程本地时区**返回 time.Time（实测 -0700），导致:
> - raw-model 端点（GET /galgame/:gid 等）吐出带本地 offset 的串（`...-07:00`），形状随部署机器时区漂移
> - DTO 端点曾用硬编码 `Format("...Z")` 但 time.Time 非 UTC → **标 Z 却是本地钟面，差一个偏移量**
>
> 已用自定义 `Timestamp` 类型（`MarshalJSON` 强制 `.UTC()`）统一修复。现在无论服务器跑在什么时区，所有 timestamp 都是真 UTC 的 `...Z`。区分：
> - **timestamp**（有具体时刻）→ `2026-05-10T16:24:28Z`（UTC RFC3339）
> - **date-only**（`release_date`，只到天）→ `2019-08-16`（无时间无时区，见 §17.7）
>
> **下游 parser 建议**：标准 RFC3339 解析即可（`Z` 后缀,UTC）。不要假设带本地 offset。

---

## 联系人

- wiki API 变更：通知此文档对应 owner
- 接入过程问题：参考 [docs/integration/galgame_wiki/](.) 完整目录
- 集成 bug：直接提 issue 到 wiki repo
