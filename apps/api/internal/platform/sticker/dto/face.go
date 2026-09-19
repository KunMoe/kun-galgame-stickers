package dto

import (
	"encoding/base64"
	"strconv"
	"strings"
)

// The public developer-platform face has its own wire types on purpose. The
// DTOs alongside serve this site's own frontend and change whenever a page
// does; these are a contract a third party builds against, gated by oasdiff.
// Sharing one struct between the two would make every UI tweak a breaking API
// change.
//
// Every object carries `object`, as catalog's /v2 items do, so a mixed result
// can be discriminated without looking at the URL it came from. Every id is a
// string, as catalog's are: a catalog or account id is a decimal string that
// happens to be numeric, not a number a client should do arithmetic on.

// FaceList is the list envelope, field for field catalog /v2's: next_cursor is
// omitted on the last page rather than null, and total only appears when the
// caller asked for it with include_total.
type FaceList[T any] struct {
	Object     string  `json:"object"`
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor,omitempty"`
	Total      *int64  `json:"total,omitempty"`
}

// FacePage is one page request against a face collection.
type FacePage struct {
	Offset       int
	Limit        int
	IncludeTotal bool
}

// NewFaceList closes a page. total is counted under the same filter as items;
// every query behind the face counts anyway, so it decides whether a next
// page exists even when the caller did not ask to see it.
func NewFaceList[T any](items []T, page FacePage, total int64) *FaceList[T] {
	out := &FaceList[T]{Object: "list", Items: items}
	// An empty page never points onward: rows deleted between the count and
	// the read would otherwise hand back the cursor the caller just sent.
	if next := page.Offset + len(items); len(items) > 0 && int64(next) < total {
		out.NextCursor = FaceCursor(next)
	}
	if page.IncludeTotal {
		out.Total = &total
	}
	return out
}

// The cursor keeps offsets underneath, which stay honest at this site's size
// -- its collections are three orders of magnitude smaller than catalog's.
// The encoding is byte for byte infra's collect.EncodeOffset, `cur_` plus the
// unpadded base64url of the decimal offset, so the platform could page this
// face with its own helpers; moyu's face encodes the same way.
const faceCursorPrefix = "cur_"

// FaceCursor is nil at offset zero, where there is nothing to resume.
func FaceCursor(offset int) *string {
	if offset <= 0 {
		return nil
	}
	out := faceCursorPrefix + base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
	return &out
}

// ParseFaceCursor is FaceCursor's inverse. ok is false for anything it could
// not have minted.
func ParseFaceCursor(raw string) (offset int, ok bool) {
	payload, found := strings.CutPrefix(raw, faceCursorPrefix)
	if !found || payload == "" {
		return 0, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(string(decoded))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// FaceImage is how every picture leaves the face. The hash is the stable
// identity -- the content address the infra image service dedupes on, shared
// across every site storing the same bytes -- and the URLs are a convenience
// assembled from it. A caller that stores anything should store the hash.
type FaceImage struct {
	// Hash and the dimensions are absent on a pack cover, which is a rendering
	// convenience pointing at cover_sticker_id; on a sticker they are always
	// present, and the hash is the thing worth storing.
	Hash     string `json:"hash,omitempty"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

// FaceWork is a catalog work as this site knows it: catalog's id plus the
// snapshot taken when a pack or sticker was linked. Names are the four-locale
// map plus `und`, which is what a catalog display name with no language tag
// folds to.
type FaceWork struct {
	Object        string           `json:"object"`
	ID            string           `json:"id"`
	Name          MultilingualText `json:"name"`
	CoverURL      string           `json:"cover_url,omitempty"`
	ContentRating string           `json:"content_rating,omitempty"`
	// StickerCount is set on the work index only. How many packs a game
	// appears in is answered by /works/{id}/packs, which owns that definition.
	StickerCount int64 `json:"sticker_count,omitempty"`
}

type FaceCharacter struct {
	Object       string           `json:"object"`
	ID           string           `json:"id"`
	Name         MultilingualText `json:"name"`
	ImageURL     string           `json:"image_url,omitempty"`
	StickerCount int64            `json:"sticker_count,omitempty"`
	Work         *FaceWork        `json:"work,omitempty"`
}

type FaceAuthor struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type FaceTag struct {
	Object    string           `json:"object"`
	Slug      string           `json:"slug"`
	Name      MultilingualText `json:"name"`
	PackCount int              `json:"pack_count"`
}

// FacePack never carries `status`: the face only ever answers with published
// packs, so the field would be a constant pretending to be data.
type FacePack struct {
	Object      string           `json:"object"`
	ID          string           `json:"id"`
	Title       MultilingualText `json:"title"`
	Description MultilingualText `json:"description"`
	Official    bool             `json:"official"`
	// ContentRating speaks catalog's vocabulary rather than this site's 0/1,
	// so one client vocabulary covers both faces. A pack is all_ages or r18;
	// the middle value catalog also has (sensitive) is not recorded here.
	ContentRating  string     `json:"content_rating"`
	StickerCount   int        `json:"sticker_count"`
	ViewCount      int64      `json:"view_count"`
	DownloadCount  int64      `json:"download_count"`
	Cover          *FaceImage `json:"cover,omitempty"`
	CoverStickerID string     `json:"cover_sticker_id,omitempty"`
	Work           *FaceWork  `json:"work,omitempty"`
	Tags           []FaceTag  `json:"tags"`
	Author         FaceAuthor `json:"author"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	PublishedAt    *string    `json:"published_at,omitempty"`
}

// FacePackDetail adds the pack's stickers and the distinct catalog identities
// they carry. The seeded official packs each span dozens of games, so works
// and characters are lists, not a single link.
type FacePackDetail struct {
	FacePack
	Stickers   []FaceSticker   `json:"stickers"`
	Works      []FaceWork      `json:"works"`
	Characters []FaceCharacter `json:"characters"`
}

type FaceSticker struct {
	Object    string         `json:"object"`
	ID        string         `json:"id"`
	PackID    string         `json:"pack_id"`
	Position  int            `json:"position"`
	Image     FaceImage      `json:"image"`
	Note      string         `json:"note,omitempty"`
	Work      *FaceWork      `json:"work,omitempty"`
	Character *FaceCharacter `json:"character,omitempty"`
}
