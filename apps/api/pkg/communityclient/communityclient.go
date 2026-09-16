// Package communityclient reads and writes the nextmoe-infra community service.
//
// community is the platform's discussion primitive: one unit, a thread, whose
// anchor decides its shape. A sticker pack's comment section is the
// "entity-resource comments" shape -- anchor kind site_resource, anchor id the
// pack's uuid -- and both comment lanes are addressed by that anchor, so this
// site never stores a thread id of its own.
//
// A comments thread is born with its first comment. `GET /comments` reads and
// reports an anchor nobody has spoken about as exactly that (no thread, no
// posts); `POST /comments` writes and creates the thread in the same
// transaction as the comment. The deprecated `comments/resolve` did both at
// once, so rendering a pack page minted a thread -- which is how kungal ended
// up with 110,918 empty threads out of 114,070.
//
// Auth is S2S Basic with the site's OAuth client credentials, and the tenant is
// NOT on the wire: community derives it from the calling client, preferring
// oauth_clients.community_site and falling back to oauth_clients.catalog_site.
// That binding is what stops one site writing into another's threads, and it is
// also why setting community_site on a client that already has threads strands
// them: they were filed under the old tenant and nothing moves them.
//
// The acting user is different -- community trusts this site to have
// authenticated them, so every write carries an author_id this site vouches
// for, and every read that renders "you liked this" carries a viewer_id on the
// same terms. Neither may come from what a browser sent; both come from the
// session.
package communityclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("community: not configured")
	ErrNotFound      = errors.New("community: not found")
	ErrForbidden     = errors.New("community: forbidden")
	ErrRateLimited   = errors.New("community: rate limited")
	ErrConflict      = errors.New("community: conflict")
	ErrUpstream      = errors.New("community: upstream unavailable")
)

func Missing(err error) bool { return errors.Is(err, ErrNotFound) }

// Anchor kinds, from the service's closed vocabulary. A sticker pack is a
// resource this site owns.
const (
	AnchorBoard         = 0
	AnchorSiteGame      = 1
	AnchorSiteResource  = 2
	AnchorCatalogWork   = 3
	AnchorCatalogPerson = 4
)

// Thread kinds. A comment wall is a thread of kind comments; the other two
// shapes belong to sites that host a forum.
const (
	KindTopic    = 0
	KindComments = 1
	KindFeedback = 2
)

// AnyKind is the wildcard the filtering read lanes take for kind and
// anchor_kind. Zero is board/topic, a real value, so "no filter" needs its own.
const AnyKind = -1

// Post statuses. Anything but visible is either awaiting review or gone, and
// this site shows neither.
const (
	StatusVisible = 0
	StatusHeld    = 1
	StatusDeleted = 2
)

// Notification levels. Posting subscribes the author at watching; community
// never downgrades a level a user set for themselves.
const (
	NotifyMuted    = 0
	NotifyNormal   = 1
	NotifyTracking = 2
	NotifyWatching = 3
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

type Client struct {
	http   *http.Client
	origin string
	auth   string
}

func New(cfg Config) *Client {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return &Client{}
	}
	return &Client{
		http:   &http.Client{Timeout: 8 * time.Second},
		origin: strings.TrimSuffix(base, "/api/v1/community"),
		auth:   base64.StdEncoding.EncodeToString([]byte(cfg.ClientID + ":" + cfg.ClientSecret)),
	}
}

func (c *Client) Configured() bool { return c != nil && c.origin != "" && c.auth != "" }

// envelope is the house shape; community uses it on every response including
// errors, so a non-zero code is the failure even when the status is 200.
type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin+"/api/v1/community"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Basic "+c.auth)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read: %v", ErrUpstream, err)
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusConflict:
		return ErrConflict
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var env envelope[json.RawMessage]
		_ = json.Unmarshal(raw, &env)
		return fmt.Errorf("%w: %d %s", ErrUpstream, resp.StatusCode, env.Message)
	}
	// The envelope carries its own verdict, and the doc above says so. Reading
	// the status alone would take a 200 that reports code 1 for a success and
	// hand the caller a zero-valued page.
	var env envelope[json.RawMessage]
	if json.Unmarshal(raw, &env) == nil && env.Code != 0 {
		return fmt.Errorf("%w: %s", ErrUpstream, env.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("community decode: %w", err)
	}
	return nil
}

// withQuery joins a lane with its query. Filters whose zero value is a real
// value -- kind 0 is topic, anchor_kind 0 is board -- are always written, so a
// caller that means "topics" is never read as "no filter".
func withQuery(lane string, q url.Values) string {
	if encoded := q.Encode(); encoded != "" {
		return lane + "?" + encoded
	}
	return lane
}

type Thread struct {
	ID                int64  `json:"id"`
	Kind              int    `json:"kind"`
	Title             string `json:"title"`
	AnchorKind        int    `json:"anchor_kind"`
	AnchorID          string `json:"anchor_id"`
	PostsCount        int    `json:"posts_count"`
	ParticipantsCount int    `json:"participants_count"`
	HighestPostNumber int    `json:"highest_post_number"`
	LastPostedAt      string `json:"last_posted_at"`
	Status            int    `json:"status"`
}

type Post struct {
	ID          int64  `json:"id"`
	ThreadID    int64  `json:"thread_id"`
	PostNumber  int    `json:"post_number"`
	AuthorID    int    `json:"author_id"`
	ContentHTML string `json:"content_html"`
	ContentRaw  string `json:"content_raw"`
	Status      int    `json:"status"`
	// The caller sends reply_to_post_id; community derives root_post_id, so a
	// reply to a reply still groups under the comment that started it.
	ReplyToPostID     int64  `json:"reply_to_post_id"`
	RootPostID        int64  `json:"root_post_id"`
	TargetUserID      int    `json:"target_user_id"`
	CreatedAt         string `json:"created_at"`
	EditedAt          string `json:"edited_at"`
	EditedByModerator bool   `json:"edited_by_moderator"`
	// Likes are community's own count, hydrated on every face that returns a
	// post it did not just create. ViewerReacted is only ever true for the
	// viewer_id the request named, and a request that named nobody gets false
	// on every post.
	ReactionCount int  `json:"reaction_count"`
	ViewerReacted bool `json:"viewer_reacted"`
}

// CommentsPage is the read lane's answer. Thread is nil until somebody
// comments -- the anchor exists, the conversation does not -- and Posts is
// empty alongside it. A caller that treats a nil thread as an error shows a
// broken comment section on every pack nobody has spoken about yet.
type CommentsPage struct {
	Thread     *Thread `json:"thread"`
	Posts      []Post  `json:"posts"`
	NextCursor string  `json:"next_cursor"`
}

// Comments reads an anchor's comment wall. It writes nothing: no thread is
// created by rendering a page. `after` is a post_number, and the empty string
// starts from the top. viewerID is whose likes to report; 0 is an anonymous
// reader and leaves every viewer_reacted false.
func (c *Client) Comments(
	ctx context.Context,
	anchorKind int,
	anchorID, after string,
	limit, viewerID int,
) (*CommentsPage, error) {
	q := url.Values{}
	q.Set("anchor_kind", strconv.Itoa(anchorKind))
	q.Set("anchor_id", anchorID)
	if after != "" {
		q.Set("after", after)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if viewerID > 0 {
		q.Set("viewer_id", strconv.Itoa(viewerID))
	}
	var env envelope[CommentsPage]
	if err := c.do(ctx, http.MethodGet, withQuery("/comments", q), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type CommentParams struct {
	AnchorKind int
	AnchorID   string
	// ContentRating is stamped on the thread, so it only takes effect when this
	// comment is the one that creates it.
	ContentRating int
	AuthorID      int
	Body          string
	ReplyToPostID int64
}

// CommentResult carries the thread as it stands after the write -- newly
// created on the first comment, and already there on every one after it. The
// post is not hydrated, and needs no hydration: it was inserted by this call,
// so nobody has liked it yet.
type CommentResult struct {
	Thread Thread `json:"thread"`
	Post   Post   `json:"post"`
}

// Comment writes to an anchor's comment wall, creating the thread with the
// first comment. Two first comments racing do not make two conversations: the
// loser appends to the winner's thread.
func (c *Client) Comment(ctx context.Context, p CommentParams) (*CommentResult, error) {
	payload := map[string]any{
		"anchor_kind":    p.AnchorKind,
		"anchor_id":      p.AnchorID,
		"content_rating": p.ContentRating,
		"author_id":      p.AuthorID,
		"body":           p.Body,
	}
	if p.ReplyToPostID > 0 {
		payload["reply_to_post_id"] = p.ReplyToPostID
	}
	var env envelope[CommentResult]
	if err := c.do(ctx, http.MethodPost, "/comments", payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Edit rewrites a post. The answer carries the post's real like count, with
// the author standing as the viewer: the response is theirs to render.
func (c *Client) Edit(ctx context.Context, postID int64, authorID int, body string) (*Post, error) {
	// The post write lanes wrap their result: data is {"post": …}.
	var env envelope[postWrapper]
	payload := map[string]any{"author_id": authorID, "body": body}
	if err := c.do(ctx, http.MethodPatch, "/posts/"+strconv.FormatInt(postID, 10), payload, &env); err != nil {
		return nil, err
	}
	return &env.Data.Post, nil
}

type postWrapper struct {
	Post Post `json:"post"`
}

// Reaction kinds and flag reasons, from the service's closed vocabularies.
const (
	ReactionLike = 0
)

const (
	FlagSpam         = 0
	FlagAbuse        = 1
	FlagOffTopic     = 2
	FlagOther        = 3
	FlagNSFWMislabel = 4
)

// ReactionResult is what a toggle answers with: the clicker's new state, the
// post's like count counted in the same transaction as the toggle, and the
// post's context, which the reaction flow has resolved anyway. Added is the
// viewer_reacted a read face would now report for this user, and ReactionCount
// is the number it would report for everyone, so a toggle needs no re-read.
type ReactionResult struct {
	Added         bool   `json:"added"`
	ReactionCount int    `json:"reaction_count"`
	AuthorID      int    `json:"author_id"`
	ThreadID      int64  `json:"thread_id"`
	AnchorKind    int    `json:"anchor_kind"`
	AnchorID      string `json:"anchor_id"`
}

func (c *Client) ToggleReaction(ctx context.Context, postID int64, userID int, kind int) (*ReactionResult, error) {
	var env envelope[ReactionResult]
	payload := map[string]any{"user_id": userID, "kind": kind}
	if err := c.do(ctx, http.MethodPost, "/posts/"+strconv.FormatInt(postID, 10)+"/reaction", payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// Flag reports a post. community owns the review queue; nothing about the
// report is stored on this site.
func (c *Client) Flag(ctx context.Context, postID int64, flaggerID, reason int, note string) error {
	payload := map[string]any{"flagger_id": flaggerID, "reason": reason}
	if note != "" {
		payload["note"] = note
	}
	return c.do(ctx, http.MethodPost, "/posts/"+strconv.FormatInt(postID, 10)+"/flag", payload, nil)
}

// Delete tombstones a post. author_id is a query param upstream: the request
// stays body-free, matching the artifact service's DELETE.
func (c *Client) Delete(ctx context.Context, postID int64, authorID int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/posts/%d?author_id=%d", postID, authorID), nil, nil)
}
