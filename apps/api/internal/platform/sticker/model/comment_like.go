package model

import "time"

// CommentLike mirrors a community reaction so this site can render counts.
// post_id belongs to the community service, not to this database, which is why
// there is no foreign key to hang it on.
//
// The mirror predates community carrying its own counts and is no longer the
// only way to render one: its read faces now return reaction_count and
// viewer_reacted. Two writers on one fact drift -- a toggle that reaches
// community and fails here is never reconciled, and community feeds the trust
// engine from its own rows either way -- so this table is a candidate for
// retirement, not a second source of truth to keep in step.
type CommentLike struct {
	PostID    int64     `gorm:"column:post_id;primaryKey"`
	UserID    int       `gorm:"column:user_id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (CommentLike) TableName() string { return "comment_like" }
