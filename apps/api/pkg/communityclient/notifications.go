package communityclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Notification kinds, from the service's closed vocabulary. This site renders
// replies, posts on a watched wall, and likes; the others exist so an unknown
// kind is still a row the unread count covers rather than a dropped badge.
const (
	NotificationReplied        = 1
	NotificationMentioned      = 2
	NotificationPosted         = 3
	NotificationThreadCreated  = 4
	NotificationLiked          = 5
	NotificationAnswerAccepted = 6
	NotificationFeedbackStatus = 7
)

// Notification is one inbox row community wrote for this site. This site has
// no inbox table of its own, so the row is the whole truth: a GET must not
// mark it read, and a row whose pack is gone is still a row.
//
// Every field is a plain value: community sends null for a missing post and
// leaves out a missing actor or read time, and zero means "none" for each.
type Notification struct {
	ID              int64  `json:"id"`
	Kind            int    `json:"kind"`
	ThreadID        int64  `json:"thread_id"`
	AnchorKind      int    `json:"anchor_kind"`
	AnchorID        string `json:"anchor_id"`
	PostID          int64  `json:"post_id"`
	PostNumber      int    `json:"post_number"`
	FirstPostNumber int    `json:"first_post_number"`
	ActorID         int    `json:"actor_id"`
	ActorCount      int    `json:"actor_count"`
	ItemCount       int    `json:"item_count"`
	ReadAt          string `json:"read_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Seq             int64  `json:"seq"`
}

type NotificationList struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    string         `json:"next_cursor"`
	UnreadCount   int            `json:"unread_count"`
}

// Notifications is this site's inbox. community writes the rows for this
// tenant; this site stores none of them, so the list is the only face a reader
// has. Every row is returned, even one whose pack is gone: the unread_count on
// the same page is the red dot, and a badge that leads to a shorter list is
// the failure this avoids.
func (c *Client) Notifications(ctx context.Context, userID int, cursor string, limit int, unreadOnly bool) (*NotificationList, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if unreadOnly {
		q.Set("unread_only", "true")
	}
	var env envelope[NotificationList]
	lane := withQuery("/users/"+strconv.Itoa(userID)+"/notifications", q)
	if err := c.do(ctx, http.MethodGet, lane, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// MarkNotificationsRead acknowledges rows the reader has seen, by id or all at
// once, and answers with the unread count that is left. A GET must not do
// this: the list can be prefetched and the header asks for the count on every
// page, and either marking rows read would eat notifications nobody opened.
func (c *Client) MarkNotificationsRead(ctx context.Context, userID int, ids []int64, all bool) (int, error) {
	payload := map[string]any{}
	if all {
		payload["all"] = true
	} else {
		payload["ids"] = ids
	}
	var env envelope[struct {
		UnreadCount int `json:"unread_count"`
	}]
	lane := "/users/" + strconv.Itoa(userID) + "/notifications/read"
	if err := c.do(ctx, http.MethodPost, lane, payload, &env); err != nil {
		return 0, err
	}
	return env.Data.UnreadCount, nil
}

type AnchorRef struct {
	AnchorKind int    `json:"anchor_kind"`
	AnchorID   string `json:"anchor_id"`
}

type AnchorState struct {
	UserID            int    `json:"user_id"`
	AnchorKind        int    `json:"anchor_kind"`
	AnchorID          string `json:"anchor_id"`
	NotificationLevel int    `json:"notification_level"`
}

// SetAnchorNotification is the follow button for a wall that has no thread
// yet. Once a thread exists the thread row outranks this one, and community's
// first read receipt copies watching across; setting the anchor is how a
// reader follows a pack nobody has spoken about.
func (c *Client) SetAnchorNotification(ctx context.Context, userID int, anchor AnchorRef, level int) (*AnchorState, error) {
	var env envelope[AnchorState]
	payload := map[string]any{
		"user_id":     userID,
		"anchor_kind": anchor.AnchorKind,
		"anchor_id":   anchor.AnchorID,
		"level":       level,
	}
	if err := c.do(ctx, http.MethodPost, "/anchors/notification", payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// AnchorStates batches one user's stored follow over a list of anchors, so a
// pack page spends one round trip on its badge. An unsubscribed (normal)
// anchor is simply absent -- the same sparse shape as ThreadStates.
func (c *Client) AnchorStates(ctx context.Context, userID int, anchors []AnchorRef) ([]AnchorState, error) {
	if len(anchors) == 0 {
		return nil, nil
	}
	var env envelope[struct {
		States []AnchorState `json:"states"`
	}]
	payload := map[string]any{"user_id": userID, "anchors": anchors}
	if err := c.do(ctx, http.MethodPost, "/anchors/states", payload, &env); err != nil {
		return nil, err
	}
	return env.Data.States, nil
}
