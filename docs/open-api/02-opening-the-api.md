# 02 — 开放 API：现状核对与建议方案

> **⚠️ §3 的现状核对已过期。** 本文写于 2026-09-08 上午；当天下午 infra 就按本文 §5.1 的建议交付了 B 档校验端点（`96b689d0`）。最新接线状态见 [03-wiring.md](./03-wiring.md)；本文的建议部分（面形状、四件拍板事项）仍然有效。
>
> 面向 infra。数据侧见 [01-data-inventory.md](./01-data-inventory.md)。
> 本文的每一条「现状」都在 2026-09-08 对着 `nextmoe-infra` 的代码与生产库核对过，**不是照抄文档**——有两处文档与实现不一致，逐条标出。

## 1. 结论先行

1. 表情包面**天然属于 tier B**（GET-only 纯读小面），这正是 infra 08 §16.3 自己给的选择原则。
2. **tier B 的校验端点至今不存在**——文档写明它「随首个下游面一并交付」。表情包面就是那个首个下游面，所以这件事的成本落在本次立项里，躲不掉。
3. 三条已写进文档的接入路径，**今天没有一条是现成可用的**（§3 逐条核对）。infra 需要在其中挑一条落地，本站按选择配合。
4. 本站建议的取舍：**先交付 tier B + 一个只读面**，写路径（用 API 发布表情包）**不进 v1**。理由在 §6。

## 2. 这个面卖什么

不是「表情包列表」。是**「galgame 角色 → 可用表情素材」的索引**：498 张图挂在 128 个 catalog 人物身份上，95 部作品。下游真正会调的形状是：

```
给我 catalog 人物 7935（鸣濑白羽）的所有贴纸
给我 catalog 作品 4（夏日口袋）相关的表情包
搜 "鸣濑"，返回人物与包
```

这三个都以 **catalog id 为入口**，而不是以本站的 uuid 为入口。这一点决定了面的形状（§4），也是这个面值得进平台的唯一理由——它是 catalog 图谱在「素材」这一维上的延伸，不是又一个内容列表。

## 3. 现状核对：文档说的三条路，今天都不通

| 文档中的路径 | 出处 | 今天的实际状态 |
|---|---|---|
| **A · 进程内中间件**，import 共享 `devapi` | 08 §16.3 | ❌ **外仓 import 不了**。infra 的 module 名就是 `module api`，没有可被外部引用的 module path；`devapi` 包在 `apps/api/internal/` 下，Go 的 internal 规则也挡着 |
| **A' · 提取共享中间件包**（`kungal-kit`） | 03 §4.3、07 §14 | ❌ **仓库不存在**。两处文档都写「候选」「过渡期同构复制」 |
| **B · Traefik ForwardAuth** | 08 §16.3 | ❌ **校验端点不存在**。全仓搜 `ForwardAuth` 只有文档提及，Go 代码零实现 |
| **A'' · IdP introspection 端点** | 03 §4.3 给了完整契约 | ❌ **不存在**。`POST /oauth/apikey/introspect` 在代码里搜不到 |

**两处文档与实现不一致，建议一并修正：**

1. 03 §4.3 写「面服务**不直接读** `kun_galgame_infra`」，但 **catalog 今天就是直接读的**：`cmd/catalog/main.go` 里 `devapi.NewRepository(oauthDB)` + `devapi.NewMiddleware(repo, store)`，`ResolveByHash` 直接 `developer_api_keys JOIN oauth_clients`，Redis 只做缓存。同段末尾的「备选」其实已经是现状。
2. 03 §4.3 描述的 introspection 契约从未落地，Redis 键空间 `apikey:*`（07 §14）也就无从谈起——现在缓存的是 `devapi` 自己的 credential 缓存（TTL 60s，负缓存 10s）。

## 4. 建议的面形状

**face 名 `sticker`，scope `sticker:read`，路径 `api.nextmoe.dev/v1/sticker/*`。**
命名沿用 03 §4.2 词表规则（`<face>:read`）；计量 face 字符串 `sticker`，落 `developer_api_usage`。

> 📌 `sticker:read` **已不存在**。它在 2026-09-08 随 B 档交付上线，当天就被 infra 撤销（PR #168）：免费只读下游面收任意有效密钥，闸门只管身份、计量与限额。常量已删，铸键点名该串会被拒。见 [03 §8](./03-wiring.md)。

> 📌 本节的 `/v1/` 是提案当时按 doc 08 §16.2 写的。**已改判为 `/v2/sticker/*`**（infra PR #167，2026-09-08；`/v1` 已于 2026-08-27 整面退役）。下文所有路径按此顺读，实际契约以 [sticker-openapi.yaml](./sticker-openapi.yaml) 与 [03 §4.1](./03-wiring.md) 为准。

```
GET /v1/sticker/packs                     ?page&limit&sort=new|hot&q&tag&rating&work
GET /v1/sticker/packs/{packId}            含 stickers[]、characters[]、works[]
GET /v1/sticker/stickers/{stickerId}
GET /v1/sticker/characters/{catalogId}    ★ 以 catalog 人物 id 为入口，返回本站该人物的全部贴纸
GET /v1/sticker/characters/{catalogId}/stickers
GET /v1/sticker/works/{catalogId}/packs   ★ 以 catalog 作品 id 为入口
GET /v1/sticker/search                    ?q= 同时搜包与人物
GET /v1/sticker/tags
```

带 ★ 的两条是这个面存在的理由，其余是配套。全部 GET，全部只读，全部只暴露 `status = 1` 的包。

**分页**：沿用本站现有的 `{ items, total }` + `page/limit`（limit 上限 50）。若 infra 希望与 catalog `/v2` 的 keyset cursor 对齐，本站可以改——但请一次定死，不要两种并存。

**响应形状**：建议**不套本站的 `{code,message,data}` 信封**，与 catalog `/v2` 一致地裸 JSON + RFC 9457 problem 错误。理由：第三方开发者拿一把 key 横跨 catalog 与 sticker 两个面，两套错误语义是没必要的认知成本。**这需要本站写一层面专用的序列化**，我们认这个工作量。

**多语言**：名字字段原样返回四键 map + `und`（01 §3），不在服务端替调用方选语言。

## 5. 需要 infra 拍板的四件事

### 5.1 走哪一档（最重要）

| 选项 | infra 工作量 | 本站工作量 | 评价 |
|---|---|---|---|
| **B · ForwardAuth**（建议） | 交付校验+计量端点 + Traefik 中间件 | 几乎为零（只需把面挂上、按 §4 调形状） | 面是纯读小面，正是文档给 B 的适用场景；且这个端点迟早要给第二个下游面用，第一次做就是最划算的一次 |
| **A'' · introspection** | 落地 03 §4.3 那个已有契约的端点 | 写一个 client + Redis 缓存 + 限流计量 | 端点是内网 s2s，比 ForwardAuth 通用（非 Traefik 环境也能用），但每个下游要各写一遍中间件 |
| **A' · 抽 `kungal-kit`** | 把 `devapi` 抽成公开 module | import + 挂路由 | 长期最干净，但等于现在就要为「共享中间件」的版本管理负责 |
| **A''' · 直读 infra 库** | 只需给一个只读 DB grant | 同构复制 catalog 那 ~200 行 | **能最快跑起来**，且与 catalog 现状一致；但要接受一个下游站点持有 `kun_galgame_infra` 的连接，与 03 §4.3 的原则冲突 |

本站的偏好是 **B**，退而求其次是 **A''**。不建议 A'''——为一个面开一条跨库读，会成为下一个「文档说不这样、实现却这样」的条目。

### 5.2 图片 URL 怎么给

贴纸字节在 image service，跨全生态去重（01 §4）。开放面吐 CDN 直链意味着：

- 引用扩散到我们控制不到的地方；一旦某个 hash 因别处的引用生命周期被回收，第三方的图会碎；
- 也没有任何计量——图片流量不经过平台的 face 计量。

三个选项，请 infra 选：
1. **直接吐 CDN URL**（最简单，接受上述两点）；
2. **只吐 hash + 一份 URL 拼装规则**，让调用方自己拼（等价于 1，但把责任说清楚）；
3. **面上提供 `GET /v1/sticker/stickers/{id}/image` 代理**，走平台计量（最贵，但引用与计量都可控）。

本站倾向 **2**：hash 是稳定标识，拼装规则本来就是公开的，同时不假装我们能保证 CDN 的生命周期。

### 5.3 r18 怎么办

95 部关联作品里 **89 部是 r18**（01 §1）。但注意两件事分开：

- **作品**的 r18 是 catalog 的 `content_rating`；
- **贴纸**本身是安全的表情裁切，本站 7 套包的 `content_rating` 全是 0（全年龄）。

NSFW 能力位已于 2026-08-25 整档退役（03 §4.2），任何 key 都能取 nsfw 内容，`nsfw` 参数语义不变。所以建议：**面沿用同一套语义**——缺省不返回 `pack.content_rating = 1` 的包，显式 `nsfw=true` 才返回；而作品的 r18 只是随快照如实带出，不作为过滤条件。这样与 catalog 的行为逐字一致，下游不用记两套规则。

### 5.4 写路径的门

**建议 v1 不开写。** 用 API 发布表情包意味着：图片要经过 image service 的配额、内容要过审核、要有 per-user 而不是 per-app 的身份（授权码 + PKCE 代表某用户，03 §4.1）。这三件事任何一件没想清楚，开出来的就是一个刷图入口。等只读面跑稳、真有下游提出「我要代用户投稿」再单独立项。

## 6. 上线清单（给执行者）

按 07 §14 与既往两次教训整理。**Traefik router 那一步在生产上被漏过两次**（`/v1/store`、`/v1/playtime`），漏了的表现是 Traefik 404 而不是应用报错，所以单列。

- [x] ~~scope `sticker:read` 进词表（03 §4.2），自助集勾选~~ — **已作废**，见 03 §8
- [ ] face 字符串 `sticker` 进 `developer_api_usage`（注意 07 §14 的 face 列宽教训）
- [ ] **Traefik router：`/v1/sticker/` → sticker-api**（孪生 router 模式；⚠️ 最容易漏的一步）
- [ ] 若走 B：ForwardAuth 中间件挂在该 router 上，指向新交付的校验端点
- [ ] sticker-api 的 CORS allowlist 加 `api.nextmoe.dev`
- [ ] Cloudflare 公开读 Cache Rules
- [ ] OpenAPI 文档：本站产出合法 OpenAPI 3（非 Huma，08 §17.3 明确允许），进门户 docs-model + oasdiff 破坏门 + operation-count 守卫
- [ ] 在 kungal-docs 登记为对外 Tier-A 契约

## 7. 本站这边的待办（不依赖 infra 就能先做）

- [ ] 把 `characters/{catalogId}` 与 `works/{catalogId}/packs` 两条查询从站内 BFF 抽成面无关的 service 方法（现在耦合在 `Viewer` 上）
- [ ] 面专用序列化：裸 JSON + RFC 9457，与信封版并存
- [ ] 产出 OpenAPI 3 文档
- [ ] `works/{id}/packs` 走的是 `sticker_catalog_work_idx`，但 `pack_id` 不在索引里，每行要回一次堆。498 行时无所谓，包多了可以换成 `(catalog_work_id) INCLUDE (pack_id)` 拿 index-only scan——**列在这里是备忘，不是现在的问题**

## 8. 一条不属于本文但相关的建议

本站是 community 服务的第三个消费者（forum / letmoe / 本站），而 community 的 post 投影**不带任何 reaction 字段**，导致每个消费方都得自己建一张 like 镜像表各数各的。三家实现已经长得一模一样。如果 infra 有余力，在 community 的 post 投影上加 `reaction_count` + `viewer_reacted`（或一条批量 reaction 读接口）会一次性消掉三份重复代码——这跟开放 API 无关，但既然在盘点跨服务契约，一并提出。

> **已采纳（2026-09-16）**：community 在 infra #213 / #215 / #216 给读接口加了 `reaction_count` + `viewer_reacted`（按 `viewer_id` 填），开关也直接返回新计数。本站随即删掉了 `comment_like`（migration 000009）。
