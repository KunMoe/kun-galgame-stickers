# Galgame Wiki 接入指南

本文档面向需要接入 Galgame Wiki 服务的站点后端（如 kungal、moyu），提供完整的对接方式。

## 1. 架构

```
站点前端 (Nuxt)
    │
    ├── 站点后端 (Go Fiber)
    │     ├── 本地交互表 (like/comment/rating/resource/galgame_stats)
    │     └── GalgameClient → 调用 Galgame Wiki Service
    │
    └── Galgame Wiki Service (独立端口 :9280)
          ├── kun_galgame_wiki DB (元数据)
          └── kun_oauth_admin DB (用户信息，只读)
```

站点后端有两类操作：
- **元数据操作**（创建/编辑/PR/标签等）→ 调 Galgame Wiki Service
- **交互操作**（点赞/评论/评分/资源）→ 操作本地数据库，不需要调 Wiki Service

## 2. GalgameClient 实现

站点后端通过 HTTP 调用 Wiki Service。推荐封装一个 client：

```go
type GalgameClient struct {
    baseURL    string // e.g. "http://127.0.0.1:9280/api"
    httpClient *http.Client
}

func NewGalgameClient(baseURL string) *GalgameClient {
    return &GalgameClient{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}
```

### 读操作（不需要认证）

```go
func (c *GalgameClient) GetDetail(ctx context.Context, gid int) (*GalgameDetail, error) {
    resp, err := c.httpClient.Get(fmt.Sprintf("%s/galgame/%d", c.baseURL, gid))
    // 解析 {code, data} 响应...
}

func (c *GalgameClient) List(ctx context.Context, page, limit int, search string) (*ListResult, error) {
    url := fmt.Sprintf("%s/galgame?page=%d&limit=%d&search=%s", c.baseURL, page, limit, url.QueryEscape(search))
    // ...
}
```

### 写操作（透传用户的 access_token）

```go
func (c *GalgameClient) Create(ctx context.Context, accessToken string, body CreateRequest) (*Galgame, error) {
    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/galgame", toJSON(body))
    req.Header.Set("Authorization", "Bearer "+accessToken)
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.httpClient.Do(req)
    // ...
}
```

**关键**：写操作直接透传用户的 OAuth access_token。Wiki Service 自己验证 JWT 并提取 uid，站点后端不需要做任何用户身份转换。

### 错误处理

```go
func (c *GalgameClient) handleResponse(resp *http.Response, target any) error {
    var result struct {
        Code    int             `json:"code"`
        Message string          `json:"message"`
        Data    json.RawMessage `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    if result.Code != 0 {
        // 直接透传 Wiki Service 的 code + message 给前端
        return &AppError{Code: result.Code, Message: result.Message}
    }

    return json.Unmarshal(result.Data, target)
}
```

## 3. 典型对接模式

### 3.1 Galgame 详情页

并行查询 Wiki Service（元数据）和本地数据库（互动数据）：

```go
func (h *GalgameHandler) GetDetail(c *fiber.Ctx) error {
    gid := c.Params("gid")
    userID := getOptionalUserID(c)

    g, _ := errgroup.WithContext(c.Context())

    var meta *GalgameDetail
    var stats *GalgameStats
    var interaction *UserInteraction

    g.Go(func() error {
        var err error
        meta, err = h.galgameClient.GetDetail(ctx, gid)
        return err
    })
    g.Go(func() error {
        var err error
        stats, err = h.localRepo.GetStats(ctx, gid)
        return err
    })
    if userID > 0 {
        g.Go(func() error {
            var err error
            interaction, err = h.localRepo.GetUserInteraction(ctx, gid, userID)
            return err
        })
    }

    if err := g.Wait(); err != nil {
        return response.Error(c, err)
    }

    return response.OK(c, mergeResponse(meta, stats, interaction))
}
```

### 3.2 创建 Galgame

先调 Wiki Service，成功后在本地处理副作用：

```go
func (h *GalgameHandler) Create(c *fiber.Ctx) error {
    accessToken := getAccessToken(c)

    // 1. 调 Wiki Service 创建元数据
    result, err := h.galgameClient.Create(ctx, accessToken, body)
    if err != nil {
        return response.Error(c, err) // 透传 Wiki Service 错误
    }

    // 2. 本地副作用（事务）
    h.db.Transaction(func(tx *gorm.DB) error {
        tx.Create(&GalgameStats{GalgameID: result.ID})
        tx.Model(&User{}).Where("id = ?", userID).
            Update("moemoepoint", gorm.Expr("moemoepoint + 3"))
        return nil
    })

    return response.OK(c, result)
}
```

### 3.3 点赞（纯本地操作）

不需要调 Wiki Service：

```go
func (h *GalgameHandler) ToggleLike(c *fiber.Ctx) error {
    // 直接操作本地 galgame_like + galgame_stats 表
}
```

### 3.4 版本历史和 PR

直接转发到 Wiki Service 或让前端直接调 Wiki Service：

```go
// 方案 A: 站点后端转发
func (h *GalgameHandler) ListRevisions(c *fiber.Ctx) error {
    gid := c.Params("gid")
    result, err := h.galgameClient.ListRevisions(ctx, gid, page, limit)
    return response.OK(c, result)
}

// 方案 B: 前端直接调 Wiki Service（推荐，减少一层转发）
// 前端配置 WIKI_BASE_URL，直接请求 /api/galgame/:gid/revisions
```

## 4. 数据库拓扑

| 数据库 | 表 | 所有者 |
|--------|---|--------|
| `kun_galgame_wiki` | galgame, galgame_series, galgame_alias, galgame_tag, galgame_tag_alias, galgame_tag_relation, galgame_official, galgame_official_alias, galgame_official_relation, galgame_engine, galgame_engine_relation, galgame_link, galgame_pr, galgame_revision, galgame_contributor | Wiki Service |
| 站点主库 | galgame_like, galgame_favorite, galgame_comment, galgame_comment_like, galgame_rating, galgame_rating_like, galgame_rating_comment, galgame_resource, galgame_resource_provider, galgame_resource_link, galgame_resource_like, galgame_stats | 站点后端 |

### galgame_stats 表（站点本地维护）

```sql
CREATE TABLE galgame_stats (
    galgame_id INT PRIMARY KEY,
    like_count INT NOT NULL DEFAULT 0,
    favorite_count INT NOT NULL DEFAULT 0,
    resource_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    rating_count INT NOT NULL DEFAULT 0
);
```

## 5. 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| KUN_GALGAME_PG_DATABASE | kun_galgame_wiki | Wiki 数据库名 |
| KUN_GALGAME_PG_HOST | 同 KUN_PG_HOST | 可单独配置 |
| KUN_GALGAME_PG_PORT | 同 KUN_PG_PORT | |
| KUN_GALGAME_PG_USER | 同 KUN_PG_USER | |
| KUN_GALGAME_PG_PASSWORD | 同 KUN_PG_PASSWORD | |
| KUN_GALGAME_PORT | 9280 | Wiki Service 端口 |

## 6. 前端直连 vs 后端转发

| 操作 | 推荐方式 | 原因 |
|------|---------|------|
| 列表/详情 | 前端直连 Wiki Service | 纯读操作，减少转发 |
| 创建/编辑 | 后端转发 | 需要本地副作用（萌萌点等） |
| 版本历史/diff | 前端直连 | 纯读操作 |
| 提交 PR | 前端直连 | 无本地副作用 |
| 合并 PR | 后端转发 | 可能需要本地萌萌点奖励 |
| 标签/开发商/引擎/系列 | 前端直连 | 纯元数据操作 |

前端直连时，CORS 已在 Wiki Service 配置中处理（`KUN_FRONTEND_CORS_ORIGIN`）。
