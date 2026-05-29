# 04 — 旧系统迁移计划

## 旧路径清单

| 站点 | 旧路径 | 类型 | 迁移策略 |
|------|--------|------|---------|
| kungal | `topic/user_${uid}/${userName}-${unixMS}.webp` | 内容型，markdown 硬编码 | ❌ **不迁移**，老桶只读永久保留 |
| kungal | `avatar/user_${uid}/avatar.webp` | 实体型，DB 可查 | ❌ **不迁移**（见下方"已压缩老图豁免"原则） |
| kungal | `avatar/user_${uid}/avatar-100.webp` | 派生图 | ❌ 不迁移（同上） |
| moyu | `topic/user_${uid}/${userName}-${unixMS}.webp` | 同 kungal | ❌ 不迁移 |
| moyu | `avatar/user_${uid}/avatar.webp` / `avatar.avif` | 实体型，DB 可查 | ❌ **不迁移**（同样豁免） |
| moyu | `avatar/user_${uid}/avatar-mini.webp` / `avatar-mini.avif` | 派生图 | ❌ 不迁移 |
| galgame wiki | `galgame/${gid}/banner/banner.webp` | 实体型，DB 可查 | ✅ **迁移** —— 需要新 `_mini` 460×259 变体 |
| galgame wiki | `galgame/${gid}/banner/banner-mini.webp` | 派生图 | ❌ 丢弃（新服务以 `_mini` 变体重新生成） |

### 已压缩老图豁免原则

kungal / moyu 的所有历史 avatar / topic 图都是**已经过压缩的 WebP / AVIF**（fit 1920×1080 / quality≈82 同量级），把它们再灌进新服务跑一遍 libvips 没收益：

- 不能"二次有损压缩"——只会让画质退化
- 调用方业务库本来就有 `users.avatar` URL 字段直接用
- 老桶继续公开可读托管，零成本（R2 几十 GB 几块钱/月）
- 新用户注册 / 用户改头像时**新写**才走 image_service，老用户保持原样

**这条原则不适用于 galgame wiki 的 banner**：wiki 的前端列表页要 460×259 的 `_mini` 变体，老桶里没有，必须走一次新服务把 `_mini` 生出来。所以 wiki banner 仍然要迁移。

### 变体命名差异（仅迁移 wiki banner 时相关）

新服务统一 `_<variant>` 后缀（`_mini` / `_256` / `_100`），不映射老命名（`banner-mini`），新服务按 preset 重新生成。

## 迁移原则

1. **新旧 URL 共存**：旧 URL 保持可访问，直到各站调用方代码切换完成
2. **不物理删除旧对象**：至少保留 6 个月，用于回滚和审计；topic markdown / kungal+moyu avatar 老桶永久保留
3. **不二次压缩**：已经是合理压缩 WebP / AVIF 的老图（kungal+moyu 全部图片）**直接不迁移**，业务库 `*_url_legacy` 字段直接当永久 URL 用
4. **增量可中断**：迁移的字段（如 wiki banner）脚本用 `WHERE *_image_hash IS NULL` 天然幂等，可中断重跑
5. **调用方代码切换各自独立**：kungal、moyu、galgame wiki 各自节奏，每站 3–4 个 PR 起步，1–2 个月落地
6. **派生图全部丢弃**：`avatar-100.webp`、`banner-mini.webp` 不映射老命名，由新服务预生成（仅对要迁移的字段适用）

## kungal / moyu 老图的特殊处理（含 topic + avatar）

kungal / moyu 的所有历史图片（topic 图床 + 用户头像）都按同一规则处理：

- 已经是合理压缩的 WebP / AVIF
- 老桶继续公开可读托管
- 业务库里的老 URL 字段（`users.avatar` / topic markdown 里的 hardcoded URL）保持指向老桶
- **新上传** 才走 image_service —— 新注册用户改头像 / 新发 topic 帖里的图

**最终方案**：

> 🔒 **kungal / moyu 老 avatar / topic 老 URL 永久保留只读，不迁移、不 rewrite、不二次压缩。新上传全部走新服务，新老数据自然分野。**

理由：
- 二次压缩对已是 WebP@82 的图只会让画质退化，没有任何收益
- R2 / S3 上几十 GB 的历史数据成本月度几块钱，不值得花工程时间折腾
- 不增加任何永久维护负担
- markdown 里的硬编码 URL 也不可能批量改写
- 新老 URL 用户不会同时看到（旧帖子看老 URL，新帖子看新 URL），无体验撕裂

## 阶段划分

### 阶段 0：新服务上线（V1 完成后）

- 图片服务独立运行于 `:9278`
- OAuth Client 为 kungal/moyu/galgame wiki 开通 `image:upload` scope + 对应 preset
- 各站点在**新功能**上先用新服务（新开模块、新注册用户的 avatar），旧数据不动
- 目标：验证新服务稳定性，收集真实流量数据

### 阶段 1：双写兼容期（1–2 周，按站点节奏）

- 旧代码保持不变，**旧 bucket 继续接收 topic 图床的上传**（直到该站 topic 切换，见阶段 4）
- avatar / banner 的新上传路径切到新服务
- 关键：确保新上传的图的 `hash` 被**同步写入调用方业务库**的新字段

例如 kungal 用户改头像：

```
// 旧代码（保留一段时间作为回退）
uploadToOldBucket(file) → 写 users.avatar_url_legacy

// 新代码
uploadToImageService(file) → 写 users.avatar_image_hash
```

前端读取优先 `avatar_image_hash`，缺失时回退到 `avatar_url_legacy`。

### 阶段 2：调用方侧迁移（仅对必须迁移的字段）

> 🔧 **平台侧不提供迁移脚本**。原因：迁移的本质是"翻转业务库的外键"，而**只有调用方知道哪个老 URL 属于哪个 entity**。平台再多扫桶 / 写 `images` 表也替代不了调用方那条 `UPDATE galgame SET banner_image_hash = ? WHERE id = ?`。让调用方直接走标准 `POST /image/upload` API 一步到位最简单。
>
> image_service 的 sha256 内容寻址 + 跨站去重已经保证：第二次以同样 bytes 调 `Upload` 是 dedup-hit（不重处理、不重存）。

#### 需要写迁移脚本的字段清单

按上一节"已压缩老图豁免原则"过滤后，三站合计**只剩极少数字段**需要写迁移脚本：

| 站点 | 字段 | 原因 |
|------|------|------|
| galgame wiki | `galgame.banner` | 列表页要 `_mini` 460×259 变体，老桶没有 |
| kungal | （无） | 全部豁免 |
| moyu | （无） | 全部豁免 |

> 如果未来某个字段决定上 V3 审核 / 跨站 dedup 等新特性，再单独把它加进迁移清单即可。骨架在下方，复用方便。

#### 迁移脚本骨架

#### 业务库 schema 准备

即便不迁移，**也要在业务库上加 `*_image_hash` 列**（用于新上传写入）；只有真要迁移的字段才需要 status / attempts 列。

**只加 hash 列（kungal `users.avatar` / moyu `users.avatar` 这种豁免迁移的字段）**：

```sql
ALTER TABLE "user"
    ADD COLUMN avatar_image_hash CHAR(64);

ALTER TABLE "user" RENAME COLUMN avatar TO avatar_url_legacy;
```

前端 resolve 函数：`avatar_image_hash` → 拼新 URL；NULL 时 fallback `avatar_url_legacy`。豁免字段永远走 fallback 分支，是 OK 的。

**加全套迁移列（galgame wiki `galgame.banner` 这种要迁移的字段）**：

```sql
ALTER TABLE galgame
    ADD COLUMN banner_image_hash CHAR(64),
    ADD COLUMN banner_migration_status   SMALLINT NOT NULL DEFAULT 0,  -- 0=未尝试 1=成功 2=永久失败
    ADD COLUMN banner_migration_attempts SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE galgame RENAME COLUMN banner TO banner_url_legacy;

CREATE INDEX idx_galgame_pending_migration
    ON galgame(id)
    WHERE banner_image_hash IS NULL
      AND banner_url_legacy IS NOT NULL
      AND banner_migration_status != 2;
```

#### 脚本骨架（以 galgame wiki `galgame.banner` 为例）

```go
// apps/api/cmd/migrate-galgame-banners-to-image-service/main.go

const maxAttempts = 3

func main() {
    cfg := mustLoadConfig()
    db := mustConnectDB(cfg)
    cli := imageclient.New(imageclient.Config{
        BaseURL:      cfg.ImageServiceBaseURL,
        ClientID:     cfg.ImageOAuthClientID,
        ClientSecret: cfg.ImageOAuthClientSecret,
    })
    ctx := context.Background()
    start := time.Now()

    var (
        processed, succeeded, failed int64
        migratedHashes               []string
    )

    rows := db.Query(`
        SELECT id, banner_url_legacy, banner_migration_attempts
        FROM galgame
        WHERE banner_image_hash IS NULL
          AND banner_url_legacy IS NOT NULL
          AND banner_migration_status != 2
        ORDER BY id ASC
    `)

    for rows.Next() {
        var gid int64
        var legacy string
        var attempts int16
        rows.Scan(&gid, &legacy, &attempts)
        processed++

        body, err := fetchOldObject(legacy)
        if err != nil {
            failed++
            recordFailure(db, gid, attempts, err)
            slog.Warn("fetch old", "gid", gid, "url", legacy, "err", err)
            goto progress
        }

        result, err := cli.Upload(ctx, bytes.NewReader(body), "banner.bin", "galgame_banner")
        if err != nil {
            failed++
            recordFailure(db, gid, attempts, err)
            // 配额耗尽时退出整个脚本（提示运维 raise 配额或换天再跑）
            if errors.Is(err, imageclient.ErrQuotaExceeded) {
                slog.Error("quota exceeded; stopping", "processed", processed)
                break
            }
            slog.Warn("upload", "gid", gid, "err", err)
            goto progress
        }

        _, err = db.Exec(`
            UPDATE galgame
               SET banner_image_hash = ?,
                   banner_migration_status = 1
             WHERE id = ?
        `, result.Hash, gid)
        if err != nil {
            slog.Error("update db", "gid", gid, "err", err)
            failed++
            goto progress
        }
        succeeded++
        migratedHashes = append(migratedHashes, result.Hash)
        slog.Debug("migrated", "gid", gid, "hash", result.Hash, "dedup", result.Deduplicated)

    progress:
        // 每 1000 行打一次 summary（可观测性）
        if processed%1000 == 0 {
            elapsed := time.Since(start)
            rate := float64(processed) / elapsed.Seconds()
            slog.Info("progress",
                "processed", processed,
                "succeeded", succeeded,
                "failed", failed,
                "rate_per_sec", fmt.Sprintf("%.1f", rate),
                "elapsed", elapsed.Truncate(time.Second),
            )
        }
    }

    slog.Info("migration finished",
        "processed", processed, "succeeded", succeeded, "failed", failed,
        "elapsed", time.Since(start).Truncate(time.Second),
    )

    // 收尾：把刚迁的 hash 主动 ping 一次。否则首次 ref-ping cron
    // 跑到之前，新写入的 last_referenced_at 就是 NOW()，没问题；
    // 但显式 ping 一次让"迁移日 = 首次 ping 日"对齐，便于事后追账。
    if len(migratedHashes) > 0 {
        for _, batch := range chunk(migratedHashes, 1000) {
            if _, err := cli.ReferencePing(ctx, batch); err != nil {
                slog.Warn("final ping", "err", err)
            }
        }
    }
}

// recordFailure 累加 attempts；超过 maxAttempts 标记为永久失败。
func recordFailure(db *sql.DB, gid int64, attempts int16, cause error) {
    newAttempts := attempts + 1
    status := 0
    if int(newAttempts) >= maxAttempts {
        status = 2 // 永久失败 — 后续 SELECT 用 status != 2 跳过
    }
    db.Exec(`
        UPDATE galgame
           SET banner_migration_attempts = ?,
               banner_migration_status = CASE WHEN ? = 2 THEN 2 ELSE banner_migration_status END
         WHERE id = ?
    `, newAttempts, status, gid)

    // 详细 error 写到独立 log 表，方便事后人工核查。
    db.Exec(`
        INSERT INTO image_migration_log(entity_id, entity_type, error_msg, attempted_at)
        VALUES (?, 'galgame.banner', ?, NOW())
    `, gid, cause.Error())
}
```

**骨架特性**：

- ✅ **死链跳过**：`banner_migration_status != 2` 把重复失败 ≥ 3 次的行排除；不会无限重试
- ✅ **断点续跑**：`banner_image_hash IS NULL` + 状态过滤是天然幂等谓词
- ✅ **配额超限自动退出**：`errors.Is(err, imageclient.ErrQuotaExceeded)` 命中就 break，避免在 429 上空转 1 万次
- ✅ **进度可观测**：每 1000 行 INFO 一行，含速率和 elapsed
- ✅ **收尾自动 ping**：迁移结束前把所有刚成功的 hash 显式 `ReferencePing` 一次
- ✅ **不删 `banner_url_legacy`**：保留作前端回退兜底（阶段 3）；半年观察期后再决定是否清字段
- ✅ **失败明细表**：`image_migration_log` 留事后排查（必要时人工 fix 后清 status 让脚本重试）

**预估耗时（galgame wiki banner 示例）**：

- 假设 ~5 万 galgame banner，平均每张 60KB → 3GB 总流量
- 受 image_service 流水线吞吐限制（含 decode / resize / encode WebP + `_mini` 变体）：约 50–200 张/秒
- 实际跑完：5 分钟 – 30 分钟；可分段、可中断

### 阶段 3：avatar URL 兼容层（可选，2–4 周）

**目的**：阶段 2 之后，业务库里 `users.avatar_image_hash` 已经有值，但可能还有：
- 浏览器缓存里的老 URL
- 第三方外链引用老 URL（很少）
- 部分未更新的前端代码

**方案（推荐最简单的）**：业务库保留 `avatar_url_legacy` 字段，前端 URL 解析函数：

```ts
function resolveAvatarUrl(user) {
  if (user.avatar_image_hash) return imageMainUrl(user.avatar_image_hash)
  if (user.avatar_url_legacy) return user.avatar_url_legacy
  return DEFAULT_AVATAR
}
```

前端代码全部切换完成后，可以删 `avatar_url_legacy` 字段（或永久保留也无妨，字段本身不占钱）。

> **不建议**在 CDN / Nginx 层写 rewrite 规则把 `/avatar/user_123/avatar.webp` 映射到新 URL——因为要永久维护一个"查业务库 → 拼 hash URL"的外部服务，复杂度远超收益。直接靠业务库字段回退就够。

### 阶段 4：业务代码切换（各站独立 1–2 个月）

每站每个图片字段独立一组 PR。以 kungal 为例（按"老图豁免"原则后实际只切 avatar 写入路径）：

| PR | 工作 | 耗时 |
|----|------|------|
| PR-1 | 业务库 migration：加 `users.avatar_image_hash`（豁免字段不需要 status / attempts 列）+ GORM model | 半天 |
| PR-2 | avatar 上传逻辑改调图片服务；上传成功同步写 `avatar_image_hash`（旧 URL 永远停留在 `avatar_url_legacy` 字段，老用户保持原样） | 1–2 天 |
| PR-3 | topic 图床上传逻辑改调图片服务（`preset=topic`）；老 markdown URL 永久保留指向老桶 | 1–2 天 |
| PR-4 | 前端 `resolveAvatarUrl(user)` 增加 fallback 链：`avatar_image_hash` → `avatar_url_legacy` → 默认头像 | 1–2 天 |

> 注意：**没有 PR-5 删 `avatar_url_legacy`**——老用户的头像永远只在这个字段里，删了就裂图。`avatar_url_legacy` 是永久字段，不是迁移期 transient 字段。

galgame wiki 类似但因为 banner 真要迁移，多一个 cmd：

| PR | 工作 | 耗时 |
|----|------|------|
| PR-1 | 加 `galgame.banner_image_hash` + `_migration_status` + `_migration_attempts` + GORM model | 半天 |
| PR-2 | banner 上传逻辑改调图片服务（`preset=galgame_banner`） | 1 天 |
| PR-3 | `cmd/migrate-galgame-banners-to-image-service` 离线迁移脚本（用上面的骨架） | 1 天 |
| PR-4 | 前端 banner 列表页改用 `_mini` 变体 URL；fallback 老 URL | 1 天 |

moyu 结构与 kungal 同（avatar 豁免，无 banner）。三站合计 7–9 个 PR，跨 1–2 个月。

#### 阶段 4.x 验收（每站独立）

- [ ] 监控显示旧 bucket 的**新上传** QPS 归零
- [ ] 监控显示 `users.avatar_image_hash` 在新注册 / 改头像后覆盖率 > 99%（老用户保持 NULL 是正常的）
- [ ] galgame wiki：`banner_image_hash` 覆盖率 > 99%
- [ ] 监控显示旧 URL 访问量稳定（**不必下降到 0**——老用户头像和 topic 历史图永远走老 URL）

### 阶段 5：旧对象生命周期管理

按"老图豁免"原则后，老桶不再有"下线时间表"——大多数都是永久保留。

#### kungal / moyu 老桶（topic + avatar）

- **永久保留只读，不进入下线议程**
- 业务库的 `avatar_url_legacy` 字段也是永久字段，不是 transient
- topic markdown 里硬编码的 URL 永远指向老桶
- 月度成本几块钱，没有任何收益促成下线

#### galgame wiki 老 banner 桶（这部分有迁移）

只有这个有真正的"老桶下线"议程：

- **最低保留 12 个月**
- **下线触发条件（必须同时满足）**：
  1. 老 URL 访问日志显示 **连续 30 天 < 1%** 流量比例
  2. `galgame.banner_image_hash` 覆盖率 > 99.5%
  3. 平台 + 调用方任一方都没有 veto
- **下线前 30 天预告**

#### 桶的物理对象不删

即便桶下线（停止公开访问），对象本身建议**继续保留**至少再 12 个月以备审计 / 回滚。最便宜的方式是转 IA / Glacier。

## 特殊情况处理

### galgame banner 的原图问题

galgame wiki 之前没有保留高清原图，只存了压缩版（与新服务的 `webp@82 fit 1920×1080` 大同小异）：

- 迁移后 `images.width/height` 用实际旧图尺寸
- `is_original = false`（实际上 V1 根本没这个字段，这里留作对比说明）
- 未来需要高清原图时，再增加新的 preset + 重新从 VNDB 采集

### 用户改名导致的 topic 图路径混淆

kungal 旧路径 `topic/user_123/alice-1700000000.webp`：`alice` 是上传当时的用户名。

由于 **topic 整体不迁移**，这个问题自然不存在。

### 重复内容的 hash 碰撞

迁移过程中会发现大量重复（同一张头像被不同用户传过）：

- `images` 以 `UNIQUE(hash)` 单行存在
- 迁移脚本遇到已有 hash：只 INSERT 一行 `image_site_usage`（或 UPDATE upload_count）+ 更新业务库外键
- 对象存储天然只有一份

## 回滚策略

迁移中任何阶段出错，回滚路径：

| 阶段 | 回滚动作 |
|------|---------|
| 1（双写期） | 停掉新代码的写入，旧代码自持续工作 |
| 2（批量迁移） | 调用方脚本失败行不影响已成功行（`avatar_image_hash IS NULL` 仍代表未迁，再跑就续上）；整体出错则停脚本观察 |
| 3（URL 回退） | 撤下前端读取的 "优先新 URL" 逻辑，切回 `avatar_url_legacy`；旧桶从未删过，直接可用 |
| 4（代码切换） | 调用方有 fallback 逻辑，撤回切换不丢数据 |

## 调用方 cron 清单（每站必备）

接入后，每个调用方**必须**在自己的后端部署一个每日 cron，否则 60 天后图片会被转冷存储。

### 需要实现的内容

```go
// 例：apps/api/internal/infrastructure/cron/image_ping.go

// 每日凌晨 3 点触发
c.AddFunc("0 3 * * *", func() {
    ctx := context.Background()

    // 1. 从业务库聚合所有 *_image_hash 非空字段
    hashes, err := collectAllReferencedHashes(db)
    if err != nil {
        slog.Error("collect hashes failed", "err", err)
        return
    }

    // 2. 按 1000 一批发到 image_service
    for _, batch := range chunkBy(hashes, 1000) {
        resp, err := imageClient.ReferencePing(ctx, batch)
        if err != nil {
            slog.Error("reference ping failed", "err", err)
            continue
        }

        // 3. not_found 的可以清自己的外键（可选，防止挂空引用）
        for _, h := range resp.NotFound {
            slog.Warn("image not found, clearing local ref", "hash", h)
            clearLocalRefsForHash(db, h)
        }
    }
})
```

### SQL 聚合模板（各调用方自行填字段）

```sql
-- kungal / moyu
SELECT DISTINCT avatar_image_hash FROM "user" WHERE avatar_image_hash IS NOT NULL
UNION
SELECT DISTINCT hash FROM unnest_topic_images_hashes() WHERE hash IS NOT NULL

-- galgame wiki
SELECT DISTINCT banner_image_hash FROM galgame WHERE banner_image_hash IS NOT NULL
UNION
SELECT DISTINCT cover_image_hash FROM galgame WHERE cover_image_hash IS NOT NULL
```

### 验收标准

- [ ] 每日 cron 上线后跑满 3 天无失败
- [ ] image_service 侧观察到目标站点的 `POST /image/reference-ping` 每日一次、hash 数合理
- [ ] `not_found` 返回数长期趋近 0（偶尔有是正常的，持续高说明本地库有挂空引用）

## 风险检查清单

- [ ] 旧 bucket 中 avatar / banner 总对象数统计完毕（用于进度条）
- [ ] 调用方业务库的 `*_image_hash` / `*_url_legacy` 字段 migration 已上线
- [ ] 图片服务能承接真实流量（V1 验收通过）
- [ ] `image_site_usage` 写入幂等（`ON CONFLICT DO UPDATE`）
- [ ] topic 图床的老桶公开可读配置不变（别意外改成私有）
- [ ] 回滚演练至少走过一次（至少在 dev 环境）

下一篇：[05 — 工程计划](./05-engineering-plan.md)
