# 开放 API · 立项材料

表情包站对外开放 API 的盘点与建议，供 **nextmoe-infra** 推进。

| # | 文件 | 内容 |
|---|---|---|
| 01 | [data-inventory.md](./01-data-inventory.md) | 库里有什么、字段含义、哪些能对外哪些不能、与其他 infra 服务的现状 |
| 02 | [opening-the-api.md](./02-opening-the-api.md) | 三条已写进 infra 文档的接入路径的**实际状态核对**、建议的面形状、需要 infra 拍板的四件事、上线清单 |
| 03 | [wiring.md](./03-wiring.md) | **接线现状**：infra 已交付什么、本站已交付什么、还差什么，含容器别名与端口 |
| 04 | [editor-packs.md](./04-editor-packs.md) | 编辑器贴纸选择器载荷的消费方契约：**存 hash 不要存 src**，以及官方贴纸只下架不删除 |
| — | [sticker-openapi.yaml](./sticker-openapi.yaml) | 面的 OpenAPI 3.1 契约（9 op，`redocly lint` 通过） |

## 三十秒版本

- 这个面卖的不是「表情包列表」，是 **catalog 人物身份 → 可用表情素材的索引**：498 张图挂在 128 个 catalog 人物上。入口是 catalog id，不是本站 uuid。
- 它是 **GET-only 纯读小面**，按 infra 08 §16.3 自己的原则属于 **tier B（Traefik ForwardAuth）**。
- tier B 的校验端点原本不存在，文档写明它「随首个下游面一并交付」——表情包面就是那个首个下游面，**infra 已于 2026-09-08 交付**（`96b689d0`，本站实测通过，见 03 §1）。当时三条路都不通的核对记录留在 02 §3。
- 顺带发现 **两处文档与实现不一致**（03 §4.3 说面服务不直接读 infra 库，catalog 今天就是直接读的；同节描述的 introspection 端点从未落地），建议一并修正。
- 本站建议：**先只读，写路径不进这一版**。公开前缀是 `api.nextmoe.dev/v2/sticker/*`（infra PR #167 改判，03 §4.1）。

## 状态

**已立项，已实现，已上线，等 oauth 重部署后复测。** infra 于 2026-09-08 交付了 B 档校验端点（`96b689d0`），本站同日交付了面实现与 OpenAPI 契约；同日 infra 两次复核，把下游面的公开前缀改判为 **`/v2/<site>/*`**（PR #167），又把 **scope 这一关整个撤掉**——免费只读面收任意有效密钥（PR #168，`sticker:read` 常量已删）。本仓已全面收敛到 `/v2/sticker` 且不再提 scope。Traefik 两条 face router 已在生产生效，冒烟走通了整条链路；剩下的是 infra 侧重部署 oauth 后复测 200，以及 spec 注册进门户，见 [03 §5/§8](./03-wiring.md)。

02 里「三条路今天都不通」的判断已被 B 档的交付取代，那一节留作记录。
