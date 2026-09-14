package service

import (
	"context"
	stderrors "errors"
	"io"
	"strings"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/imageclient"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *Service) GetSticker(id uuid.UUID, v Viewer) (*dto.Sticker, *errors.AppError) {
	row, appErr := s.visibleSticker(id, v)
	if appErr != nil {
		return nil, appErr
	}
	out := s.stickerDTO(*row)
	return &out, nil
}

func (s *Service) visibleSticker(id uuid.UUID, v Viewer) (*model.Sticker, *errors.AppError) {
	row, err := s.stickers.Get(id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrStickerNotFound()
		}
		return nil, errors.ErrInternal("failed to load sticker")
	}
	if _, appErr := s.visiblePack(row.PackID, v); appErr != nil {
		return nil, errors.ErrStickerNotFound()
	}
	return row, nil
}

func (s *Service) UploadImage(
	ctx context.Context,
	packID uuid.UUID,
	v Viewer,
	filename string,
	body io.Reader,
	size int64,
) (*dto.UploadResult, *errors.AppError) {
	if _, appErr := s.editablePack(packID, v); appErr != nil {
		return nil, appErr
	}
	if size > MaxUploadBytes {
		return nil, errors.ErrFileTooLarge(MaxUploadBytes)
	}
	if s.images == nil {
		return nil, errors.ErrImageUnavailable()
	}

	since := nowMinusDay()
	recent, err := s.stickers.CountRecentByOwner(v.UID, since)
	if err != nil {
		return nil, errors.ErrInternal("failed to check the upload quota")
	}
	if recent >= MaxUploadsPerDay {
		return nil, errors.ErrUploadDailyLimit(MaxUploadsPerDay)
	}

	result, err := s.images.UploadWithSub(ctx, body, filename, imagePreset, v.Sub)
	if err != nil {
		return nil, mapImageErr(err)
	}
	main, thumb := s.urls(result.Hash)
	return &dto.UploadResult{
		Hash:     result.Hash,
		ImageURL: main,
		ThumbURL: thumb,
		Width:    result.Width,
		Height:   result.Height,
	}, nil
}

func (s *Service) AddSticker(ctx context.Context, packID uuid.UUID, v Viewer, req dto.CreateStickerRequest) (*dto.Sticker, *errors.AppError) {
	if _, appErr := s.editablePack(packID, v); appErr != nil {
		return nil, appErr
	}
	hash := strings.TrimSpace(req.ImageHash)
	if !isImageHash(hash) {
		return nil, errors.ErrInvalidParams("image_hash must be a 64 character hex digest")
	}

	work, appErr := s.resolveWork(ctx, req.CatalogWorkID)
	if appErr != nil {
		return nil, appErr
	}
	characterID, characterName, characterImage, appErr := s.resolveCharacter(ctx, req.CatalogCharacterID)
	if appErr != nil {
		return nil, appErr
	}

	row := &model.Sticker{
		ImageHash:             hash,
		Width:                 req.Width,
		Height:                req.Height,
		Game:                  sanitizeML(req.Game),
		CharacterName:         sanitizeML(req.CharacterName),
		VndbID:                positive(req.VndbID),
		CatalogWorkID:         work.ID,
		CatalogWorkName:       work.Name,
		CatalogWorkRating:     work.Rating,
		CatalogCharacterID:    characterID,
		CatalogCharacterName:  characterName,
		CatalogCharacterImage: characterImage,
	}
	if req.Note != nil {
		row.Note = truncateRunes(strings.TrimSpace(*req.Note), MaxTextRunes)
	}

	if _, err := s.stickers.Append(packID, row, MaxStickersPerPack); err != nil {
		if repository.IsStickerLimit(err) {
			return nil, errors.ErrStickerLimit(MaxStickersPerPack)
		}
		return nil, errors.ErrInternal("failed to add sticker")
	}

	// The first sticker becomes the cover, so a pack is never coverless.
	if pack, err := s.packs.Get(packID); err == nil && pack.CoverStickerID == nil {
		pack.CoverStickerID = &row.ID
		_ = s.packs.Save(pack)
	}

	if work.ID != nil {
		s.syncPackSearchText(packID)
	}
	out := s.stickerDTO(*row)
	return &out, nil
}

func (s *Service) PatchSticker(
	ctx context.Context,
	packID, stickerID uuid.UUID,
	v Viewer,
	req dto.PatchStickerRequest,
) (*dto.Sticker, *errors.AppError) {
	if _, appErr := s.editablePack(packID, v); appErr != nil {
		return nil, appErr
	}
	row, err := s.stickers.Get(stickerID)
	if err != nil || row.PackID != packID {
		return nil, errors.ErrStickerNotFound()
	}

	if req.Game != nil {
		row.Game = sanitizeML(*req.Game)
	}
	if req.CharacterName != nil {
		row.CharacterName = sanitizeML(*req.CharacterName)
	}
	if req.VndbID != nil {
		row.VndbID = positive(req.VndbID)
	}
	if req.Note != nil {
		row.Note = truncateRunes(strings.TrimSpace(*req.Note), MaxTextRunes)
	}
	gameChanged := false
	if req.CatalogWorkID != nil {
		work, appErr := s.resolveWork(ctx, req.CatalogWorkID)
		if appErr != nil {
			return nil, appErr
		}
		row.CatalogWorkID = work.ID
		row.CatalogWorkName = work.Name
		row.CatalogWorkRating = work.Rating
		gameChanged = true
	}
	if req.CatalogCharacterID != nil {
		characterID, characterName, characterImage, appErr := s.resolveCharacter(ctx, req.CatalogCharacterID)
		if appErr != nil {
			return nil, appErr
		}
		row.CatalogCharacterID = characterID
		row.CatalogCharacterName = characterName
		row.CatalogCharacterImage = characterImage
	}
	if err := s.stickers.Save(row); err != nil {
		return nil, errors.ErrInternal("failed to save sticker")
	}
	if gameChanged {
		s.syncPackSearchText(packID)
	}
	out := s.stickerDTO(*row)
	return &out, nil
}

func (s *Service) DeleteSticker(packID, stickerID uuid.UUID, v Viewer) *errors.AppError {
	pack, appErr := s.editablePack(packID, v)
	if appErr != nil {
		return appErr
	}
	if appErr := refuseOfficialDelete(pack); appErr != nil {
		return appErr
	}
	if err := s.stickers.Delete(packID, stickerID); err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.ErrStickerNotFound()
		}
		return errors.ErrInternal("failed to delete sticker")
	}

	s.syncPackSearchText(packID)

	if pack.CoverStickerID != nil && *pack.CoverStickerID == stickerID {
		fresh, err := s.stickers.ListByPack(packID)
		if err == nil {
			pack.CoverStickerID = nil
			if len(fresh) > 0 {
				pack.CoverStickerID = &fresh[0].ID
			}
			_ = s.packs.Save(pack)
		}
	}
	return nil
}

func (s *Service) Reorder(packID uuid.UUID, v Viewer, rawIDs []string) *errors.AppError {
	if _, appErr := s.editablePack(packID, v); appErr != nil {
		return appErr
	}
	ids := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id, ok := parseUUID(raw)
		if !ok {
			return errors.ErrInvalidParams("sticker_ids must all be uuids")
		}
		ids = append(ids, id)
	}
	if err := s.stickers.Reorder(packID, ids); err != nil {
		if repository.IsReorderMismatch(err) {
			return errors.ErrInvalidParams("sticker_ids must list every sticker in the pack exactly once")
		}
		return errors.ErrInternal("failed to reorder stickers")
	}
	return nil
}

func mapImageErr(err error) *errors.AppError {
	switch {
	case stderrors.Is(err, imageclient.ErrQuotaExceeded):
		return errors.ErrUploadDailyLimit(MaxUploadsPerDay)
	case stderrors.Is(err, imageclient.ErrModerationRejected):
		return errors.ErrModeration()
	case stderrors.Is(err, imageclient.ErrMIMEDenied):
		return errors.ErrImageRejected("only png, jpeg and webp images are accepted")
	default:
		// Everything else, credential failures included, is the image service
		// being unusable from here. The integration guide forbids falling back
		// to local storage, so this is a 503 and never a silent success.
		return errors.ErrImageUnavailable()
	}
}

func isImageHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, r := range hash {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func positive(in *int) *int {
	if in == nil || *in <= 0 {
		return nil
	}
	return in
}

func truncateRunes(value string, max int) string {
	if runes := []rune(value); len(runes) > max {
		return string(runes[:max])
	}
	return value
}
