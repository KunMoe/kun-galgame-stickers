# 01 — 数据清单：表情包站现在有什么

> 面向 infra。这一份只讲**事实**：库里有什么、字段是什么意思、哪些能对外、哪些不能。
> 开放方案在 [02-opening-the-api.md](./02-opening-the-api.md)。
> 数字采自生产库 `kungalgame_sticker`，2026-09-08。

## 0. 一句话

表情包站是 **catalog 的一个消费者**，也是**图片的一个持有者**：它自己只存「哪几张图属于哪一套、按什么顺序、是谁发的」，而**作品与人物的身份全部委托给 catalog**，**图片字节全部住在 image service**。所以对外开放时，我们能给出的独有数据只有一样——**贴纸集合与它们到 catalog 身份的映射**。

## 1. 规模

| 项 | 数量 | 说明 |
|---|---:|---|
| 表情包（pack） | 7 | 全部已发布、全部 `is_official` |
| 贴纸（sticker） | 498 | 每张一个 image hash，无重复 |
| 关联到 catalog 作品的贴纸 | 498 | 100% |
| 关联到 catalog 人物的贴纸 | 497 | 差的 1 张画面里有两个角色，一贴纸一人物表达不了 |
| 涉及的 catalog 作品 | 95 | 其中 **89 部是 r18** |
| 涉及的 catalog 人物 | 128 | |
| 标签（tag） | 0 | 表结构在，官方包没打标签 |
| 发布者 | 1 | 目前只有官方账号（uid 2） |
| 累计下载 / 浏览 | 2 / 55 | 站点刚重建，数据没有参考价值 |

**给 infra 的判断依据**：这不是一个「大数据量」的面。它的价值不在体量，在于 **498 张图 × 128 个 catalog 人物身份的映射**——这是全生态唯一一份「galgame 角色 → 可用表情素材」的索引。下游想要的多半是「给我 XX 的表情」，而不是「给我全部贴纸」。

## 2. 数据模型

四张表，主键全是 PostgreSQL 18 原生 `uuidv7()`。

### `pack` — 表情包

```
id                  uuid          PK, uuidv7
owner_uid           integer       发布者，OP 的 user id（本站不存 user 表）
status              smallint      0 草稿 / 1 已发布 / 2 隐藏 / 3 移除
is_official         boolean
content_rating      smallint      0 全年龄 / 1 R18（本站自己的分级，作者填）
title               jsonb         多语言，键空间见 §3
description         jsonb         多语言
cover_sticker_id    uuid          → sticker.id，ON DELETE SET NULL
sticker_count       integer       写路径维护，非触发器
view_count          bigint
download_count      bigint
search_text         text          各语言拍平，trigram GIN 索引载体
created_at / updated_at / published_at   timestamptz
catalog_work_id     bigint        ← catalog 作品身份（可空）
catalog_work_name   jsonb         展示快照
catalog_work_cover  text          展示快照（catalog CDN URL）
catalog_work_rating text          展示快照，catalog 的 all_ages|sensitive|r18
```

### `sticker` — 贴纸

```
id                      uuid       PK, uuidv7
pack_id                 uuid       → pack.id, ON DELETE CASCADE
position                integer    UNIQUE(pack_id, position)，可重排
image_hash              text       image service 的 sha256，本站不存字节
width / height          integer
game / character_name   jsonb      2026 年之前人工标注的自由文本，保留作回退
vndb_id                 integer    历史列，生产全为 NULL
note                    text
catalog_work_id         bigint     ← catalog 作品身份（可空）
catalog_work_name       jsonb      展示快照
catalog_work_rating     text       展示快照
catalog_character_id    bigint     ← catalog 人物身份（可空）
catalog_character_name  jsonb      展示快照
catalog_character_image text       展示快照（catalog CDN URL）
created_at / updated_at            timestamptz
```

**为什么两级都有 `catalog_work_id`**：7 套官方包是混合包，每套横跨 12–32 个不同游戏，包级别的一个作品描述不了它们；而单作品的用户包又需要在卡片和筛选上声明一次。所以包级是「这套包声明的作品」，贴纸级是「这张图的角色出自哪部作品」，两者都可空。

**快照的性质**：`catalog_*_name` / `_cover` / `_image` / `_rating` 全部是**展示缓存**，catalog 永远是身份的真相。它们存在的唯一理由是列表页不必为了画一个名字去调上游，以及 catalog 不可达时页面仍然完整。中文名缺失时用本站手工翻译补空（catalog 对其中 27 个人物没有中文名）。

### `tag` / `pack_tag`

```
tag       id uuid PK, slug text UNIQUE, name jsonb, pack_count integer
pack_tag  (pack_id, tag_id) PK
```
`pack_count` 只数已发布的包。

### ~~`comment_like`~~（已由 migration 000009 删除）

曾是 community reaction 在本地的镜像，理由是 community 的 post 投影不带任何 reaction 字段。community 后来在读接口上补了 `reaction_count` / `viewer_reacted`，开关也直接返回新计数，这张表就退役了：点赞只存在于 community，本站一行都不存。

## 3. 多语言键空间

数据库用 `zh-cn / zh-tw / ja-jp / en-us`，比站点的三个 UI locale 宽。catalog 用 BCP-47（`zh-Hans` / `zh-Hant` / `ja` / `en`），写入时映射成上面四个键，另加一个 **`und`** 存 catalog 的 canonical `display_name`（通常是日文原名），作为任何语言都缺时的兜底。

对外开放时**建议原样吐这个 map**，不要在服务端替调用方选语言：下游站点的语言偏好不归我们决定。

⚠️ 一个已知的上游数据怪癖：catalog 把人物的**罗马音存在 `lang=ja` 下**（查 久島鴎 拿到 `localized["ja"] = "Kushima Kamome"`）。本站入库时按字形判断，拉丁字母的值不进日语槽而进 `en-us`，所以我们的 `ja-jp` 是可信的。**下游如果直连 catalog 会踩到这个坑，走我们的面不会。**

## 4. 图片

贴纸字节住在 image service，preset `sticker`，variant `320` / `128`。URL 形如：

```
{cdnBase}/{hash[0:2]}/{hash[2:4]}/{hash}.webp          原图
{cdnBase}/{hash[0:2]}/{hash[2:4]}/{hash}_320.webp      缩略
```

- 图片**跨全生态去重**：同一个 hash 可能同时被别的站引用。所以删贴纸**不调 image delete**，只是不再进每日 reference-ping，由 image service 按 60 天转冷 / 365 天软删的 TTL 回收。
- 开放 API 若直接吐 CDN URL，等于把这些 hash 的引用扩散到我们控制不到的地方。**这一点需要 infra 拍板**（见 02 §5）。

## 5. 哪些数据不能对外

| 数据 | 原因 |
|---|---|
| `status != 1` 的包 | 草稿/隐藏，只有作者和有 `pack.view_hidden` 的人能看 |
| `owner_uid` 之外的用户信息 | 本站不存 user 表，姓名头像来自 OP 批量接口，是 OP 的数据不是我们的 |
| 评论正文 | 住在 community，租户是 `sticker`；要开放得由 community 决定，不该由本站转发 |
| 评论点赞 | 同上；本站不存，计数和「我赞过」都直接读 community |
| 评论通知、关注 | 同上；收件箱和关注状态都是 community 按投递站 `sticker` 存的，本站只读和回执 |
| `search_text` | 内部索引载体，拍平了各语言，对调用方无意义 |
| 站内管理端点（`/me/*`） | 与 infra 03 §4、06 §11 同则：staff / 管理端点永不入面 |

## 6. 现有 API 面（站内 BFF，cookie 会话）

开放面**不应该**是这一套的复制品——它是给自己前端用的，鉴权是 httpOnly cookie。列在这里是为了让 infra 看见哪些读路径已经存在、可以复用实现。

```
公开读（无鉴权，带 ETag + Cache-Control）
  GET  /api/v1/packs                       分页 / 排序 / 搜索 / 标签 / 分级 / linked / work
  GET  /api/v1/tags
公开读（optionalAuth，作者能看到自己的草稿）
  GET  /api/v1/packs/{packId}
  GET  /api/v1/packs/{packId}/download     整包 zip
  GET  /api/v1/stickers/{stickerId}
  GET  /api/v1/stickers/{stickerId}/download
  GET  /api/v1/users/{uid}/packs
  GET  /api/v1/characters/{characterId}    catalog 资料 + 本站该人物的贴纸
  GET  /api/v1/search                      表情包 + 人物，供命令面板

会话写（cookie，本站前端专用）
  /api/v1/me/packs/**                      建包 / 改包 / 发布 / 上传 / 排序
  /api/v1/packs/{id}/comments, /comments/** 评论、点赞、举报、关注
  /api/v1/me/notifications/**              通知收件箱、未读数、标已读（转发 community）
  /api/v1/catalog/works, /catalog/works/{id}/characters   编辑器的选择器（转发 catalog）
```

响应统一信封 `{ code, message, data }`，`code === 0` 为成功；错误码分域（90001–90019）。

## 7. 与其他 infra 服务的现状

| 服务 | 本站的关系 | 凭据 |
|---|---|---|
| OAuth / OP | RP，BFF httpOnly cookie | client `c5cd7b07…` |
| image service | 上传 + reference-ping | 同一 client id/secret，preset `sticker` |
| catalog `/v2` | 只读消费者 | 应用密钥 `nmk_live_…`，scope `catalog:read`，tier `internal` |
| community | 评论租户 `sticker` | 同一 client id/secret（Basic），tenant 由 `catalog_site` 推导 |
| nextmoe-og | 分享图 | per-site key `sticker` |

**本站从不写 catalog**，`catalog_site='sticker'` 这个绑定目前只用于 community 租户。
