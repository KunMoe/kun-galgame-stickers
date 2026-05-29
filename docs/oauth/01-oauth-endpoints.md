# OAuth 2.0 端点

返回 [README](./README.md)

本节是 OAuth 2.0 Authorization Code（含 PKCE）+ Refresh Token 协议的端点契约。完整的对接走查（环境变量、回调处理、并发刷新、安全注意事项）请看 [oauth-integration-guide.md](./oauth-integration-guide.md)，Nuxt 快速上手见 [nuxt-integration-prompt.md](./nuxt-integration-prompt.md)。

| 端点 | 方法 | 鉴权 | 用途 |
|------|------|------|------|
| `/oauth/authorize` | GET | 终端用户 Bearer Token | 用户同意授权后拿授权码 |
| `/oauth/token` | POST | client_id+secret（或 PKCE） | 授权码 / refresh_token 换 access_token |
| `/oauth/userinfo` | GET | Bearer Token | 读用户公开信息（受 scope 控制） |
| `/oauth/revoke` | POST | 不需要 | 主动吊销 token（登出） |

---

## POST /oauth/token

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

> **限流**：此端点用 client_id 维度限流（不是 IP），所以 kungal/moyu 这种走 SSR 后端代理整个用户群的 confidential client 不会被踢到匿名 IP 桶里。一般不会撞到上限。

---

## GET /oauth/authorize

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

**授权码有效期**：10 分钟，一次性使用。

---

## GET /oauth/userinfo

获取当前登录用户信息。OIDC 标准端点；scope 控制下哪些字段会被返回。

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
>
> 想拿更全字段（如 `moemoepoint`、`bio`、`avatar_image_hash`）可以走 [GET /auth/me](./02-user-profile.md#get-authme)；想批量回拉多个用户走 [GET /users/batch](./03-cross-service.md#get-usersbatch)。

---

## POST /oauth/revoke

吊销令牌。遵循 RFC 7009，无论成功失败都返回 200。

**请求体**：

```json
{
  "token": "要吊销的 refresh_token"
}
```

---

错误码（15001 - 15009）的详细含义见 [04-tokens-and-errors.md §OAuth 错误](./04-tokens-and-errors.md#oauth-错误-15xxx)。
