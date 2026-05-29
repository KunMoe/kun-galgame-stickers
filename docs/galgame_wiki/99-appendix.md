> [📖 文档索引](./README.md) · 上一节：[06 — 管理统计](./06-admin.md)

## 错误码

### Galgame (20xxx)

| Code | 消息 | 说明 |
|------|------|------|
| 20001 | Galgame 不存在 | ID 不存在或已被封禁 |
| 20002 | Galgame 已存在 | — |
| 20003 | 无效的 VNDB ID | 格式不匹配 `v\d+` |
| 20004 | 该 VNDB ID 的 Galgame 已存在 | VNDB ID 重复 |
| 20005 | 无权操作此 Galgame | 非创建者且非 admin |
| 20006 | 草稿不可认领 | claim 时目标 status ≠ 2 |
| 20007 | 仅提交者可编辑 | PATCH/DELETE 时不是 user_id 本人 |
| 20008 | 草稿仅可在待审/已拒状态编辑 | PATCH 时 status ∉ {3,4} |
| 20009 | 今日投稿配额已用尽 | submit 超出每日 5 条 |

### 通用

| Code | 消息 |
|------|------|
| 1 | 请求格式错误 |
| 2 | 无效的 ID |
| 4 | 资源不存在 |
| 5 | 访问被拒绝 |
| 7 | 参数验证失败 |
| 10 | 操作失败 |
| 10001 | 未授权 |
| 10002 | 无效的令牌 |
| 10003 | 令牌已过期 |

---

## 端点总览

| 模块 | 方法 | 路径 | 认证 | 数量 |
|------|------|------|------|------|
| **Galgame** | GET | `/galgame`, `/galgame/search`, `/galgame/batch`, `/galgame/check`, `/galgame/user/:id/stats`, `/galgame/:gid` | 公开 | 6 |
| | POST/PUT | `/galgame`, `/galgame/:gid` | Bearer | 2 |
| **Revision** | GET | `/galgame/:gid/revisions`, `.../:rev`, `.../:rev/diff` | 公开 | 3 |
| | POST | `/galgame/:gid/revert` | Bearer | 1 |
| **PR** | GET | `/galgame/:gid/prs`, `.../:id` | 公开 | 2 |
| | POST/PUT | `/galgame/:gid/prs`, `.../merge`, `.../decline` | Bearer | 3 |
| **Link** | GET/POST/DELETE | `/galgame/:gid/links` | 读公开，写Bearer | 3 |
| **Alias** | GET/POST/DELETE | `/galgame/:gid/aliases` | 读公开，写Bearer | 3 |
| **Contributor** | GET/DELETE | `/galgame/:gid/contributors` | 读公开，删Bearer | 2 |
| **Tag** | GET | `/tag`, `/tag/search` (MS), `/tag/multi`, `/tag/:name` | 公开 | 4 |
| | PUT | `/tag` | admin/mod | 1 |
| **Official** | GET | `/official`, `/official/search` (MS), `/official/:name` | 公开 | 3 |
| | PUT | `/official` | admin/mod | 1 |
| **Engine** | GET | `/engine`, `/engine/:name` | 公开 | 2 |
| | PUT | `/engine` | admin/mod | 1 |
| **Series** | GET | `/series`, `/series/search`, `/series/:id` | 公开 | 3 |
| | POST/PUT/DELETE | `/series`, `/series/modal`, `/series/:id` | Bearer/admin | 4 |
| **Admin** | GET | `/admin/stats`, `/admin/galgame`, `/admin/galgame/:gid`, `/admin/galgame/messages` | Bearer + admin | 4 |
| | PUT | `/admin/galgame/:gid/status` | Bearer + admin | 1 |
| **Submission** | POST | `/galgame/submit`, `/galgame/:gid/claim` | Bearer | 2 |
| | PATCH/DELETE | `/galgame/:gid` (草稿) | Bearer | 2 |
| | GET | `/galgame/mine` | Bearer | 1 |
| **Messages** | GET | `/galgame/messages/mine` | Bearer | 1 |
| | GET | `/galgame/messages/feed` | Basic Auth | 1 |
| | | | **总计** | **65** |

> **标注 (MS) = Meilisearch 驱动**；其余 search 端点（如 `/series/search`）仍基于 Postgres。

---

## 附录：Meilisearch 运维

### 部署

- 生产环境运行一个 Meilisearch 实例，通过 `KUN_MEILISEARCH_HOST` 注入到 wiki 服务
- Index 前缀：生产无前缀（`galgames` / `galgame_tags` / `galgame_officials`）；开发/测试可设 `KUN_MEILISEARCH_INDEX_PREFIX=dev_` 避免污染

### 三层同步机制

| 时机 | 机制 | 说明 |
|------|------|------|
| **服务启动** | `EnsureIndexes` | 自动创建 3 个 index + PATCH settings（searchable / filterable / sortable / ranking rules / typo tolerance / faceting）；**不推文档** |
| **服务启动** | `WarnIfIndexesEmpty` | EnsureIndexes 之后检查每个 index 的 `numberOfDocuments`，0 则 `slog.Warn` 提示运维跑 `cmd/reindex-search` |
| **CRUD 触发** | write-through Hook | 每次 Galgame / Tag / Official 创建或编辑成功后启 goroutine upsert 到 MS；fire-and-forget，失败只 log |
| **手动批量** | `cmd/reindex-search` | 全量从 Postgres 重建索引；唯一的 backfill 路径 |

### 🔴 fresh Meilisearch 实例（无 `data.ms`）必须做的一步

**默认情况下，wiki 服务启动时不会自动从 Postgres 推文档到 MS。** 这是 by design 的"撤销=重跑"哲学，避免每次启动都跑 1 分钟的全量重建。

如果你刚:
- 第一次部署
- 重新挂载新 Meilisearch volume
- `docker compose down -v` 清掉 data.ms
- 看到启动日志 `slog.Warn` 抱怨 index 为空

→ **必须手动跑：**

```bash
go run ./cmd/reindex-search                       # 三个索引全跑
go run ./cmd/reindex-search --index=galgames      # 只重建一个
go run ./cmd/reindex-search --batch=500           # 调批大小（默认 1000）
```

预估时间（典型规模）：
| Index | 文档数 | 时长 |
|---|---|---|
| galgames | ~60k | 60–120s |
| galgame_tags | ~3k | 5s |
| galgame_officials | ~22k | 20s |

跑完 `GET /galgame/search?q=fate` 立即返回结果。

### 其他需要重跑 reindex 的场景

- `sync-vndb` / `migrate-galgame-data` / 任何批量 ETL 脚本之后（这些脚本绕过 write-through）
- 改了 `settings.go` 里的 `searchableAttributes` / `filterableAttributes` 顺序或集合
- 监控发现 MS 跟 PG 漂移（write-through 失败累积）—— 建议每周低峰期 cron 兜底跑一次

### 索引 settings 变更

改 `internal/platform/galgame/search/settings.go` 重启服务即生效（`EnsureIndexes` 始终 PATCH）。Meilisearch 内部 diff 只对真正变化的部分重新索引，无须人工干预。但若是改了 doc 内容结构（比如新加 `effective_banner_hash` 之类的派生字段），则要重跑一次 `reindex-search` 让旧文档也带上新字段。
