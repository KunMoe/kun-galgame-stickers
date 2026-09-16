package communityclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// The feed faces read across threads rather than down one: a site's newest
// comments, and substring search over what commenters typed. Both answer with a
// post plus the thread it came from, because a comment read outside its own
// page is only useful if the reader can get back to the page.

// PostContext is the thread a fed post came from. It is the anchor, not a
// title, that lets this site turn a hit back into a page of its own.
type PostContext struct {
	ThreadID   int64  `json:"thread_id"`
	Title      string `json:"title"`
	AnchorKind int    `json:"anchor_kind"`
	AnchorID   string `json:"anchor_id"`
}

type FeedPost struct {
	Post   Post        `json:"post"`
	Thread PostContext `json:"thread"`
}

type PostFeed struct {
	Posts      []FeedPost `json:"posts"`
	NextCursor string     `json:"next_cursor"`
}

// SearchPosts matches a post's markdown source, case-insensitively, 2-100
// characters. It searches the source rather than the cooked HTML: searching the
// HTML makes `nofollow` a hit on every post that carries a link.
func (c *Client) SearchPosts(ctx context.Context, query string, kind int, cursor string, limit int) (*PostFeed, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("kind", strconv.Itoa(kind))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var env envelope[PostFeed]
	if err := c.do(ctx, http.MethodGet, withQuery("/search/posts", q), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type FeedParams struct {
	Kind       int
	AnchorKind int
	// AnchorID narrows to a single anchor and requires AnchorKind.
	AnchorID string
	// RepliesOnly drops post_number 1. On a comment wall that is the first
	// comment rather than an opening post, so this site leaves it false.
	RepliesOnly bool
	Cursor      string
	Limit       int
}

// LatestPosts is the site's newest posts across every thread. It keysets on
// creation time rather than id because the kungal import gave historical
// comments fresh ids, making id order import order.
func (c *Client) LatestPosts(ctx context.Context, p FeedParams) (*PostFeed, error) {
	q := url.Values{}
	q.Set("kind", strconv.Itoa(p.Kind))
	q.Set("anchor_kind", strconv.Itoa(p.AnchorKind))
	if p.AnchorID != "" {
		q.Set("anchor_id", p.AnchorID)
	}
	if p.RepliesOnly {
		q.Set("replies_only", "true")
	}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	var env envelope[PostFeed]
	if err := c.do(ctx, http.MethodGet, withQuery("/posts", q), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
