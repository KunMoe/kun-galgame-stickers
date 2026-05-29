> [📖 文档索引](./README.md) · 上一节：[04 — 分类轴](./04-taxonomy.md) · 下一节：[06 — 管理统计](./06-admin.md)

## 搜索 (Search)

由 **Meilisearch** 驱动的搜索，对应 3 个 index：

| Index uid | 对应实体 | 文档数级别 |
|-----------|---------|-----------|
| `galgames` | Galgame | ~60k |
| `galgame_tags` | Tag | ~3k |
| `galgame_officials` | Official | ~22k |

`/tag/search` 和 `/official/search` 也走这套 —— 本节的能力（typo 容错、高亮、facet 聚合、CJK 分词）对它们同样适用。

**共同特性**：
- CJK 分词由 Meilisearch Charabia 原生支持（中/日/英/繁中混合友好）
- Typo 容错：4 字以上 1 个 typo，8 字以上 2 个 typo；`vndb_id` 禁用 typo（必须精确）
- 响应时间通常 <20ms
- 所有 GET，**无需认证**

### GET /galgame/search

Galgame 全文搜索 + 多条件过滤。

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| q | string | `""` | 搜索词；空时仅按 filter 返回 |
| status | int (csv) | — | 不传即不过滤；可传 `0` / `0,1,2` 等 |
| content_limit | `sfw` \| `nsfw` \| `all` | **`sfw`** | NSFW 过滤；**省略 = sfw**（safe-by-default）。`all` = 不过滤（显式 opt-in 含 NSFW）。详见 [00-handbook §NSFW](./00-handbook-for-downstream.md#nsfw-content_limit-协议) |
| age_limit | `all` \| `r18` | — | 可选；当前与 `content_limit` 语义重复，仅作为元数据回传，wiki 端不做过滤 |
| original_language | string (csv) | — | `ja-jp,en-us` 等；逗号分隔 OR |
| tag_ids | int (csv) | — | AND：galgame 必须同时命中所有 tag |
| official_ids | int (csv) | — | 同上 |
| engine_ids | int (csv) | — | 同上 |
| series_id | int | — | 精确 |
| released_from | string | — | 发售日期下限（含）。`YYYY`（按年，向后兼容）或 `YYYY-MM`（按月）。详见 [00-handbook §17 日期筛选](./00-handbook-for-downstream.md#17-发售日期-release_date-筛选协议) |
| released_to | string | — | 发售日期上限（含）。同上格式。 |
| released_months | int (csv) | — | 不连续月份集合（1–12），叠加在年份区间上的 AND 过滤。如 `3,7,12`。详见 [00-handbook §17.10](./00-handbook-for-downstream.md#1710-不连续月份筛选released_months2026-05-28-新增) |
| include_intro | bool | `false` | `true` 时把 `intro_*` 四语言简介也纳入搜索 |
| include_pending | bool | `false` | `true` 且带 Bearer JWT 时，响应额外包含 `pending` 数组，列出当前用户的 status ∈ {3,4} 命中条目。详见 [07 — 投稿](./07-submission.md#get-galgamesearch-增量参数) |
| sort | string | `relevance` | `relevance` / `released_desc` / `released_asc` / `view` / `updated` |
| page | int | 1 | 1-based |
| limit | int | 24 | max 100 |
| facets | bool | `true` | 是否返回 facet 聚合（`age_limit`, `original_language`）|
| highlight | bool | `true` | 是否返回高亮片段 |
| fields | string (csv) | — | 字段投影：只返回指定字段，省带宽。空 = 返回索引内所有字段 |

**`fields` 参数（字段投影）**：

galgame 索引文档约 30 字段，其中 `intro_zh_cn` / `_ja_jp` / `_en_us` / `_zh_tw` 是 markdown，**单条可能 1–10 KB**。一次默认返回 24 条 → 24 × 4 = 96 个 intro 字段，可能 **几百 KB 浪费**。

列表页通常只需要 5–7 个字段，建议显式指定：

```
GET /galgame/search?q=fate&fields=id,vndb_id,name_zh_cn,name_ja_jp,name_en_us,banner,effective_banner_hash,view,release_date,release_date_tba
```

详情页可以省略 `fields` 拿全字段；或者显式列出需要的：

```
GET /galgame/search?q=fate&fields=id,name_zh_cn,intro_zh_cn,tag_names,official_names
```

可投影的字段就是索引文档的 [Galgame schema 全部字段](./01-galgame.md#get-galgamegid)（`id`, `vndb_id`, `name_*`, `banner`, `effective_banner_hash`, `intro_*`, `content_limit`, `age_limit`, `original_language`, `view`, `release_date`, `release_date_tba`, `released_year`, `released_month`, `released_ts`, `tag_ids`, `tag_names`, `official_ids`, `official_names`, `engine_ids`, `engine_names`, `series_id`, `created_ts`, `updated_ts` 等）。

> 注：本字段只控制**返回**的 attributes，不影响搜索 / 过滤范围。比如 `fields=id` 仍然会按全文匹配（不会因为 `name_*` 没在 `fields` 里就跳过 name 匹配）。

**响应示例**：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "items": [
      {
        "id": 1142,
        "vndb_id": "v30132",
        "name_zh_cn": "...",
        "name_ja_jp": "...",
        "name_en_us": "Fate/Empire of Dirt",
        "banner": "https://...",
        "status": 0,
        "content_limit": "sfw",
        "age_limit": "r18",
        "original_language": "ja-jp",
        "release_date": "2020-05-15",
        "release_date_tba": false,
        "effective_banner_hash": "abcd1234...ef",
        "tag_ids": [1, 2, 5],
        "official_ids": [7],
        "_formatted": {
          "name_en_us": "<mark>Fate</mark>/Empire of Dirt"
        }
      }
    ],
    "total": 182,
    "facets": {
      "age_limit": {"all": 1, "r18": 181},
      "original_language": {"ja-jp": 61, "en-us": 101, "zh-cn": 7}
    },
    "processing_time_ms": 2
  }
}
```

**行为说明**：
- `tag_ids / official_ids / engine_ids`：多值 **AND**（必须同时命中）
- `status / original_language`：多值 **OR**
- `q` 空 + 无 `sort`：返回 `updated_ts` 倒序的近期条目
- `q` 非空 + 无 `sort`：按相关度排序，相同分时 `view` 倒序
- `include_intro=true` 会把简介纳入搜索，召回扩大但可能引入噪声（简介里随口提到某个词的 VN 也会被命中）
- highlight 片段字段包含 `_formatted`，仅在命中的字段上出现

### GET /tag/search

见上方 [标签 (Tag)](#标签-tag) 章节的 `/tag/search`。

### GET /official/search

见上方 [开发商 (Official)](#开发商-official) 章节的 `/official/search`。

---


---

下一节：[06 — 管理统计](./06-admin.md)
