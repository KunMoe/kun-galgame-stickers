# Galgame Wiki API 参考

基础路径：`/api`

| 环境 | Base URL |
|------|----------|
| 开发 | `http://127.0.0.1:9280/api` |
| 生产 | `https://galgame.kungal.com/api` |

## 文档索引

| # | 文件 | 内容 |
|---|------|------|
| 00 | [handbook-for-downstream.md](./00-handbook-for-downstream.md) | **kungal / moyu 接入手册** — 端到端流程 + 决策回顾 + 工作量估算 + checklist |
| 01 | [galgame.md](./01-galgame.md) | Galgame 核心 CRUD（含 banner 文件上传） |
| 02 | [revisions-and-prs.md](./02-revisions-and-prs.md) | 版本历史 + PR 编辑请求流程 |
| 03 | [relations.md](./03-relations.md) | 链接 / 别名 / 贡献者子资源 |
| 04 | [taxonomy.md](./04-taxonomy.md) | Tag / Official / Engine / Series 分类轴 CRUD |
| 05 | [search.md](./05-search.md) | Meilisearch 驱动的搜索接口 |
| 06 | [admin.md](./06-admin.md) | 管理统计与状态变更 |
| 07 | [submission.md](./07-submission.md) | 用户投稿与审核流程（submit / claim / patch-draft） |
| 08 | [messages.md](./08-messages.md) | 消息系统（投稿事件流，wiki 单一来源） |
| 99 | [appendix.md](./99-appendix.md) | 错误码、端点总览、Meilisearch 运维 |

> 🔴 **强制范围变更**：galgame 的编辑面（PR、修订历史、关系、分类轴增删改）**不再是 wiki-only**——kungal 与 moyu **各自必须完整实现一份**（后端代理 + 前端 UI，与 wiki 对齐）。权威清单见 [00-handbook §15](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。

> 📦 **2026-Q2 升级摘要（PR1–PR5，已上线）**：
> - **PR1**：`released` 字符串拆为 `release_date` (`YYYY-MM-DD` 或 `""`) + `release_date_tba` (bool)。两者均参与 revision/PR diff。详见 [01-galgame.md PUT 端点](./01-galgame.md#put-galgamegid)。
> - **PR2 / PR5**：新增 `galgame_cover` / `galgame_screenshot` 关联表（hash 化的 image_service 资源）；`banner_image_hash` **字段已退役**，banner 由 `covers[sort_order=0]` 唯一表达；响应里有派生只读字段 `effective_banner_hash`。详见 [03-relations.md 封面/截图段](./03-relations.md#封面--截图pr2-新增) 与 [00-handbook §15 PR5 BREAKING](./00-handbook-for-downstream.md#15-kungal--moyu-必须各自完整实现的-galgame-编辑面强制全覆盖)。
> - **PR4**：tag / official / engine / series 4 种 taxonomy 实体获得**完整版本历史 + 回滚**，端点形态与 galgame 修订完全对齐（`GET /tag/:id/revisions`, `POST /tag/:id/revert` 等共 12 个新端点）。详见 [04-taxonomy.md §修订与回滚](./04-taxonomy.md#修订与回滚-pr4-新增4-实体同款)。
> - **K-PR（2026-05-22，非破坏性）**：`GET /galgame/:gid` 详情响应里嵌入的每个 `tag.tag`、`official.official` 和 `engine.engine` **新增 `galgame_count` 字段**（已发布作品数，口径与独立的 `GET /tag` / `GET /official` / `GET /engine` 列表一致）。下游可直接在 galgame 详情页面渲染"标签 +N / 会社 +N / 引擎 +N" badge，**不再需要为每个嵌入实体单独发 `GET /tag/:name` / `GET /official/:name` 请求**。详见 [01-galgame.md GET /galgame/:gid](./01-galgame.md#get-galgamegid) 与 [04-taxonomy.md GET /official](./04-taxonomy.md#get-official)。
> - **K-PR（2026-05-22，非破坏性）**：`GET /galgame/:gid/revisions/:rev/diff` 和 `GET /galgame/:gid/prs/:id` 响应**新增 `names` 字段**，包含本次 diff 涉及的所有 tag / official / engine / series ID 到**显示名称**的映射。让前端直接渲染"tag added: 校园"而不是"tag added: 42"，避免 N+1 后续查询。被删除的实体 key 不出现在 names 里，前端 fallback 渲染 `已删除 #<id>` 即可。详见 [02-revisions-and-prs.md](./02-revisions-and-prs.md#get-galgamegidrevisionsrevdiff)。
> - **K-PR（2026-05-28，非破坏性）**：发售日期筛选。`GET /galgame` 与 `GET /galgame/search` 的 `released_from` / `released_to` 升级为 **`YYYY` 或 `YYYY-MM`**（年/月精度，纯年仍向后兼容）；`GET /galgame` 的 `sort_field` 新增 `release_date`。详见 [00-handbook §17](./00-handbook-for-downstream.md#17-发售日期-release_date-筛选协议)。
> - **K-PR（2026-05-28，非破坏性）**：新增 `released_months`（逗号分隔 1–12）—— 在连续年份区间上叠加**不连续月份**过滤（如"历年三月发售"`released_months=3`、"2020–2024 的 3/7 月"`released_from=2020&released_to=2024&released_months=3,7`），两端点同款。搜索端需跑一次 `reindex-search` 回填 `released_month`。详见 [00-handbook §17.10](./00-handbook-for-downstream.md#1710-不连续月份筛选released_months2026-05-28-新增)。
> - **K-PR（2026-05-28，bug fix）**：`release_date` 响应序列化从 RFC3339 `"2019-08-16T00:00:00Z"` 修正为文档承诺的纯日期 **`"2019-08-16"`**（消除 date-only 值的幻影时区，避免下游 off-by-one-day）。REST live / snapshot / search 三条路径现统一。详见 [00-handbook §17.7](./00-handbook-for-downstream.md#177-响应里-release_date-的格式yyyy-mm-dd不是-rfc3339)。并澄清 **`GET /galgame/batch` 轻量 DTO 不含 `release_date`**（要发售日期走详情/列表/搜索端点，§17.8）。
> - **K-PR（2026-05-28，bug fix）**：所有 **timestamp 字段统一 UTC RFC3339**（`created` / `updated` / `resource_update_time` / `created_at` / `completed_time` 等）。修复前 raw-model 端点吐带本地 offset 的串（`...-07:00`，随机器时区漂移）、DTO 端点的硬编码 `Z` 实为本地钟面（差一个偏移量）。用自定义 `Timestamp` 类型（强制 `.UTC()`）统一为 `"2026-05-10T16:24:28Z"`。详见 [00-handbook §17.9](./00-handbook-for-downstream.md#179-所有-timestamp-字段统一-utc-rfc3339z)。

---

## 响应格式

```json
{
  "code": 0,
  "message": "成功",
  "data": { ... }
}
```

分页响应的 `data` 结构：

```json
{
  "items": [...],
  "total": 42
}
```

## 认证

- **读操作（GET）**：无需认证
- **写操作（POST/PUT/DELETE）**：需要 OAuth Bearer Token

```
Authorization: Bearer <access_token>
```

access_token 由 鲲 Galgame OAuth 系统签发，JWT claims 中包含 `uid`（integer user ID）和 `roles`。

---

下一节：[01 — Galgame 核心 CRUD](./01-galgame.md)
