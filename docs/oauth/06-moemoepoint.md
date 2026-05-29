# 06 — 萌萌点（moemoepoint）统一货币（精简版）

返回 [README](./README.md)

> 🚧 **状态：设计规范（待实现）**。库表 / 端点 / 错误码尚未落地。

## 0. 决策与定位

- **moemoepoint 全站统一**：一个用户在 kungal / moyu / 未来所有接入站点**共享一个余额**，**单一真源在 OAuth**（共享身份库）。
- 本设计**刻意精简**：萌萌点是软性 karma（非货币、低频写入、出错最坏只是"数字不对"，不涉资损）。所以只保留三个真正有价值的属性 —— **幂等、审计、单源** —— 砍掉金融账本级的严谨度和多团队治理（详见 §9 与早期完整版的差异）。

## 1. 数据模型

### 1.1 余额：`users.moemoepoint`（可变列，保留）

仍是一个普通可变整型列，作为**当前余额**。每次调整在**同一事务**里 `moemoepoint += delta`。不把它变成"日志的派生值"——对 karma 来说没必要。

### 1.2 审计日志：`moemoepoint_log`（只追加，不改不删）

```sql
CREATE TABLE moemoepoint_log (
  id              BIGSERIAL PRIMARY KEY,
  user_id         INTEGER     NOT NULL,
  delta           INTEGER     NOT NULL,              -- 有符号，非 0
  reason          VARCHAR(40) NOT NULL,              -- 见 §2 小枚举
  source_app      VARCHAR(32) NOT NULL,              -- 来源站点（服务端从认证 client 推导）
  ref             VARCHAR(80),                       -- 触发实体，自由格式如 "galgame:1207"（可空）
  actor_user_id   INTEGER     NOT NULL DEFAULT 0,    -- 谁导致：0=系统 / 管理员 id
  idempotency_key VARCHAR(128) NOT NULL,             -- 防重放，全局唯一
  note            VARCHAR(255),                       -- 备注（管理员操作填）
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_mp_log_idem ON moemoepoint_log (idempotency_key);
CREATE INDEX        idx_mp_log_user ON moemoepoint_log (user_id, id DESC);
CREATE INDEX        idx_mp_log_reason ON moemoepoint_log (reason);  -- 分类查看
```

- 日志**只追加**：发错了写一条反向记录（`delta` 取负），不改旧行。
- "分类查看" = 按 `reason`（或 `source_app`）过滤 / 聚合。

## 2. reason（一张小而稳的通用枚举，OAuth 拥有）

扁平、通用、少。具体业务细节靠 `source_app` + `ref` 区分，**不为每个站点维护各自的枚举**（那是被砍掉的治理层）。

| reason | 方向 | 说明 |
|---|---|---|
| `admin_grant` / `admin_deduct` | ± | 管理员发放 / 扣除 |
| `migration` | + | 迁移起始值（§6）|
| `content_approved` | + | 产出被采纳（Wiki 投稿通过、补丁发布…，用 source_app+ref 区分）|
| `content_removed` | − | 上述产出被删 / 撤回时回收（与发放同 `ref`）|
| `daily_checkin` | + | 每日签到 |
| `liked` | + | 内容被点赞 |

约定：`delta` 禁止为 0；可回收的产出，回收用相同 `ref` 对账。新增一种来源 = 往这张表加一行（你一个人控制全部 client，一次小改即可，无需治理协调）。

## 3. 服务到服务 API

**鉴权**：与 [`/users/batch`](./03-cross-service.md) 相同 —— **OAuth Client Basic Auth**。`source_app` 服务端从认证 client 推导，不信任请求体自报。

| 端点 | 方法 | 用途 |
|------|------|------|
| `/users/:id/moemoepoint` | POST | **调整**余额（发放 / 扣除），幂等 |
| `/users/:id/moemoepoint` | GET | 读当前余额 |
| `/users/:id/moemoepoint/log` | GET | 分页拉流水（可选；用户"积分明细" / 排查）|

### 3.1 POST /users/:id/moemoepoint

```json
{
  "delta": 3,
  "reason": "content_approved",
  "ref": "galgame:1207",
  "actor_user_id": 0,
  "idempotency_key": "moyu:wiki_approved:1207",
  "note": ""
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| delta | 是 | 有符号整数，非 0，且 \|delta\| ≤ 1,000,000（防呆上限）|
| reason | 是 | §2 枚举之一。**s2s 不可用** `admin_grant` / `admin_deduct` / `migration`（OAuth 保留）|
| ref | 否 | 触发实体（建议填，用于对账）|
| actor_user_id | 否 | 默认 0（系统）；管理员操作填管理员 id |
| idempotency_key | 是 | 全局唯一，**调用方生成稳定键**（见 §4）|
| note | 否 | 备注 |

**成功响应**（首次执行 与 幂等重放 一致）：

```json
{ "code": 0, "message": "成功", "data": { "user_id": 1207, "balance": 42, "applied": true } }
```

`applied=false` 表示幂等键命中、未重复执行。

**错误**（HTTP 400 + 对应 code，除非另注）：`16002` delta 为 0 或超 ±1,000,000；`16003` reason 非法 / 用了保留 reason；`16004` 幂等键已存在但请求体不一致；`404/10005` 用户不存在；`401` Basic Auth 失败。

> 余额**允许为负**（精简取舍：不做非负约束，保证回收/反转永不被挡）。

### 3.2 读取

- `GET /users/:id/moemoepoint` → `{ "balance": 42 }`。
- `GET /users/:id/moemoepoint/log?limit=20&before_id=&reason=` → 分页流水（`reason` 可选过滤）。**s2s 返回精简视图**：`{ id, delta, reason, source_app, ref, created_at }`，**不含** `note` / `actor_user_id`（这俩可能含管理处罚备注，下游可能渲染给终端用户，故不下发；管理端 `/admin/.../log` 返回完整视图）。
- 也可在 `/auth/me` / userinfo 里直接返回 OAuth 的实时余额（替掉现在的冻结快照）。

## 4. 幂等（唯一需要严谨的点）

下游发放常由会重试 / 重放的路径触发（典型：moyu cron「Wiki 消息 → +3」），没有幂等就会重复加分。

- 调用方为**每个业务事件**生成**稳定**键，推荐 `<app>:<event>:<事件唯一id>`，如 `moyu:wiki_approved:1207`、`kungal:checkin:1207:2026-05-29`。
- 服务端：`idempotency_key` 唯一索引。已存在 → 不重复执行，回原结果（`applied:false`）。请求体不一致 → `400/16004`。
- 写入在单事务内对该用户行加锁（`SELECT … FOR UPDATE`）防并发竞态；唯一索引兜底。

## 5. 管理端

- 发放 / 扣除走**同一个 Adjust 入口**（`reason=admin_grant`/`admin_deduct`、`actor_user_id=管理员id`、`note` 填理由、幂等键用表单 token）→ 自动进同一审计日志。
- OAuth admin（用户管理页）加：发放/扣除弹窗 + 流水查看。
- **不要**给管理员开"直接编辑整型"的口子（绕过日志）。

## 6. 迁移（一次性）

1. 下游停止本地 `moemoepoint` 写入（或短期双写过渡）。
2. 每个用户取各站本地值之和作为统一起始余额。
3. 写一条 `reason=migration` 日志（`idempotency_key=oauth:migration:v1:<userId>`，可重复跑）+ 设 `users.moemoepoint`。
4. 下游删本地写逻辑，改：发放/扣除调 §3.1；显示余额回读 OAuth。

## 7. 下游接入

| 现在 | 改成 |
|---|---|
| 本地 `UPDATE user SET moemoepoint = moemoepoint + N` | 调 `POST /users/:id/moemoepoint`（Basic Auth + 稳定幂等键）|
| cron 重放发放 | 同上，幂等键用业务事件唯一 id → 重放安全 |
| 渲染余额读本地列 | 读 OAuth 实时余额（`/auth/me` 或 §3.2）|

> **可用性注意**：发放现在依赖 OAuth 可达。对**非关键**奖励（签到、点赞），调用失败应"记录待补 + 不阻塞用户主流程"，靠幂等键之后重试补发；不要让 OAuth 抖动卡住下游核心操作。

OAuth **不发布 SDK**，每个 consumer 自己写薄客户端（同 `/users/batch` 的 Basic Auth）。

## 8. 错误码（16xxx）

| code | 常量 | 含义 |
|---|---|---|
| 16002 | `ErrMoemoepointInvalidDelta` | delta 为 0 或 \|delta\| > 1,000,000 |
| 16003 | `ErrMoemoepointInvalidReason` | reason 不在枚举内，或 s2s 用了保留 reason（admin_*/migration）|
| 16004 | `ErrMoemoepointIdemConflict` | idempotency_key 已存在但请求体不一致 |

> 已实现状态：上述 3 个码 + `moemoepoint_log` 表 + s2s/admin 端点 + 管理端 UI **均已落地**（待 oauth 后端重启生效）。下游消费 + 数据合并迁移（§6/§7）仍待各站对接。
> 并发同键竞态：唯一索引兜底（不会重复加分），极少数并发同键会得到一次性 500，调用方重试即转为 `applied:false`。

## 9. 刻意没做的（将来需要时再升级）

为对齐当前规模（~9 万用户的爱好社区、单人维护、karma 非货币），以下**故意省略**——等真有需求再加：

| 砍掉的 | 完整账本版才需要 / 何时再加 |
|---|---|
| 余额 = `SUM(delta)` 派生 + `balance_after` 快照 + 定时对账巡检 | 资损级系统 / 需要逐行可证一致性时 |
| 两级 `category` 闭集 + `reason` 命名空间防伪 + per-app reason 清单模板 | 出现**多个独立团队**各自定义大量积分玩法、需要解耦发版时 |
| per-client category 白名单、单次/单日累计上限 | 萌萌点可兑换实物、出现真实刷分/欺诈动机时 |

升级是可逆且渐进的：日志表已记 `reason`/`source_app`/`ref`，将来要分两级或加约束都能在现有数据上演进，不必现在预付复杂度。
