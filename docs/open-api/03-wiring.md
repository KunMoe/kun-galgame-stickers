# 03 — 接线：infra 已交付的、本站已交付的、还差的

> 2026-09-08。本文取代 [02](./02-opening-the-api.md) 的「现状」一节——02 写于同日更早，那时 tier B 的校验端点还不存在。02 的建议部分仍然有效，**infra 选了其中的 B 档**。

## 1. infra 已交付（已核对，不是照抄提交说明）

`96b689d0 feat(devapi): B-tier ForwardAuth endpoint and the first two downstream faces (doc 08 §16.5)`

| 事项 | 状态 |
|---|---|
| `GET /internal/devapi/forward-auth?face=<name>` | ✅ 已上线。oauth 容器 2026-09-08T10:51:53Z 重建 |
| face 注册表 `moyu` / `sticker` | ✅ 手维护于 `devapi/forwardauth.go`，未注册 face 进不了计量表 |
| scope `sticker:read` / `moyu:read` | ~~✅ 常量、自助集、门户 picker、quickstart 文档一并落地~~ **当天即撤销**，见 §8 |
| Traefik 标签配方 | ✅ 记在 doc 08 §16.5，占位符待填 |
| 网关接线本身 | ⬜ 未做（本次交付明确不含） |

**两条验收判据我已在 dokploy-network 内侧实测通过**（从 sticker 的 web 容器直接打 `oauth:9277`）：

```
/internal/devapi/forward-auth?face=sticker  → 401 {"code":10001,"message":"未授权，请先登录"}
/internal/devapi/forward-auth?face=nope     → 500 {"code":3,"message":"unregistered forward-auth face"}
/internal/devapi/forward-auth（无 face）     → 500 同上
```

即 infra 清单里的第 1 项（部署）**已经完成**，不必再等。

## 2. infra 要的那个输入：别名与端口

生产实测（`docker inspect` 的 `NetworkSettings.Networks."dokploy-network".Aliases`，不是从 compose 推的）：

| face | 容器别名 | 端口 | 容器名 |
|---|---|---|---|
| `sticker` | **`sticker-api`** | **9421** | `kun-visual-novel-sticker-eaxaym-sticker-api-1` |
| `moyu` | **`moyu-api`** | **5214** | `kun-visual-novel-patch-ndxhtp-moyu-api-1` |
| forwardAuth 目标 | **`oauth`**（亦有 `infra-oauth`） | **9277** | `kun-visual-novel-infra-vqvqbc-oauth-1` |

**sticker 这一侧的标签我已经写进本仓 `docker-compose.prod.yml`**（`sticker-api` 服务的 `labels:`），照 §16.5 配方填好，只等面板点 Deploy。moyu 那一侧要 moyu 仓自己加，配方相同，把 `sticker` 换成 `moyu`、端口换 5214。

注意生产上 sticker 现有的两个 router 是 **Dokploy 面板生成的**（名字形如 `kun-visual-novel-sticker-eaxaym-44-websecure`），不在 compose 里；新加的 `labels:` 与它们共存，Deploy 后请确认两组都在。

## 3. 本站已交付

面已经写好并跑通了，不是「准备好了可以开始写」。

```
GET /v2/sticker/packs                              ?limit&cursor&include_total&sort&q&tag&work&official&linked&nsfw
GET /v2/sticker/packs/{pack_id}
GET /v2/sticker/stickers/{sticker_id}
GET /v2/sticker/characters                         ?limit&cursor&include_total&q&work
GET /v2/sticker/characters/{character_id}        ★
GET /v2/sticker/characters/{character_id}/stickers ★ ?limit&cursor&include_total
GET /v2/sticker/works                              ?limit&cursor&include_total&q
GET /v2/sticker/works/{work_id}/packs            ★ ?limit&cursor&include_total&sort&nsfw
GET /v2/sticker/tags                               ?limit&cursor&include_total
```

★ 是这个面存在的理由：以 catalog id 为入口。9 个 operation，全 GET，只暴露 `status = 1` 的包。

- **契约**：[sticker-openapi.yaml](./sticker-openapi.yaml)，OpenAPI 3.1，`redocly lint` 通过（08 §17.3 允许手写）。这就是 infra 清单第 3 项要的东西。
- **响应**：裸 JSON，信封与 catalog `/v2` 逐字段一致：`{object: "list", items, next_cursor?, total?}`；错误是 RFC 9457 `application/problem+json`，type 落在 infra 的 `problems/platform/` 域，错误码逐字取自 infra 的封闭注册表，第三方一套解码器通吃两个面。细节见 §9（2026-09-18 收敛）。
- **本站自己的 `/api/v1` 不受影响**，仍是 `{code,message,data}` 信封 + cookie 会话，两套错误语言井水不犯河水（`errorHandler` 按路径分流）。
- **缓存**：逐字沿用 catalog `/v2` 公开档的 `public, max-age=300, s-maxage=1800, stale-while-revalidate=3600` + ETag/304。
- **鉴权**：本服务一行都没写。B 档的全部意义就在这里。三个 `X-NextMoe-*` 头本站不读——这个面对谁都是同一个答案。
- **限流**：本地不挂。这个服务能看见的唯一 client address 是 Traefik 的，按 IP 限流等于把所有应用塞进一个额度。真正的 per-key 额度在网关。

## 4. 三件已裁决（infra PR #167，2026-09-08）

### 4.1 路径：裁决为 `/v2/sticker`

08 §16.2 / §16.5 原本写的是 `api.nextmoe.dev/v1/<site>/*`，写于 2026-07-23——当时 `/v1` 就是平台在产的公开前缀。但 wave R3 于 2026-08-27 把 `/v1` **整面退役**，六个前缀一律回 `410 Gone` 并 `Link: rel="successor-version"` 指向 `/v2`：

```
GET https://api.nextmoe.dev/v1/store/prices     → 410, Link: <https://api.nextmoe.dev/v2>; rel="successor-version"
GET https://api.nextmoe.dev/v1/playtime/works/1 → 410, 同上
GET https://api.nextmoe.dev/v2/catalog/works    → 401 application/problem+json（在产）
```

把两个全新的公开面钉回那个前缀，等于自己打脸门户上「v1 已整面退役」。infra 复核后改判 **`/v2/<site>/*`**，并写进 doc 08 §16.2/§16.5，注册表 `pathLabel`、测试、门户文档、配方一并改到 `/v2`。

同时查实了唯一的技术疑点：门户 docs-model 的 `FACES` **每项带自己的 spec 文件，oasdiff 也按文件逐个比**——闸门按文档切分，不按前缀，所以两份 spec 同时描述 `/v2/...` 不冲突。

本仓已收敛到单一前缀：`face.go` 的 `facePrefix = "/v2/sticker"`、compose 里两条 `PathPrefix`、spec 里 9 条路径，全部只剩 `/v2/sticker`。

### 4.2 preflight 过不了闸 — 采纳

ForwardAuth 对所有方法生效，浏览器的 `OPTIONS` 预检不带任何认证头，会被 401。结论是这个面在浏览器里直接 `fetch` 用不了，这是想要的（key 本来就不该进浏览器）。infra 已把这句写进门户 `authentication.md` 的 NOTE 与 doc 08。

本站的 CORS 已按闸门放行的前提配好（`Access-Control-Allow-Origin: *`、不带 credentials、`expose` 了限额头），真要放行 `OPTIONS` 时浏览器侧即刻可用。

### 4.3 CDN 命中不计量 — 接受少计

`public, s-maxage=1800` 意味着 Cloudflare 命中的请求不进 `developer_api_usage`。infra 接受这个少计（与 catalog `/v2` 公开档同一取舍），已记进 §16.5。要严格计量就去掉 `s-maxage`。

## 5. 上线清单

- [x] ~~scope `sticker:read` 进词表 + 自助集~~ — **已作废**：免费只读面不设 scope（§8）
- [x] face 字符串 `sticker` 进注册表 — infra 已做
- [x] 校验端点上线并验收 — infra 已做，本文 §1 实测
- [x] 面实现 + OpenAPI 3 契约 — 本站已做
- [x] 前缀裁决为 `/v2/sticker` — infra PR #167；本仓已全面收敛（§4.1）
- [x] **面板点 Deploy** — 2026-09-08 完成，两条 face router 已在 Traefik 生效（§7）
- [x] 冒烟：无 key → 401，真 key → 403 `missing required scope: sticker:read`（§7；该 403 已随 #168 消失）
- [x] ~~给某个 client / key 授予 `sticker:read`~~ — **需求消失**，任意有效 key 即可（§8）
- [ ] **oauth 重部署后复测 200** —— 这是 #168 唯一剩下的验收动作（§8）
- [ ] moyu 仓照同一配方加标签（`moyu-api`、5214、`face=moyu`）
- [ ] spec 注册进门户 docs-model + oasdiff 门 + operation-count 守卫（9 op）+ kungal-docs 登记
- [ ] #167 合并后 oauth 再部署一次，`pathLabel` 记账才是新值（纯计量，端点行为不变）——与 #168 同一次部署一并生效

## 6. Deploy 前查到的一件事：`/v2` 已有 catch-all router

改判到 `/v2` 之后我查了生产 Traefik 的路由表（`dokploy-traefik` 的 `/api/http/routers`，`api.nextmoe.dev` 上共 15 条），发现：

```
infra-v2-pub@docker   priority 44   Host(`api.nextmoe.dev`) && PathPrefix(`/v2`)
```

**`/v2/sticker/*` 今天已经能通，由 infra 自己的 `/v2` 服务兜底应答**（实测 `GET https://api.nextmoe.dev/v2/sticker/packs` → `404 application/problem+json`，`type` 是 `problems/platform/not-found`，带 `request_id`，且不需要 key）。两个后果：

1. **优先级**。Traefik 默认优先级就是 rule 字符串长度：我们的 `PathPrefix(/v2/sticker)` 是 52，catch-all 是 44，所以本来就赢——但只赢在「多 8 个字符」上，infra 哪天给 `/v2` 的 rule 加个条件就可能反超并静默吃掉这个面。所以本仓的两条 router 都显式写了 `priority: '100'`。
2. **冒烟测试的判读**。「router 漏挂」在这里**不表现为裸 Traefik 404**，而是 infra 的 problem 文档。~~看 `type` 就能三选一~~——§9 之后本面自己的 problem 也是 `problems/platform/*` 且带 `request_id`，`type` 已分不出是谁答的。改看**状态码**，并且只拿 `/v2/sticker/packs` 这条对任何有效 key 都回 200 的路由探：
   - `404`（`problems/platform/not-found`）→ 标签没生效，请求还在走 catch-all
   - ForwardAuth 的 401 → 标签生效了，key/scope 的问题
   - 正常 JSON（`{"object":"list", …}`）→ 通了

> ⚠️ Traefik router 这一步在生产上被漏过两次（`/v1/store`、`/v1/playtime`）。Deploy 后第一件事就是拿真 key 打一次，按上面三条判读。

## 7. Deploy 后实测（2026-09-08）

Traefik 侧（`dokploy-traefik` 的 `/api/http/routers`）：

```
enabled 100 sticker-face-pub@docker       Host(`api.nextmoe.dev`) && PathPrefix(`/v2/sticker`)  mw: sticker-face-forwardauth@docker
enabled 100 sticker-face-pub-http@docker  同上                                                   mw: redirect-to-https@file
enabled  48 kun-visual-novel-sticker-eaxaym-44-web{,secure}@docker   Host(`sticker.kungal.com`) && PathPrefix(`/api`)
enabled  26 kun-visual-novel-sticker-eaxaym-12-web{,secure}@docker   Host(`sticker.kungal.com`)
```

面板生成的四条 router 一条没少，新加的两条优先级 100 稳压 `infra-v2-pub`（44）。

请求侧：

| 探测 | 结果 |
|---|---|
| `GET /v2/sticker/packs`（无 key） | `401 {"code":10001,…}` —— ForwardAuth 拦住，不再是 catch-all 的 404 |
| `GET /v2/sticker/packs`（真 `nmk_live_` key，仅 `catalog:read`） | **`403 {"code":5,"message":"missing required scope: sticker:read"}`**（当日实测；#168 之后应为 200，见 §8） |
| 同一把 key 打 `/v2/catalog/works` | `200` —— key 本身健康，403 是 scope 判定不是 key 失效 |
| `http://api.nextmoe.dev/v2/sticker/packs` | `301` → https |
| 容器内 `sticker-api:9421/v2/sticker/packs` | `200 {"object":"list",…}` —— 镜像带着面 |
| `sticker.kungal.com` 首页与 `/api/v1/*` | `200`，站内不受影响 |

那句 403 是**整条链路打通的证据**：Traefik 命中了我们的 router → forwardAuth 打到 oauth → oauth 认出 face `sticker`、校验了 key、按注册表比对 scope 后拒绝。

> ⚠️ **本节以下的判断已被 §8 取代。** 当时的结论是「唯一还差的一步：没有任何 key 持有 `sticker:read`」——全库 50 把 key 的 scope 只有 `catalog:read` / `store:read` / `galgame:*` / `claim_events:read`，表情包自己那个 client（`c5cd7b07…`）的 `allowed_scopes` 里也没有它，所以要拿 200 就得先去门户勾一个 scope。infra 当天复核后没有选择「补发 scope」，而是**把 scope 这一关整个撤掉**（PR #168）。这个 403 因此不是一个待补的配置，而是一段已被删除的代码。

生产数据现状：7 个已发布包、498 张贴纸、**0 个标签**（所以 `/v2/sticker/tags` 会诚实地回空列表，不是 bug）。

## 8. 免 scope 改判（infra PR #168，2026-09-08）

§7 那个 403 触发了 infra 的复核，结论不是「给谁补一个 scope」，而是**撤掉这一档面的 scope 检查**：

> 免费只读的 B 档下游面收**任意有效密钥**。闸门在这类面上的职责是身份、计量与限额，不是授权。

落地形态（`9d843a05`，已并入 `main`，CI 全绿）：

- `forwardauth.go` 的注册表从 `face → {scope, pathLabel}` 退化为 `face → pathLabel`，`HasScope` 那一段连同它记的那条 403 计量**整段删除**。
- `ScopeMoyuRead` / `ScopeStickerRead` **两个常量直接删掉**，不是留着不查——生产核实过零把密钥曾铸入任一，所以撤得干净。自助集回到 `catalog:read` / `store:read` 两项。
- 铸键时仍然点名这两串的请求（比如缓存了旧门户构建的页面）拿到的是 `ErrScopeNotAllowed`，不是静默成功。
- doc 08 §16.2/§16.5 记录改判：「加一张脸 = 注册表一行」，要授权的敏感面另立审批 scope 或走 A 档，**不复用这条免 scope 通道**。

**闸门在这个面上现在只会回三种非 2xx**：401（无/坏密钥）、429（超限）、500（face 未注册——那是接线错误不是客户端错误）。**403 不再存在。**

### 本站因此要改的

只有文档与注释——面的代码一行没动，因为它本来就不读 scope，B 档的全部意义就在这里。已改：

| 位置 | 改动 |
|---|---|
| `sticker-openapi.yaml` | `info.description` 去掉 scope 声明并写明「不要在铸键时勾 `sticker:read`，那个串已经不存在，会被拒」；`securitySchemes.apiKey.description` 改为「任意有效 key」；顶部的 401/403/429 三件套去掉 403。`redocly lint` 仍通过 |
| `apps/api/internal/app/face.go` | 面的头注释去掉 scope，并记下这条改判与它的理由 |
| `handler/face.go` | 「已被 keyed, scoped and metered」→ `budgeted` |
| `docker-compose.prod.yml` | forwardAuth 注释不再说「查 `sticker:read`」（**注释改动，标签一字未动，不需要重新 Deploy**） |
| 本文 §1 / §5 / §7 | 标注作废 |

`info.version` 保持 `1.0.0`：这份 spec 还没注册进门户，没有任何第三方见过 1.0.0，也就没有需要 oasdiff 比对的基线——**首次发布的那一版应当直接就是现在这个事实**。

### 还差的验收

oauth 重部署（#167 的 `pathLabel` 与 #168 的免 scope 同一次生效）之后，拿任意一把健康的 `nmk_live_` key 打：

```
GET https://api.nextmoe.dev/v2/sticker/packs
Authorization: Bearer nmk_live_...
→ 期望 200 {"object":"list","items":[…7 个包…]}（§9 之后的信封；7 个包不满一页，所以没有 next_cursor）
```

三条判读照 §6（已按 §9 修订）：`/v2/sticker/packs` 回 404 = 标签没生效；401 = key 的问题；200 = 通了。

顺带那句「确认三个 `X-NextMoe-*` 头有没有被 Traefik 转给 sticker」，**从外部观察不到**——本站不读也不回显这三个头（`face.go` 的注释就写着「every answer here is the same for every caller」）。能查的是接线本身：`dokploy-traefik` 的 `/api/http/middlewares` 里 `sticker-face-forwardauth@docker` 的 `authResponseHeaders` 应为 `X-NextMoe-Client-Id,X-NextMoe-Key-Id,X-NextMoe-Tier`（compose 里就是这么写的，Deploy 已生效）。真要看到头本身落地，得临时加一条回显——本站不打算为此改代码。

## 9. 与 catalog `/v2` 契约收敛（2026-09-18）

infra 复核门户上的两份下游 spec 后提出的清单，本站全部落地。**这是破坏性变更**，`info.version` 因此 `1.0.0 → 2.0.0`——与 §8 不同，这一次门户上已经有了基线（infra `c7e91a36` 把 1.0.0 登记进了 `apps/developer/public/specs/`）。

| 项 | 之前 | 之后 |
|---|---|---|
| catalog / 账号 id | `Work.id` / `Character.id` / `Author.id` 是 number | 十进制 string，spec 声明 `type: string, pattern: '^[0-9]+$'`；`work_id` / `character_id` 路径参数与 `work` 查询参数同样声明（线上格式不变） |
| 分页 | `page`（1–1000）+ `limit`（1–50） | 不透明 `cursor` + `limit`（1–100，默认 20）；底下仍是 offset，编码与 infra `collect.EncodeOffset` 逐字节一致（`cur_` + 无填充 base64url），moyu 同款 |
| 列表信封 | `{object, items, total, page, limit}` | `{object, items, next_cursor?, total?}`：`next_cursor` 末页**省略**（不是 null），`total` 仅 `include_total=true` 时出现 |
| 坏 cursor | — | `400 INVALID_CURSOR` |
| problem `type` | `problems/sticker/<slug>`（门户上不存在的页） | `problems/platform/<slug>` |
| problem 成员 | `detail` 可缺；无 `request_id`、无 `errors` | `detail`、`request_id`（= `X-Request-ID` 响应头）、`errors`（非字段错误时为 `[]`）恒在；每个 400 的 `errors[0].parameter` 指出要改的参数 |
| spec 声明 | 没写缓存 | 每个 200 声明 `ETag` / `Cache-Control`，每个 operation 声明 `304`（代码本来就在发） |

顺手改掉的三处，都是上面这些改动逼出来的：

- **`/tags` 也能翻页了**。以前它只是「前 N 个」，信封里 `total = len(items)`；新信封下缺 `next_cursor` 就等于宣称「已经是末页」，标签一多就是假话。
- **`work=abc` 从静默忽略改为 400**。spec 既然声明了 `pattern`，服务端再把不合规的值当成「没传」、回一份未过滤的列表，就是在自相矛盾。
- **problem 一律 `Cache-Control: no-store`**，并把 5xx 连同 `request_id` 记进日志。面上的公共缓存策略是给「对谁都一样」的答案用的；problem 带着本次请求的 id，被边缘缓存的话就会把同一个 id 发给许多调用方。另外 `POST /v2/sticker/*` 以前落进 `INTERNAL_ERROR 500`，现在是注册表里的 `METHOD_NOT_ALLOWED 405`。

**infra 侧还要做一步**：门户上的副本（`nextmoe-infra/apps/developer/public/specs/sticker-openapi.yaml`）仍是 1.0.0，需要从本仓重新同步；oasdiff 会把这次判为 breaking，这是预期内的。

