# 鲲 Galgame OAuth 文档

基础路径：`/api/v1`

| 环境 | Base URL |
|------|----------|
| 开发 | `http://127.0.0.1:9277/api/v1` |
| 生产 | `https://oauth.kungal.com/api/v1` |

## 🔒 重要约定：身份操作必须在 OAuth 完成

下游 kungal / moyu / wiki **不要在自己前端实现下列操作**：

- **新用户注册**（跳转到 `oauth.kungal.com/auth/register?redirect=<oauth-authorize-url>`，注册成功后自动 SSO 回跳）—— 详见 [05-registration.md](./05-registration.md)
- **改邮箱**（POST /auth/email/send-code + PUT /auth/email）
- **改密码**（PUT /auth/password）
- 重设密码 / 启用 2FA / 管理登录设备 / 注销账号 / 撤销已授权 OAuth Client（未来）

跳转目标：注册去 `/auth/register?redirect=...`，账号管理去 `https://oauth.kungal.com/profile`。

技术上这些端点都能通过 end-user JWT 代理，但身份层操作**必须集中在一个前端**：安全审计单点、未来加 2FA / 异地通知时只改一处、避免邮箱劫持攻击面跨多个站点放大。

展示层操作（name / avatar / bio）可以站内提供 UI 或跳转，任选。

详细分类表 + 跳转按钮代码示例见 [02-user-profile.md §身份操作 vs 展示操作](./02-user-profile.md#身份操作-vs-展示操作)。

---

## 文档索引

### API 参考（按主题）

| # | 文件 | 内容 |
|---|------|------|
| 01 | [oauth-endpoints.md](./01-oauth-endpoints.md) | OAuth 2.0 协议端点：`/oauth/token`、`/oauth/authorize`、`/oauth/userinfo`、`/oauth/revoke` |
| 02 | [user-profile.md](./02-user-profile.md) | 用户自助：`GET/PATCH /auth/me` + `POST /auth/me/avatar`（含头像上传） |
| 03 | [cross-service.md](./03-cross-service.md) | 服务到服务：`/users/batch`、`/users/search`（OAuth Client Basic Auth） |
| 04 | [tokens-and-errors.md](./04-tokens-and-errors.md) | JWT Access Token claims + 完整错误码速查（OAuth 15xxx / 认证 10xxx / 通用） |
| 05 | [registration.md](./05-registration.md) | 🆕 用户注册流程：跳转 OAuth 注册 + 邮箱验证码 + 自动 SSO 回跳；`POST /auth/register/send-code` + `POST /auth/register`、`GET /oauth/client-info`；下游 PKCE 跳转示例 |
| 06 | [moemoepoint.md](./06-moemoepoint.md) | 🚧 **设计规范（精简版）**：萌萌点全站统一货币（单一真源在 OAuth）。可变余额列 + append-only 审计日志 + 幂等发放/扣除 RPC + 迁移与下游接入；含"刻意没做的"清单（将来需要再升级）|

### 完整接入指南

| 文件 | 内容 |
|------|------|
| [oauth-integration-guide.md](./oauth-integration-guide.md) | 端到端 OAuth 接入走查：注册 client、PKCE、token 轮换、并发刷新、跨域 / 跨站坑、安全注意事项 |
| [nuxt-integration-prompt.md](./nuxt-integration-prompt.md) | Nuxt 3/4 项目的快速上手指南（含 SSR 回调处理代码） |

---

## 响应格式

```json
{
  "code": 0,        // 0 = 成功，非零 = 错误码
  "message": "成功",
  "data": { ... }   // 成功时有数据；失败时一般为 null 或缺省
}
```

> ⚠️ **业务级 401 走 HTTP 200**：`/auth/me` / `/auth/*` 等需要 token 的端点在 token 缺失 / 失效 / 过期时返回 **HTTP 200** + `{ code: 10001 | 10002 | 10003, message }`，而**不是** HTTP 401。下游客户端不能只靠 HTTP status 判断未授权，必须同时检查 `code`。完整列表见 [04-tokens-and-errors.md §认证错误](./04-tokens-and-errors.md#认证错误-10xxx)。

## 认证

OAuth 一共有三种鉴权方式，按场景区分：

| 方式 | 用在哪 | 谁有 |
|------|------|------|
| **Bearer Token**（用户 JWT） | `/auth/*` 用户自助 + `/oauth/userinfo` | 已登录的终端用户 |
| **OAuth Client Basic Auth** | `/users/batch`、`/users/search`（跨服务） | 已注册的 OAuth Client（kungal / moyu / wiki 等下游后端） |
| **Admin JWT**（Bearer + role=admin） | `/admin/*`（不在本文档范围） | OAuth 后台管理员 |

终端用户 JWT 通过完整的 OAuth Authorization Code + PKCE 流程拿到（详见 [oauth-integration-guide.md](./oauth-integration-guide.md)）。Client Basic Auth 的 client_id / client_secret 在 OAuth 后台创建 Client 时生成。

---

## 变更摘要

> 🆕 **2026-05-23 注册流程统一（L1，重要）**：新增 [05-registration.md](./05-registration.md) 文档；引入**邮箱验证码两步注册**——`POST /auth/register/send-code` 寄码 + `POST /auth/register` 带 code 创建账号并发 token（**注册即登录**，返回 access_token + 写 refresh cookie）；新增 [GET /oauth/client-info](./05-registration.md#get-oauthclient-info) 公开元数据端点；`oauth_clients` 加 `auto_consent` 列，5 个第一方 client 默认开启——同意页对第一方静默跳过，用户感知是"注册完一闪回到原站点已登录"。下游 kungal / moyu 的 legacy 注册端点全部删除，"注册"按钮改为复用登录的 PKCE 跳转模式（目标 URL 换成 `/auth/register?redirect=<authorize_url>`）。

> 🔒 **2026-05-23 政策**：明确"身份层 vs 展示层"分类。下游禁止在自己前端做改邮箱 / 改密码 / 注销账号等身份操作，必须跳转 OAuth profile。详见上方"重要约定"小节和 [02-user-profile.md](./02-user-profile.md#身份操作-vs-展示操作)。

> 🆕 **2026-05-23**：新增 [POST /auth/me/avatar](./02-user-profile.md#post-authmeavatar) 端点。一次性的"上传头像图片 → 写库" multipart 端点，**避免下游 kungal / moyu 自己维护 image_service client**。配额从 OAuth 一侧扣；老的两步法（`PATCH /auth/me { avatar_image_hash }`）继续保留。

> 🆕 **2026-05-23**：正式收录 [POST /auth/email/send-code](./02-user-profile.md#post-authemailsend-code) / [PUT /auth/email](./02-user-profile.md#put-authemail) / [PUT /auth/password](./02-user-profile.md#put-authpassword) 端点文档（以前只有口头提及）。同时把对应的错误码 10004 / 10006 / 10010-10013 补全到 [04-tokens-and-errors.md](./04-tokens-and-errors.md#认证错误-10xxx)。

> 📦 **文档拆分（2026-05-23）**：原 `api-reference.md` 拆为 4 个主题文件（01-04）。所有内容保留，按"OAuth 协议 / 用户自助 / 跨服务 / Token 与错误"四块组织。完整 OAuth 接入指南仍是单独的 [oauth-integration-guide.md](./oauth-integration-guide.md)。
