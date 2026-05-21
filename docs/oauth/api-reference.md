# KUN OAuth API 参考

所有 API 基础路径：`/api/v1`

所有响应格式：

```json
{
  "code": 0,       // 0=成功, 非零=错误码
  "message": "成功",
  "data": { ... }  // 成功时有数据，失败时无
}
```

---

## OAuth 2.0 端点

### POST /oauth/token

用授权码或刷新令牌换取 access token。

**请求体（授权码模式）**：

```json
{
  "grant_type": "authorization_code",
  "code": "64位hex授权码",
  "redirect_uri": "https://www.kungal.com/auth/callback",
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "code_verifier": "PKCE验证器（如果authorize时使用了code_challenge）"
}
```

**请求体（刷新令牌模式）**：

```json
{
  "grant_type": "refresh_token",
  "refresh_token": "eyJhbGc...",
  "client_id": "your-client-id",
  "client_secret": "your-client-secret"
}
```

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "access_token": "eyJhbGc...",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "eyJhbGc...",
    "scope": "openid profile"
  }
}
```

| 字段 | 说明 |
|------|------|
| access_token | JWT，有效期 15 分钟 |
| token_type | 固定 "Bearer" |
| expires_in | 900 秒（15 分钟） |
| refresh_token | JWT，有效期 7 天。每次刷新会轮换 |
| scope | 可选，回显授权时的 scope |

---

### GET /oauth/authorize

获取授权码。用户必须已登录（带 Bearer Token）。

**查询参数**：

| 参数 | 必填 | 说明 |
|------|------|------|
| client_id | 是 | OAuth 客户端 ID |
| redirect_uri | 是 | 回调地址，必须与注册时一致 |
| response_type | 是 | 固定 `code` |
| state | 是 | 随机字符串，防 CSRF |
| scope | 否 | 权限范围，空格分隔 |
| code_challenge | 否 | PKCE code challenge |
| code_challenge_method | 否 | `S256`（默认）或 `plain` |

**成功响应**：HTTP 302 重定向到 `redirect_uri?code=xxx&state=xxx`

**授权码有效期**：10 分钟，一次性使用

---

### GET /oauth/userinfo

获取当前登录用户信息。

**请求头**：`Authorization: Bearer <access_token>`

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": 12345,
    "sub": "550e8400-e29b-41d4-a716-446655440000",
    "name": "KUN",
    "email": "kun@kungal.com",
    "picture": "https://...",
    "roles": ["user", "admin"],
    "updated_at": 1234567890
  }
}
```

| 字段 | 说明 |
|------|------|
| id | 用户整数 ID（= OAuth `users.id`，与 kungal/moyu 业务表的 `user_id` 外键对齐） |
| sub | 用户 UUID（OIDC 标准的 subject），与 `id` 标识同一用户，调用方任选其一 |
| name | 用户名（仅 `profile` scope 或空 scope 时返回） |
| email | 邮箱（仅 `email` scope 或空 scope 时返回） |
| picture | 头像 URL（仅 `profile` scope 或空 scope 时返回，可能为空） |
| roles | 角色名称数组，与 JWT `roles` claim 一致 |
| updated_at | 最后更新时间（Unix 时间戳） |

**关于 scope 与字段过滤**：

`id`、`sub`、`roles` 始终返回（不被 scope 过滤）—— 因为这三项已经在 JWT 里，调用方既然能用这个 JWT 调 /userinfo，就已经拿到了这些信息，再隐藏没有意义。`name`、`email`、`picture` 按 OIDC 标准受 `profile` / `email` scope 控制。

> **跨服务接入提示**：kungal/moyu/galgame_wiki 后端处理 OAuth callback 时，应该在登录环节就拿 `id` 入库（作为本地 user 表的主键 / 外键），不要只存 `sub` —— 后续业务表关联、`/users/batch` 批量回拉、SDK 缓存键，全部基于 `id` 整数键。

---

### POST /oauth/revoke

吊销令牌。遵循 RFC 7009，无论成功失败都返回 200。

**请求体**：

```json
{
  "token": "要吊销的 refresh_token"
}
```

---

## 用户自助资料管理

这一组端点是给"已登录用户用自己的 access_token 修改自己资料"用的。kungal/moyu 等下游服务有两种典型用法：

1. **跳转模式**（推荐）：站点的"修改头像/简介"按钮直接跳转到 OAuth 前端的 profile 页面，让用户在 OAuth 完成修改。
2. **代理模式**：站点保留自己的"修改资料"端点，但内部把请求转发到下面这些 OAuth 端点。要求请求带的是终端用户 JWT（不是 OAuth Client Basic Auth）。

### GET /auth/me

获取当前登录用户的完整资料。与 `/oauth/userinfo` 的区别：`/auth/me` 是面向 OAuth 自己前端的内部端点，无 scope 过滤、字段更全（含 moemoepoint）。下游服务若用得着也可以调。

**请求头**：`Authorization: Bearer <access_token>`

**成功响应**：

```json
{
  "code": 0,
  "data": {
    "uuid": "550e8400-e29b-...",
    "name": "kun",
    "email": "kun@kungal.com",
    "avatar": "https://...",
    "bio": "...",
    "moemoepoint": 1234,
    "status": 0,
    "roles": ["user", "admin"],
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### PATCH /auth/me

修改当前登录用户的展示字段。所有字段都可选，不传的字段保持不变。

**请求头**：`Authorization: Bearer <access_token>`

**请求体**：

```json
{
  "name": "newname",
  "avatar": "https://...",
  "avatar_image_hash": "abc123...",
  "bio": "新简介"
}
```

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| name | string? | 2..17 字符；全局唯一 | 用户名 |
| avatar | string? | ≤255 字符 | 头像 URL（legacy；image_service 普及前继续用） |
| avatar_image_hash | string? | ≤64 字符 | 头像的 image_service 哈希；前端 resolveAvatarUrl 优先用此字段 |
| bio | string? | ≤107 字符 | 个人简介 |

字段都用指针类型语义：**没传 = 不动；传了 = 设为该值**（包括传空字符串 = 清空）。

**成功响应**：返回更新后的完整 `UserResponse`（同 GET /auth/me 的 `data` shape）。

**错误响应**：

| HTTP | code | 触发条件 |
|------|------|----------|
| 400  | 1    | JSON 格式错误 |
| 400  | 7    | 字段约束未通过（name 长度、bio 长度等） |
| 400  | 10007 | name 与其他用户重复 |
| 401  | 10001/10002/10003 | 未提供 / 无效 / 过期 token |

**修改 email 不在这里** —— email 必须走 `/auth/email/send-code` + `/auth/email`（带验证码的两步流程，防止账号被劫持）。

**修改 password 也不在这里** —— password 必须走 `/auth/password`（需要旧密码或重置 token）。

**举例**：仅改头像 hash（image_service 上传完毕之后）：

```bash
curl -X PATCH https://oauth.kungal.com/api/v1/auth/me \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"avatar_image_hash":"abc123def456..."}'
```

---

## 跨服务批量查询

### GET /users/batch

跨服务批量获取用户公开资料。专为 kungal / moyu / galgame_wiki 等下游业务使用 ——
这些服务不在本地缓存 `users.name` / `users.avatar`，渲染时按 `user_id` 列表回拉。

**鉴权**：OAuth Client Basic Auth（`Authorization: Basic base64(client_id:client_secret)`）。
不是终端用户 JWT。任何已注册的 OAuth Client 都可以调用。

**查询参数**：

| 参数 | 必填 | 说明 |
|------|------|------|
| ids | 是 | 1..100 个用户 ID（OAuth 用户表主键），逗号分隔，如 `?ids=1,2,3` |

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "users": [
      {
        "id": 1,
        "uuid": "9e00220a-8079-4e81-8e98-49e26ce23edc",
        "name": "kun",
        "avatar": "https://image.kungal.com/avatar/user_1/avatar.webp",
        "avatar_image_hash": "abc123...",
        "bio": "KUN IS THE CUTEST!",
        "status": 0,
        "roles": ["admin"]
      }
    ],
    "not_found": [9999]
  }
}
```

| 字段 | 说明 |
|------|------|
| users[].id | 用户 ID（与 kungal/moyu 中 `*_user_id` 外键对齐） |
| users[].uuid | 用户 UUID |
| users[].name | 用户名 |
| users[].avatar | 头像 URL（可能为空字符串） |
| users[].avatar_image_hash | 头像 image_service 哈希（可空） |
| users[].bio | 个人简介 |
| users[].status | 0=正常；非 0 时调用方应隐藏或脱敏渲染 |
| users[].roles | 角色名称数组，如 `["admin"]` |
| not_found | 请求中存在但 OAuth 库里查不到的 ID 列表 |

**错误响应**：

| HTTP | code | 触发条件 |
|------|------|----------|
| 400  | 9    | `ids` 为空或包含非数字 |
| 400  | 9    | `ids` 个数超过 100 |
| 401  | 10001/15001/15009 | Basic Auth 缺失/格式错/client_id 不存在/secret 错误 |

**注意**：响应中**不包含** `email`、`moemoepoint`、`created_at` 等隐私字段。
若调用方需要邮箱（如发邮件通知），应该走专门的 RPC 而不是渲染管线。

**客户端实现**：OAuth 这边**不发布 SDK 代码**。每个 consumer 自己实现一个薄客户端（30 行起步，按工作负载需要加 TTL 缓存 / singleflight / 分片）。完整的实现指南、可直接复用的 Go 参考代码、以及决定层级的判断标准，见 [docs/migration/user/08-downstream-integration.md §4](../../migration/user/08-downstream-integration.md#4-客户端实现指南)。

---

### GET /users/search

按用户名搜索用户，case-insensitive 子串匹配。结果按相关度排序：精确匹配 > 前缀匹配 > 子串匹配，每一档内按字母升序。

适用场景：@提及自动补全、用户搜索框、管理后台用户检索。

**鉴权**：与 `/users/batch` 相同（OAuth Client Basic Auth）。

**查询参数**：

| 参数 | 必填 | 说明 |
|------|------|------|
| q | 是 | 搜索关键词，trim 后 1..50 字符。`%` `_` `\` 等 LIKE 通配符按字面匹配（已转义） |
| limit | 否 | 返回条数，默认 20，封顶 50 |

**成功响应**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "users": [
      { "id": 2, "uuid": "...", "name": "鲲", "avatar": "...", "bio": "...", "status": 0, "roles": ["admin"] },
      { "id": 79063, "uuid": "...", "name": "鲲1", ... },
      { "id": 38359, "uuid": "...", "name": "鲲114514", ... }
    ]
  }
}
```

**错误响应**：

| HTTP | code | 触发条件 |
|------|------|----------|
| 400  | 9    | `q` 为空或缺失 |
| 400  | 9    | `q` 超过 50 字符 |
| 400  | 9    | `limit` 不是正整数 |
| 401  | 10001/15001/15009 | Basic Auth 缺失/格式错/凭证错 |

> **注意**：搜索结果**不应缓存**（query 空间无界、结果随注册/改名漂移，缓存命中率低还容易出脏数据）。前端要做实时自动补全，调用方在前端 debounce（推荐 200–300ms）即可。

---

## JWT Access Token Claims

```json
{
  "sub": "用户UUID",
  "email": "邮箱",
  "name": "用户名",
  "roles": ["user", "admin"],
  "site_id": 0,
  "exp": 1700000000,
  "iat": 1699999100,
  "nbf": 1699999100
}
```

签名算法：HS256

---

## 错误码速查

### OAuth 错误 (15xxx)

| Code | HTTP | 消息 | 说明 |
|------|------|------|------|
| 15001 | 400 | 无效的客户端 | client_id 不存在 |
| 15002 | 400 | 无效的回调地址 | redirect_uri 未注册 |
| 15003 | 400 | 无效的授权码 | code 已过期 / 已使用 / 不存在 |
| 15004 | 400 | 无效的代码验证器 | PKCE code_verifier 不匹配 |
| 15005 | 400 | 无效的授权类型 | client 的 `grants` 列里没有当前 grant_type — **常见：admin 创建 client 时漏勾 `refresh_token`** |
| 15006 | 400 | 无效的权限范围 | 请求的 scope 不在 client 的 `allowed_scopes` 内 |
| 15007 | 400 | 访问被拒绝 | 用户拒绝授权 |
| 15008 | 400 | 无效的 client secret | confidential client 没传或填错 client_secret |
| 15009 | 400 | 需要 PKCE | public client 没传 code_verifier |

### 认证错误 (10xxx)

| Code | HTTP | 消息 | 说明 |
|------|------|------|------|
| 10001 | 401 | 未授权 | 未提供 Bearer Token |
| 10002 | 401 | 无效的令牌 | Token 格式错误 / 签名无效 / **refresh 时 client_id 与签发时的不匹配** |
| 10003 | 401 | 令牌已过期 | access_token 或 refresh_token 已过期，需要刷新或重新登录 |
| 10005 | 401 | 用户不存在 | UUID 对应的用户不存在（账号被硬删等罕见情况） |
| **10014** | **403** | **账号已封禁** | 用户被 admin 封号 — **前端应跳错误页（"账号被封禁"）而非登录页**，让用户再登也是同样的 403 |

### 通用错误

| Code | 消息 |
|------|------|
| 1 | 请求格式错误 |
| 7 | 参数验证失败 |
| 10 | 操作失败 |
