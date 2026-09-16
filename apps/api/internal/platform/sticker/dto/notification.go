package dto

type NotificationItem struct {
	ID       int64 `json:"id"`
	Kind     int   `json:"kind"`
	ThreadID int64 `json:"thread_id"`
	// Pack is absent when the anchor is not a published pack of this site.
	// The row is still shown, so the badge and the list agree.
	Pack *CommentPackRef `json:"pack,omitempty"`
	// Actor is absent when community has erased the actor (a purged account).
	Actor           *Author `json:"actor,omitempty"`
	ActorCount      int     `json:"actor_count"`
	ItemCount       int     `json:"item_count"`
	PostID          int64   `json:"post_id,omitempty"`
	PostNumber      int     `json:"post_number,omitempty"`
	FirstPostNumber int     `json:"first_post_number,omitempty"`
	Read            bool    `json:"read"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type NotificationPage struct {
	Items       []NotificationItem `json:"items"`
	NextCursor  string             `json:"next_cursor,omitempty"`
	UnreadCount int                `json:"unread_count"`
	Enabled     bool               `json:"enabled"`
}

type NotificationCount struct {
	UnreadCount int  `json:"unread_count"`
	Enabled     bool `json:"enabled"`
}

type NotificationReadRequest struct {
	IDs []int64 `json:"ids"`
	All bool    `json:"all"`
}
