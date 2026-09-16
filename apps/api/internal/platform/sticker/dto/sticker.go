package dto

type MultilingualText map[string]string

type Author struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type Tag struct {
	ID        string           `json:"id"`
	Slug      string           `json:"slug"`
	Name      MultilingualText `json:"name"`
	PackCount int              `json:"pack_count"`
}

// CatalogWork and CatalogCharacter are the snapshot this site stores of an
// infra catalog identity. ID is catalog's, and is what a link resolves against.
type CatalogWork struct {
	ID          int64            `json:"id"`
	Name        MultilingualText `json:"name"`
	CoverURL    string           `json:"cover_url"`
	ReleaseDate *string          `json:"release_date,omitempty"`
	Medium      string           `json:"medium,omitempty"`
	// ContentRating is catalog's all_ages | sensitive | r18. Unlike the images'
	// sexual field it is populated, so it is what the UI badges.
	ContentRating string `json:"content_rating,omitempty"`
}

type CatalogCharacter struct {
	ID         int64            `json:"id"`
	Name       MultilingualText `json:"name"`
	ImageURL   string           `json:"image_url"`
	RosterRole string           `json:"roster_role,omitempty"`
	Gender     *string          `json:"gender,omitempty"`
	Birthday   *string          `json:"birthday,omitempty"`
	BloodType  *string          `json:"blood_type,omitempty"`
	Traits     []CatalogTrait   `json:"traits,omitempty"`
	Aliases    []string         `json:"aliases,omitempty"`
	// Set on search hits: the game the character is from, and how many
	// stickers of them this site has, so a palette row can say where it leads.
	WorkName     MultilingualText `json:"work_name,omitempty"`
	StickerCount int              `json:"sticker_count,omitempty"`
}

// SearchResults is the command palette's payload: two short lanes, each
// already shaped the way its card renders.
type SearchResults struct {
	Packs      []Pack             `json:"packs"`
	Characters []CatalogCharacter `json:"characters"`
}

type CatalogTrait struct {
	Name  MultilingualText `json:"name"`
	Group MultilingualText `json:"group"`
}

type Sticker struct {
	ID            string           `json:"id"`
	PackID        string           `json:"pack_id"`
	Position      int              `json:"position"`
	Width         int              `json:"width"`
	Height        int              `json:"height"`
	Game          MultilingualText `json:"game"`
	CharacterName MultilingualText `json:"character_name"`
	VndbID        *int             `json:"vndb_id,omitempty"`
	Note          string           `json:"note"`
	ImageURL      string           `json:"image_url"`
	ThumbURL      string           `json:"thumb_url"`

	CatalogWork      *CatalogWork      `json:"catalog_work,omitempty"`
	CatalogCharacter *CatalogCharacter `json:"catalog_character,omitempty"`
}

type Pack struct {
	ID            string           `json:"id"`
	Status        int16            `json:"status"`
	IsOfficial    bool             `json:"is_official"`
	ContentRating int16            `json:"content_rating"`
	Title         MultilingualText `json:"title"`
	Description   MultilingualText `json:"description"`
	CoverURL      string           `json:"cover_url"`
	CoverThumbURL string           `json:"cover_thumb_url"`
	// Which sticker the cover is. The editor sets it and the face resolves
	// through it; before this the only way to read it back was to guess.
	CoverStickerID string  `json:"cover_sticker_id,omitempty"`
	StickerCount   int     `json:"sticker_count"`
	ViewCount      int64   `json:"view_count"`
	DownloadCount  int64   `json:"download_count"`
	Author         Author  `json:"author"`
	Tags           []Tag   `json:"tags"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	PublishedAt    *string `json:"published_at,omitempty"`

	CatalogWork *CatalogWork `json:"catalog_work,omitempty"`
}

type PackDetail struct {
	Pack
	Stickers []Sticker `json:"stickers"`
	// Characters are the distinct catalog characters across this pack's
	// stickers, and Works the distinct games. A pack that declares one game is
	// the common case, but the seeded official packs span dozens.
	Characters []CatalogCharacter `json:"characters"`
	Works      []CatalogWork      `json:"works"`
}

// CharacterPage is the public /character/{id} route: catalog's profile plus
// every sticker on this site that is tagged with that character.
type CharacterPage struct {
	Character   CatalogCharacter   `json:"character"`
	Stickers    []CharacterSticker `json:"stickers"`
	Packs       map[string]Pack    `json:"packs"`
	Appearances []CatalogWork      `json:"appearances"`
	// Profile says whether the catalog half loaded. False means the page is
	// rendering from stored snapshots because catalog was unreachable.
	Profile bool `json:"profile"`
}

type CharacterSticker struct {
	ID       string `json:"id"`
	PackID   string `json:"pack_id"`
	ImageURL string `json:"image_url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PackListPage struct {
	Packs []Pack `json:"packs"`
	Total int64  `json:"total"`
}

type UploadResult struct {
	Hash     string `json:"hash"`
	ImageURL string `json:"image_url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// Comment is one post in a pack's comment thread. The body arrives already
// cooked and sanitized by community; this site renders content_html and never
// re-processes it.
type Comment struct {
	ID          int64  `json:"id"`
	PostNumber  int    `json:"post_number"`
	ContentHTML string `json:"content_html"`
	ContentRaw  string `json:"content_raw"`
	CreatedAt   string `json:"created_at"`
	EditedAt    string `json:"edited_at,omitempty"`
	Author      Author `json:"author"`
	// Told to the client so the UI does not offer an action the API will
	// refuse; the API checks again regardless.
	CanEdit   bool `json:"can_edit"`
	CanDelete bool `json:"can_delete"`
	LikeCount int  `json:"like_count"`
	IsLiked   bool `json:"is_liked"`
	// ReplyTo is what the author answered; RootID is the top-level comment the
	// exchange hangs under, which community derives rather than the caller.
	ReplyTo     int64  `json:"reply_to,omitempty"`
	RootID      int64  `json:"root_id,omitempty"`
	ReplyToName string `json:"reply_to_name,omitempty"`
}

type CommentPage struct {
	// ThreadID is 0 until the first comment creates the thread. A pack nobody
	// has spoken about has a comment section and no thread behind it.
	ThreadID   int64     `json:"thread_id"`
	Comments   []Comment `json:"comments"`
	Total      int       `json:"total"`
	NextCursor string    `json:"next_cursor,omitempty"`
	// HighestPostNumber is what a read receipt reports having reached. It
	// counts tombstones, so it is not len(Comments).
	HighestPostNumber int `json:"highest_post_number"`
	// Viewer is the reader's own state on this thread, and is absent for an
	// anonymous reader or one who has never touched it. Before the thread
	// exists it carries only the signed-in reader's follow of the pack.
	Viewer *CommentViewerState `json:"viewer,omitempty"`
	// Enabled is false when the community service is not configured, which is
	// how a pack page knows to leave the section out rather than show an error.
	Enabled bool `json:"enabled"`
}

// CommentViewerState mirrors community's sparse (thread, user) row: how far
// this reader has read and whether they are subscribed.
type CommentViewerState struct {
	LastReadPostNumber int `json:"last_read_post_number"`
	UnreadCount        int `json:"unread_count"`
	NotificationLevel  int `json:"notification_level"`
}

type CommentRequest struct {
	Body    string `json:"body"`
	ReplyTo int64  `json:"reply_to"`
}

type CommentFlagRequest struct {
	Reason int    `json:"reason"`
	Note   string `json:"note"`
}

type CommentLikeResult struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

type CommentReadRequest struct {
	PostNumber int `json:"post_number"`
}

type CommentNotificationRequest struct {
	Level int `json:"level"`
}

// CommentPackRef is the pack a comment was left on, as much of it as a list
// row needs. The full pack is one click away and costs a query per row here.
type CommentPackRef struct {
	ID            string           `json:"id"`
	Title         MultilingualText `json:"title"`
	CoverThumbURL string           `json:"cover_thumb_url,omitempty"`
}

// CommentFeedItem is a comment seen from outside its pack -- enough to read the
// line and click through, without the actions that only mean something in
// place. Every feed on this site (latest, search) is made of these.
type CommentFeedItem struct {
	ID          int64          `json:"id"`
	PostNumber  int            `json:"post_number"`
	ContentHTML string         `json:"content_html"`
	CreatedAt   string         `json:"created_at"`
	Author      Author         `json:"author"`
	Pack        CommentPackRef `json:"pack"`
}

type CommentFeed struct {
	Items      []CommentFeedItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	Enabled    bool              `json:"enabled"`
}

// AvatarPool is the default-avatar manifest served to other NextMoe sites.
// URLs are absolute and content-addressed; Variant is informational (which
// image-service variant they point at). Consumers index the array with
// hash(seed) % len(urls) and must not parse or rebuild the URLs.
type AvatarPool struct {
	Version string   `json:"version"`
	Variant string   `json:"variant"`
	URLs    []string `json:"urls"`
}

// EditorPacks is the sticker-picker payload other sites' editors render. The
// field names match @kungal/editor-core's StickerPack/StickerItem so a
// consumer can hand the response straight to its `stickerSource` adapter.
type EditorPacks struct {
	// Variant is which image-service derivative Src points at, so a consumer
	// storing Hash instead of Src can rebuild the same URL. One value for the
	// whole payload: a picker grid renders every tile at one size.
	Variant string       `json:"variant"`
	Packs   []EditorPack `json:"packs"`
}

type EditorPack struct {
	Name     string          `json:"name"`
	Stickers []EditorSticker `json:"stickers"`
}

type EditorSticker struct {
	Src  string `json:"src"`
	Name string `json:"name"`
	// Hash is the identity Src is built from. A consumer that stores Src in
	// post content bakes this host into every post it ever writes, and the
	// forum lost 1620 sticker embeds to exactly that when the previous host
	// stopped serving them. Hash lets a consumer store a host-free reference
	// and resolve it at render time instead of parsing Src back apart.
	Hash string `json:"hash"`
}
