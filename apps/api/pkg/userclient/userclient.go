// Package userclient resolves author identities from the infra OAuth service.
//
// Stickers keeps no local user table: pack.owner_uid is the OP's integer user
// id and nothing else. Rendering "pack by @name" for a page of packs therefore
// needs one batch lookup, not one call per pack.
package userclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"kun-galgame-sticker-api/pkg/imageclient"
	"kun-galgame-sticker-api/pkg/perm"
)

const maxIDsPerRequest = 100

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	ImageCDNBase string
	CacheTTL     time.Duration
	NegCacheTTL  time.Duration
	HTTPTimeout  time.Duration
}

type User struct {
	ID              int        `json:"id"`
	UUID            string     `json:"uuid"`
	Name            string     `json:"name"`
	Avatar          string     `json:"avatar"`
	AvatarImageHash string     `json:"avatar_image_hash"`
	Bio             string     `json:"bio"`
	Status          int        `json:"status"`
	Roles           []string   `json:"roles"`
	SiteRoles       []string   `json:"site_roles"`
	Cosmetics       *Cosmetics `json:"cosmetics,omitempty"`
}

type Cosmetics struct {
	AvatarFrame       *Decoration `json:"avatar_frame,omitempty"`
	ProfileBackground *Decoration `json:"profile_background,omitempty"`
}

type Decoration struct {
	ItemID      int64  `json:"item_id"`
	Name        string `json:"name"`
	StaticURL   string `json:"static_url"`
	AnimatedURL string `json:"animated_url,omitempty"`
}

type Client struct {
	cfg     Config
	http    *http.Client
	authHd  string
	cdnBase string

	mu     sync.Mutex
	hot    map[int]entry
	missAt map[int]time.Time
}

type entry struct {
	user   User
	expire time.Time
}

func New(cfg Config) *Client {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Minute
	}
	if cfg.NegCacheTTL == 0 {
		cfg.NegCacheTTL = time.Minute
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 5 * time.Second
	}
	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: cfg.HTTPTimeout},
		authHd:  "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret)),
		cdnBase: strings.TrimRight(cfg.ImageCDNBase, "/"),
		hot:     map[int]entry{},
		missAt:  map[int]time.Time{},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.cfg.BaseURL != "" && c.cfg.ClientID != ""
}

// Placeholder is what a caller renders for an id the OP does not know, or when
// the OP is unreachable. A missing author must not blank out a whole page.
func Placeholder(id int) User {
	return User{ID: id, Name: "user" + strconv.Itoa(id)}
}

func (c *Client) Users(ctx context.Context, ids []int) map[int]User {
	out := make(map[int]User, len(ids))
	if len(ids) == 0 {
		return out
	}

	var wanted []int
	now := time.Now()
	c.mu.Lock()
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, done := out[id]; done {
			continue
		}
		if e, ok := c.hot[id]; ok && e.expire.After(now) {
			out[id] = e.user
			continue
		}
		if until, ok := c.missAt[id]; ok && until.After(now) {
			out[id] = Placeholder(id)
			continue
		}
		wanted = append(wanted, id)
	}
	c.mu.Unlock()

	if len(wanted) == 0 || !c.Configured() {
		for _, id := range wanted {
			out[id] = Placeholder(id)
		}
		return out
	}

	for chunk := range slicesChunk(wanted, maxIDsPerRequest) {
		users, err := c.fetch(ctx, chunk)
		if err != nil {
			for _, id := range chunk {
				out[id] = Placeholder(id)
			}
			continue
		}
		found := make(map[int]bool, len(users))
		for i := range users {
			u := &users[i]
			u.Avatar = c.avatarURL(*u)
			u.Roles = perm.Union(u.Roles, u.SiteRoles)
			out[u.ID] = *u
			found[u.ID] = true
		}
		c.remember(users, chunk, found)
		for _, id := range chunk {
			if !found[id] {
				out[id] = Placeholder(id)
			}
		}
	}
	return out
}

func (c *Client) User(ctx context.Context, id int) User {
	users := c.Users(ctx, []int{id})
	if u, ok := users[id]; ok {
		return u
	}
	return Placeholder(id)
}

func (c *Client) remember(users []User, asked []int, found map[int]bool) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, u := range users {
		c.hot[u.ID] = entry{user: u, expire: now.Add(c.cfg.CacheTTL)}
		delete(c.missAt, u.ID)
	}
	for _, id := range asked {
		if !found[id] {
			c.missAt[id] = now.Add(c.cfg.NegCacheTTL)
		}
	}
}

func (c *Client) avatarURL(u User) string {
	if c.cdnBase != "" && u.AvatarImageHash != "" {
		if url := imageclient.MainURL(c.cdnBase, u.AvatarImageHash, "webp"); url != "" {
			return url
		}
	}
	return u.Avatar
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type batchData struct {
	Users    []User `json:"users"`
	NotFound []int  `json:"not_found"`
}

func (c *Client) fetch(ctx context.Context, ids []int) ([]User, error) {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	endpoint := c.cfg.BaseURL + "/users/batch?ids=" + strings.Join(parts, ",")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHd)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userclient: %w", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("userclient: decode envelope: %w", err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("userclient: op returned %d: %s", env.Code, env.Message)
	}
	var data batchData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("userclient: decode data: %w", err)
	}
	return data.Users, nil
}

func slicesChunk(ids []int, size int) func(func([]int) bool) {
	return func(yield func([]int) bool) {
		for start := 0; start < len(ids); start += size {
			end := min(start+size, len(ids))
			if !yield(ids[start:end]) {
				return
			}
		}
	}
}
