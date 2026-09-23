package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/catalogclient"
	"kun-galgame-sticker-api/pkg/communityclient"
	"kun-galgame-sticker-api/pkg/imageclient"
	"kun-galgame-sticker-api/pkg/perm"
	"kun-galgame-sticker-api/pkg/userclient"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	MaxStickersPerPack = 80
	MaxPacksPerUser    = 20
	MaxUploadsPerDay   = 200
	MaxUploadBytes     = 10 << 20
	MaxTagsPerPack     = 10
	MaxTextRunes       = 200

	// characterStickerLimit caps the character page. A popular character can
	// appear in far more stickers than one page should render, and the page
	// leads with the newest.
	characterStickerLimit = 120

	imagePreset  = "sticker"
	thumbVariant = "320"
)

// localeKeys is the multilingual key space the database uses. It is wider than
// the three UI locales on purpose: zh-tw has no UI of its own but the seeded
// official packs carry it.
var localeKeys = map[string]bool{
	"zh-cn": true,
	"zh-tw": true,
	"ja-jp": true,
	"en-us": true,
}

// Viewer is who is asking. Roles come from the OP and decide whether a
// moderator may see something its author has not published.
type Viewer struct {
	UID int
	// Sub is the OP's uuid for this user. The image service records it as the
	// uploader so an image can be traced back to a person, not just to a site.
	Sub   string
	Roles []string
}

func (v Viewer) canSeeUnpublished(p *model.Pack) bool {
	if v.UID > 0 && p.OwnerUID == v.UID {
		return true
	}
	return perm.Can(v.Roles, perm.PackViewHidden)
}

func (v Viewer) canEdit(p *model.Pack) bool {
	if v.UID > 0 && p.OwnerUID == v.UID {
		return true
	}
	return perm.Can(v.Roles, perm.PackEditAny)
}

type Service struct {
	packs     *repository.PackRepo
	stickers  *repository.StickerRepo
	tags      *repository.TagRepo
	images    *imageclient.Client
	users     *userclient.Client
	catalog   *catalogclient.Client
	community *communityclient.Client
	http      *http.Client
}

func New(
	packs *repository.PackRepo,
	stickers *repository.StickerRepo,
	tags *repository.TagRepo,
	images *imageclient.Client,
	users *userclient.Client,
	catalog *catalogclient.Client,
	community *communityclient.Client,
) *Service {
	return &Service{
		packs:     packs,
		stickers:  stickers,
		tags:      tags,
		images:    images,
		users:     users,
		catalog:   catalog,
		community: community,
		http:      &http.Client{Timeout: downloadTimeout},
	}
}

func (s *Service) urls(hash string) (main string, thumb string) {
	if hash == "" || s.images == nil {
		return "", ""
	}
	return s.images.MainURL(hash), s.images.VariantURL(hash, thumbVariant)
}

func (s *Service) stickerDTO(row model.Sticker) dto.Sticker {
	main, thumb := s.urls(row.ImageHash)
	return dto.Sticker{
		ID:            row.ID.String(),
		PackID:        row.PackID.String(),
		Position:      row.Position,
		Width:         row.Width,
		Height:        row.Height,
		Game:          decodeML(row.Game),
		CharacterName: decodeML(row.CharacterName),
		VndbID:        row.VndbID,
		Note:          row.Note,
		ImageURL:      main,
		ThumbURL:      thumb,
		CatalogWork:   workDTO(row.CatalogWorkID, row.CatalogWorkName, "", row.CatalogWorkRating),
		CatalogCharacter: characterDTO(
			row.CatalogCharacterID, row.CatalogCharacterName, row.CatalogCharacterImage,
		),
	}
}

func (s *Service) packDTO(
	row model.Pack,
	cover *model.Sticker,
	tags []model.Tag,
	author userclient.User,
) dto.Pack {
	out := dto.Pack{
		ID:            row.ID.String(),
		Status:        row.Status,
		IsOfficial:    row.IsOfficial,
		ContentRating: row.ContentRating,
		Title:         decodeML(row.Title),
		Description:   decodeML(row.Description),
		StickerCount:  row.StickerCount,
		ViewCount:     row.ViewCount,
		DownloadCount: row.DownloadCount,
		Author:        authorDTO(author),
		Tags:          tagDTOs(tags),
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.UTC().Format(time.RFC3339),
		CatalogWork:   workDTO(row.CatalogWorkID, row.CatalogWorkName, row.CatalogWorkCover, row.CatalogWorkRating),
	}
	if cover != nil {
		out.CoverURL, out.CoverThumbURL = s.urls(cover.ImageHash)
		out.CoverStickerID = cover.ID.String()
	}
	if row.PublishedAt != nil {
		published := row.PublishedAt.UTC().Format(time.RFC3339)
		out.PublishedAt = &published
	}
	return out
}

func authorDTO(u userclient.User) dto.Author {
	return dto.Author{ID: u.ID, Name: u.Name, Avatar: u.Avatar, Cosmetics: u.Cosmetics}
}

func tagDTOs(rows []model.Tag) []dto.Tag {
	out := make([]dto.Tag, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.Tag{
			ID:        row.ID.String(),
			Slug:      row.Slug,
			Name:      decodeML(row.Name),
			PackCount: row.PackCount,
		})
	}
	return out
}

func decodeML(raw datatypes.JSON) dto.MultilingualText {
	out := dto.MultilingualText{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	for key, value := range out {
		if value == "" {
			delete(out, key)
		}
	}
	return out
}

// sanitizeML drops unknown locale keys, trims, and truncates by rune. The old
// implementation sliced by byte, which cut multi-byte characters in half.
func sanitizeML(in dto.MultilingualText) datatypes.JSON {
	clean := map[string]string{}
	for key, value := range in {
		key = strings.ToLower(strings.TrimSpace(key))
		if !localeKeys[key] {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if runes := []rune(value); len(runes) > MaxTextRunes {
			value = string(runes[:MaxTextRunes])
		}
		clean[key] = value
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return datatypes.JSON("{}")
	}
	return datatypes.JSON(raw)
}

func hasText(raw datatypes.JSON) bool { return len(decodeML(raw)) > 0 }

// searchText flattens every language of every searchable field into the one
// column the trigram index sits on, so a Japanese query finds a pack whose
// Japanese title matches even when the UI is Chinese. Callers pass the title,
// the description, and the names of the games involved -- a pack made from a
// game should be findable by that game's name even when its own title never
// mentions it.
func searchText(fields ...datatypes.JSON) string {
	parts := make([]string, 0, 12)
	seen := map[string]bool{}
	for _, field := range fields {
		for _, value := range decodeML(field) {
			if seen[value] {
				continue
			}
			seen[value] = true
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}

func parseUUID(raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}
