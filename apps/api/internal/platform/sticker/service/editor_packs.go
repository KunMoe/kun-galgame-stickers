package service

import (
	"log/slog"
	"strconv"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/pkg/errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// editorPackVariant is what a picker grid renders. 320 is the same variant the
// site's own grid uses; an editor tile is the same size as a pack-page tile.
const editorPackVariant = "320"

// EditorPacks is the whole official sticker catalogue in one response, shaped
// for a sticker picker.
//
// It exists because every editor in the ecosystem shipped its own copy of the
// same 498 URLs, generated from `KUNgal{set}/{n}.webp` with a hardcoded set of
// pack sizes -- forum, moyu and this site each had one, all of them dead the
// day this host stopped serving static files, and all of them silently wrong
// long before that (a pack gained stickers; the arrays did not).
//
// Two shape decisions:
//
//   - ONE request, not the face's list-then-fetch-each. A picker needs every
//     sticker of every pack at once, and 1+N round trips to build one panel is
//     the sort of thing consumers work around by hardcoding again.
//   - `src` is the CDN URL, and it is what the picker INSERTS INTO POST
//     CONTENT. That is the actual fix for the class of bug: the old path
//     encoded a position in a mutable collection, so years of posts rotted
//     when the collection moved. A content hash encodes the bytes and cannot.
//     The corollary is a rule for this site, not for consumers: official
//     stickers are retired by status, never hard-deleted, or the image
//     service's refcount drops to zero and the old posts break anyway.
//
// Official packs only. A picker offered in someone else's editor is not the
// place to surface unmoderated user uploads.
func (s *Service) EditorPacks() (*dto.EditorPacks, *errors.AppError) {
	rows, err := s.packs.OfficialPublished()
	if err != nil {
		slog.Error("editor packs query failed", "error", err)
		return nil, errors.ErrInternal("failed to load the sticker packs")
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	byPack, err := s.stickers.ForPacks(ids)
	if err != nil {
		slog.Error("editor packs sticker query failed", "error", err)
		return nil, errors.ErrInternal("failed to load the sticker packs")
	}

	packs := make([]dto.EditorPack, 0, len(rows))
	for _, row := range rows {
		label := displayTitle(row.Title)
		stickers := make([]dto.EditorSticker, 0, len(byPack[row.ID]))
		for _, st := range byPack[row.ID] {
			if st.ImageHash == "" || s.images == nil {
				continue
			}
			stickers = append(stickers, dto.EditorSticker{
				Src:  s.images.VariantURL(st.ImageHash, editorPackVariant),
				Name: label + " - " + strconv.Itoa(st.Position),
				Hash: st.ImageHash,
			})
		}
		if len(stickers) == 0 {
			continue
		}
		packs = append(packs, dto.EditorPack{Name: label, Stickers: stickers})
	}
	return &dto.EditorPacks{Variant: editorPackVariant, Packs: packs}, nil
}

// displayTitle flattens a multilingual title to the one string a picker tab
// can show. The picker has no locale to negotiate with -- it is rendered
// inside somebody else's app -- so this is a fixed preference order rather
// than content negotiation, and zh-cn leads because that is what the official
// packs are authored in.
var titlePreference = []string{"zh-cn", "zh-tw", "ja-jp", "en-us", "und"}

func displayTitle(raw datatypes.JSON) string {
	title := decodeML(raw)
	for _, key := range titlePreference {
		if value := title[key]; value != "" {
			return value
		}
	}
	for _, value := range title {
		if value != "" {
			return value
		}
	}
	return "Stickers"
}
