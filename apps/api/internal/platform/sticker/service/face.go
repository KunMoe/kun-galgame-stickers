package service

import (
	"context"
	"strconv"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"

	"github.com/google/uuid"
)

// This file is the public developer-platform face. Everything here answers as
// an anonymous reader of published content: there is no Viewer, because the
// gateway authenticates an application, not a person, and an application is
// never the author of anything here. Passing the zero Viewer into the shared
// queries is what enforces that -- it can see exactly what a logged-out
// browser can.

// FaceListPacks pages published packs. It is the site's own list query with
// the viewer removed, so the two can never drift on what "published" means.
func (s *Service) FaceListPacks(ctx context.Context, q dto.ListQuery, page dto.FacePage) (*dto.FaceList[dto.FacePack], *errors.AppError) {
	q.Offset, q.Limit = page.Offset, page.Limit
	res, appErr := s.List(ctx, q, Viewer{})
	if appErr != nil {
		return nil, appErr
	}
	items := make([]dto.FacePack, 0, len(res.Packs))
	for _, pack := range res.Packs {
		items = append(items, facePack(pack))
	}
	return dto.NewFaceList(items, page, res.Total), nil
}

// FaceGetPack is the site's detail query minus two things it does for a
// browser: it does not bump view_count -- a crawler should not be able to move
// the "hot" ranking -- and it does not call catalog to refresh work covers,
// because the face answers from the stored snapshot and must stay up when
// catalog is not.
func (s *Service) FaceGetPack(ctx context.Context, id uuid.UUID) (*dto.FacePackDetail, *errors.AppError) {
	pack, appErr := s.visiblePack(id, Viewer{})
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.stickers.ListByPack(pack.ID)
	if err != nil {
		return nil, errors.ErrInternal("failed to load stickers")
	}
	hydrated, appErr := s.hydrate(ctx, []model.Pack{*pack})
	if appErr != nil {
		return nil, appErr
	}

	// Both shapes are built in the one pass over rows: the site DTO because
	// distinctCatalog reads it, the face DTO because it needs the image hash,
	// which the site DTO does not carry.
	stickers := make([]dto.Sticker, 0, len(rows))
	faceStickers := make([]dto.FaceSticker, 0, len(rows))
	for _, row := range rows {
		sticker := s.stickerDTO(row)
		stickers = append(stickers, sticker)
		faceStickers = append(faceStickers, faceSticker(sticker, row.ImageHash))
	}
	works, characters := distinctCatalog(stickers)

	out := &dto.FacePackDetail{
		FacePack:   facePack(hydrated[0]),
		Stickers:   faceStickers,
		Works:      make([]dto.FaceWork, 0, len(works)),
		Characters: make([]dto.FaceCharacter, 0, len(characters)),
	}
	for _, work := range works {
		out.Works = append(out.Works, faceWork(&work))
	}
	for _, character := range characters {
		out.Characters = append(out.Characters, faceCharacter(&character))
	}
	return out, nil
}

func (s *Service) FaceGetSticker(id uuid.UUID) (*dto.FaceSticker, *errors.AppError) {
	row, appErr := s.visibleSticker(id, Viewer{})
	if appErr != nil {
		return nil, appErr
	}
	out := faceSticker(s.stickerDTO(*row), row.ImageHash)
	return &out, nil
}

// FaceCharacters is one of the two reasons this face exists: it enumerates the
// catalog character identities this site holds material for, so a caller can
// join their own catalog data against it without guessing.
func (s *Service) FaceCharacters(search string, work int64, page dto.FacePage) (*dto.FaceList[dto.FaceCharacter], *errors.AppError) {
	rows, total, err := s.stickers.CharacterIndex(repository.IndexParams{
		Search: search, Work: work, Offset: page.Offset, Limit: page.Limit,
	})
	if err != nil {
		return nil, errors.ErrInternal("failed to list characters")
	}
	items := make([]dto.FaceCharacter, 0, len(rows))
	for _, row := range rows {
		items = append(items, characterRowDTO(row))
	}
	return dto.NewFaceList(items, page, total), nil
}

func (s *Service) FaceCharacter(characterID int64) (*dto.FaceCharacter, *errors.AppError) {
	row, err := s.stickers.Character(characterID)
	if err != nil {
		return nil, errors.ErrInternal("failed to load character")
	}
	if row == nil {
		return nil, errors.ErrCharacterNotFound()
	}
	out := characterRowDTO(*row)
	return &out, nil
}

func (s *Service) FaceCharacterStickers(characterID int64, page dto.FacePage) (*dto.FaceList[dto.FaceSticker], *errors.AppError) {
	rows, total, err := s.stickers.StickersByCharacter(characterID, page.Offset, page.Limit)
	if err != nil {
		return nil, errors.ErrInternal("failed to load stickers")
	}
	items := make([]dto.FaceSticker, 0, len(rows))
	for _, row := range rows {
		items = append(items, faceSticker(s.stickerDTO(row), row.ImageHash))
	}
	return dto.NewFaceList(items, page, total), nil
}

// FaceWorks is the other catalog-id entry point: which games this site has
// material for, and how much of it.
func (s *Service) FaceWorks(search string, page dto.FacePage) (*dto.FaceList[dto.FaceWork], *errors.AppError) {
	rows, total, err := s.stickers.WorkIndex(repository.IndexParams{
		Search: search, Offset: page.Offset, Limit: page.Limit,
	})
	if err != nil {
		return nil, errors.ErrInternal("failed to list works")
	}
	items := make([]dto.FaceWork, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.FaceWork{
			Object:        "work",
			ID:            faceID(row.CatalogWorkID),
			Name:          decodeML(row.CatalogWorkName),
			CoverURL:      row.CatalogWorkCover,
			ContentRating: row.CatalogWorkRating,
			StickerCount:  row.StickerCount,
		})
	}
	return dto.NewFaceList(items, page, total), nil
}

func (s *Service) FaceTags(page dto.FacePage) (*dto.FaceList[dto.FaceTag], *errors.AppError) {
	rows, err := s.tags.Popular(page.Offset, page.Limit)
	if err != nil {
		return nil, errors.ErrInternal("failed to load tags")
	}
	total, err := s.tags.CountPopular()
	if err != nil {
		return nil, errors.ErrInternal("failed to count tags")
	}
	items := make([]dto.FaceTag, 0, len(rows))
	for _, tag := range tagDTOs(rows) {
		items = append(items, faceTag(tag))
	}
	return dto.NewFaceList(items, page, total), nil
}

// The converters below are the whole difference between the site's DTOs and
// the published contract. They are deliberately dumb: no lookups, no upstream
// calls, so a change here can only ever be a renaming, never a behaviour.

func facePack(in dto.Pack) dto.FacePack {
	out := dto.FacePack{
		Object:        "pack",
		ID:            in.ID,
		Title:         in.Title,
		Description:   in.Description,
		Official:      in.IsOfficial,
		ContentRating: faceRating(in.ContentRating),
		StickerCount:  in.StickerCount,
		ViewCount:     in.ViewCount,
		DownloadCount: in.DownloadCount,
		Work:          faceWorkPtr(in.CatalogWork),
		Tags:          make([]dto.FaceTag, 0, len(in.Tags)),
		Author: dto.FaceAuthor{
			Object: "author", ID: strconv.Itoa(in.Author.ID), Name: in.Author.Name, AvatarURL: in.Author.Avatar,
		},
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		PublishedAt: in.PublishedAt,
	}
	for _, tag := range in.Tags {
		out.Tags = append(out.Tags, faceTag(tag))
	}
	// The cover travels as URLs plus the id of the sticker it is, so a caller
	// that wants the hash or the dimensions can ask for that sticker rather
	// than be handed an invented half-record.
	if in.CoverURL != "" {
		out.Cover = &dto.FaceImage{URL: in.CoverURL, ThumbURL: in.CoverThumbURL}
		out.CoverStickerID = in.CoverStickerID
	}
	return out
}

func faceTag(in dto.Tag) dto.FaceTag {
	return dto.FaceTag{Object: "tag", Slug: in.Slug, Name: in.Name, PackCount: in.PackCount}
}

func faceSticker(in dto.Sticker, hash string) dto.FaceSticker {
	return dto.FaceSticker{
		Object:   "sticker",
		ID:       in.ID,
		PackID:   in.PackID,
		Position: in.Position,
		Image: dto.FaceImage{
			Hash: hash, URL: in.ImageURL, ThumbURL: in.ThumbURL,
			Width: in.Width, Height: in.Height,
		},
		Note:      in.Note,
		Work:      faceWorkPtr(in.CatalogWork),
		Character: faceCharacterPtr(in.CatalogCharacter),
	}
}

func faceWork(in *dto.CatalogWork) dto.FaceWork {
	return dto.FaceWork{
		Object: "work", ID: faceID(in.ID), Name: in.Name,
		CoverURL: in.CoverURL, ContentRating: in.ContentRating,
	}
}

func faceWorkPtr(in *dto.CatalogWork) *dto.FaceWork {
	if in == nil {
		return nil
	}
	out := faceWork(in)
	return &out
}

func faceCharacter(in *dto.CatalogCharacter) dto.FaceCharacter {
	return dto.FaceCharacter{
		Object: "character", ID: faceID(in.ID), Name: in.Name, ImageURL: in.ImageURL,
	}
}

func faceCharacterPtr(in *dto.CatalogCharacter) *dto.FaceCharacter {
	if in == nil {
		return nil
	}
	out := faceCharacter(in)
	return &out
}

func characterRowDTO(row repository.CharacterRow) dto.FaceCharacter {
	out := dto.FaceCharacter{
		Object:       "character",
		ID:           faceID(row.CatalogCharacterID),
		Name:         decodeML(row.CatalogCharacterName),
		ImageURL:     row.CatalogCharacterImage,
		StickerCount: row.StickerCount,
	}
	if row.CatalogWorkID != nil {
		out.Work = &dto.FaceWork{
			Object: "work", ID: faceID(*row.CatalogWorkID), Name: decodeML(row.CatalogWorkName),
		}
	}
	return out
}

// faceID spells a catalog id the way catalog's own /v2 does: a decimal
// string, so a client in a language without 64-bit integers never rounds one.
func faceID(id int64) string { return strconv.FormatInt(id, 10) }

// faceRating speaks catalog's vocabulary. This site stores 0 or 1; catalog's
// middle value (sensitive) has no equivalent here, so it is never emitted.
func faceRating(in int16) string {
	if in == model.RatingNSFW {
		return "r18"
	}
	return "all_ages"
}
