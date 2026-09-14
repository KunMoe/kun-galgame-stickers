# `GET /api/v1/editor-packs` — 编辑器贴纸选择器的载荷

不需要密钥、可缓存、只含官方已发布包。一次响应给出全部官方包的全部贴纸，形状对齐
`@kungal/editor-core` 的 `StickerPack` / `StickerItem`，消费方可以直接丢给自己的
`stickerSource` 适配器。

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "variant": "320",
    "packs": [
      {
        "name": "鲲 Galgame 表情包 [1]",
        "stickers": [
          {
            "src": "https://image.kungal.iloveren.link/d5/2a/d52ac5fb…e8_320.webp",
            "name": "鲲 Galgame 表情包 [1] - 1",
            "hash": "d52ac5fb…e8"
          }
        ]
      }
    ]
  }
}
```

## 存 `hash`，不要存 `src`

`src` 是本次部署的图床当前域名下的一个 URL。它存在的唯一目的是让选择器把缩略图画出来。

**消费方一旦把 `src` 写进帖子正文，就把这个域名焊死进了它以后写的每一篇帖子。** 这不是假设：
论坛就是这么干的，等上一代贴纸域名停止发静态文件的那天，**215 张贴纸、1620 处引用同时变成裂图**，
而且唯一的回滚材料是从一张改过名的 legacy 表里手工还原出来的「位置 → hash」对照表。

所以：**持久化 `hash`，把 `variant` 一起存着，渲染时再用当时配置的 base 拼 URL。** URL 形状是图床的，
见 `docs/image_service/`，两级 hex 分片：

```
{cdn_base}/{hash[0:2]}/{hash[2:4]}/{hash}_{variant}.webp
```

论坛现在把它存成不含域名的 `/image/{hash}_{variant}` token，在自己的 markdown 渲染器里展开 ——
这就是可以照抄的形状。

## 官方贴纸只下架，永不删除

`ON DELETE CASCADE` 叠上图床的 reference-ping TTL，意味着这边一次硬删会**追溯性地**弄坏所有消费方
用过这张图的帖子。`apps/api/internal/platform/sticker/service/pack.go` 的 `refuseOfficialDelete`
在 `is_official` 为真时同时挡住删贴纸和删包两条路。要让一个包退出流通，把 `status` 改回
`PackDraft`：选择器不再提供它，而行和 reference ping 都还在。
