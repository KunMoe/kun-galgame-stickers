# Image Service 设计文档

这里是 **kun-oauth-admin** 即将新增的集中式图片服务（临时代号 `image_service`）的设计与工程文档。服务目标是替代 kungal / moyu / galgame wiki 三个站点各自独立的图片处理逻辑，成为一个通用的 "hash-addressed blob store + 元数据" 平台。

## 文档索引

| #   | 文件                                                      | 内容                                                                           |
| --- | --------------------------------------------------------- | ------------------------------------------------------------------------------ |
| 01  | [design.md](./01-design.md)                               | 背景、目标、服务边界、核心架构决策、关键权衡                                   |
| 02  | [storage-and-schema.md](./02-storage-and-schema.md)       | 对象存储路径设计、数据库 schema、站点配置、变体命名                            |
| 03  | [api-design.md](./03-api-design.md)                       | 对外 API 接口规范（上传、元信息、reference-ping）                              |
| 04  | [migration-plan.md](./04-migration-plan.md)               | 从三个旧站点迁移到新服务的分阶段计划                                           |
| 05  | [engineering-plan.md](./05-engineering-plan.md)           | 工程里程碑 V1–V4、交付物、验收标准、本地 dev 环境                              |
| 06  | [integration-guide.md](./06-integration-guide.md)         | **调用方视角**：OAuth 注册、业务库 migration、SDK、降级策略、cron              |
| 07  | [reopen-and-cgo-deploy.md](./07-reopen-and-cgo-deploy.md) | **运维 checklist**：重新开放上传、CGO/libwebp 部署、质量验证、回滚             |
| 08  | [integration-status.md](./08-integration-status.md)       | **接入现状快照**（2026-05-26）：哪些项目接了 / 没接 / 接得不完整，附 file:line |

## 一句话总结

> **图片服务不管业务实体、不存引用关系、不允许调用方主动删图。只管 hash 对应的二进制 + 元数据。上传时预生成固定变体；生命周期由 TTL 驱动。**

## 关键决策速查

- ✅ **决策 0：调用方不主动删图** —— 图片生命周期完全由 `last_referenced_at` + TTL 驱动，调用方只改自己库里的外键
- ✅ **复用 OAuth** —— 不新增 API Key 体系，沿用 `oauth_client` 作为"站点"的 source of truth
- ✅ **内容寻址** —— 存储 key = `sha256(content)`，无 site 前缀，跨站彻底物理去重
- ✅ **`UNIQUE(hash)` 单行** —— 物理 + 审核态都是单行；站点维度用独立 `image_site_usage` 审计表
- ✅ **调用方管引用** —— `users.avatar_image_hash` 放在各调用方库里，不在图片服务
- ✅ **上传时预生成固定变体** —— 按 preset 生成已知变体（avatar-100、banner-mini 等），不走 imgproxy
- ✅ **软清理** —— 靠 `last_referenced_at` + TTL，不用引用计数
- 🕒 **审核 + Admin UI 延后到 V3** —— V1 不做，`review_status` 列保留默认 `approved`

## V1 必要性下限（不可拆）

V1 上线**必须**包含以下一揽子，少一块就会翻车：

1. OAuth Client Credentials 鉴权
2. sha256 内容寻址 + `images` 表 + `image_site_usage` 审计
3. 压缩管线：`webp@82` / `fit 1920×1080` / strip EXIF
4. 预生成固定变体（总共 6 个：avatar×3（含 main/256/100）、banner×2、topic×1）
5. Redis day-window 配额限制
6. `POST /image/reference-ping`

> 缺压缩 + 变体 → 前端拿到 5MB 原图直接渲染会废
> 缺配额 → 上线第一周会被 4K 截图刷爆 R2

## 非目标

- ❌ 不是通用 CDN / 文件仓库（只接图片，不接视频、PDF、任意文件）
- ❌ 不做图片编辑器（裁剪、滤镜、水印等由调用方自行处理后再上传）
- ❌ 不做图床（不对外公开上传接口，仅服务已注册的 OAuth Client）
- ❌ 不支持调用方主动删图（唯一删除路径是 TTL 自然消亡）
