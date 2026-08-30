package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"strings"
	"time"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"
	"kun-galgame-sticker-api/pkg/imageclient"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	maxStickersPerPack = 80
	maxPacksPerUser    = 20
	maxUploadsPerDay   = 200
	maxUploadBytes     = 10 << 20
	imagePreset        = "sticker"
	thumbVariant       = "320"
)

type StickerService struct {
	repo   *repository.StickerRepo
	images *imageclient.Client
}

func New(repo *repository.StickerRepo, images *imageclient.Client) *StickerService {
	return &StickerService{repo: repo, images: images}
}

func (s *StickerService) ListPacks() ([]dto.Pack, *errors.AppError) {
	rows, err := s.repo.ListPublishedPacks()
	if err != nil {
		return nil, errors.ErrInternal("failed to list sticker packs")
	}
	return s.toPackDTOs(rows), nil
}

func (s *StickerService) ListMine(uid int) ([]dto.Pack, *errors.AppError) {
	rows, err := s.repo.ListPacksByOwner(uid)
	if err != nil {
		return nil, errors.ErrInternal("failed to list packs")
	}
	return s.toPackDTOs(rows), nil
}

func (s *StickerService) GetPack(sid int, viewerUID int) (*dto.PackDetail, *errors.AppError) {
	pack, appErr := s.loadVisiblePack(sid, viewerUID)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.repo.FindBySid(sid)
	if err != nil {
		return nil, errors.ErrInternal("failed to load sticker pack")
	}
	stickers := make([]dto.Sticker, 0, len(rows))
	for i := range rows {
		stickers = append(stickers, s.toStickerDTO(&rows[i]))
	}
	out := s.toPackDTO(pack)
	out.Count = len(stickers)
	return &dto.PackDetail{Pack: out, Stickers: stickers}, nil
}

func (s *StickerService) GetOne(sid, pid, viewerUID int) (*dto.Sticker, *errors.AppError) {
	if _, appErr := s.loadVisiblePack(sid, viewerUID); appErr != nil {
		return nil, appErr
	}
	row, err := s.repo.FindOne(sid, pid)
	if err != nil {
		return nil, errors.ErrNotFound("sticker not found")
	}
	out := s.toStickerDTO(row)
	return &out, nil
}

func (s *StickerService) CreatePack(uid int, req dto.CreatePackRequest) (*dto.Pack, *errors.AppError) {
	title := sanitizeText(req.Title)
	if !hasAny(title) {
		return nil, errors.ErrBadRequest("title is required")
	}
	n, err := s.repo.CountPacksByOwner(uid)
	if err != nil {
		return nil, errors.ErrInternal("failed to create pack")
	}
	if n >= maxPacksPerUser {
		return nil, errors.ErrBadRequest("pack limit reached")
	}
	now := time.Now()
	row := &model.Pack{
		OwnerUID:    uid,
		Status:      model.PackDraft,
		Title:       mustJSON(title),
		Description: mustJSON(sanitizeText(req.Description)),
		PreviewPid:  1,
		Created:     now,
		Updated:     now,
	}
	if err := s.repo.CreatePack(row); err != nil {
		return nil, errors.ErrInternal("failed to create pack")
	}
	out := s.toPackDTO(row)
	return &out, nil
}

func (s *StickerService) PatchPack(sid, uid int, req dto.PatchPackRequest) (*dto.Pack, *errors.AppError) {
	pack, appErr := s.requireOwner(sid, uid)
	if appErr != nil {
		return nil, appErr
	}
	if title := sanitizeText(req.Title); hasAny(title) {
		pack.Title = mustJSON(title)
	}
	if req.Description != nil {
		pack.Description = mustJSON(sanitizeText(req.Description))
	}
	if req.PreviewPid != nil {
		if _, err := s.repo.FindOne(sid, *req.PreviewPid); err != nil {
			return nil, errors.ErrBadRequest("preview sticker not found")
		}
		pack.PreviewPid = *req.PreviewPid
	}
	pack.Updated = time.Now()
	if err := s.repo.SavePack(pack); err != nil {
		return nil, errors.ErrInternal("failed to update pack")
	}
	out := s.toPackDTO(pack)
	return &out, nil
}

func (s *StickerService) Publish(sid, uid int) (*dto.Pack, *errors.AppError) {
	pack, appErr := s.requireOwner(sid, uid)
	if appErr != nil {
		return nil, appErr
	}
	n, err := s.repo.CountStickers(sid)
	if err != nil {
		return nil, errors.ErrInternal("failed to publish pack")
	}
	if n < 1 {
		return nil, errors.ErrBadRequest("pack needs at least one sticker")
	}
	now := time.Now()
	pack.Status = model.PackPublished
	pack.Updated = now
	if pack.PublishedAt == nil {
		pack.PublishedAt = &now
	}
	if err := s.repo.SavePack(pack); err != nil {
		return nil, errors.ErrInternal("failed to publish pack")
	}
	out := s.toPackDTO(pack)
	return &out, nil
}

func (s *StickerService) Unpublish(sid, uid int) (*dto.Pack, *errors.AppError) {
	pack, appErr := s.requireOwner(sid, uid)
	if appErr != nil {
		return nil, appErr
	}
	pack.Status = model.PackHidden
	pack.Updated = time.Now()
	if err := s.repo.SavePack(pack); err != nil {
		return nil, errors.ErrInternal("failed to unpublish pack")
	}
	out := s.toPackDTO(pack)
	return &out, nil
}

func (s *StickerService) UploadImage(ctx context.Context, sid, uid int, sub string, filename string, body io.Reader, size int64) (*dto.UploadResult, *errors.AppError) {
	if _, appErr := s.requireOwner(sid, uid); appErr != nil {
		return nil, appErr
	}
	if s.images == nil {
		return nil, errors.ErrUnavailable("image service is not configured")
	}
	if size > maxUploadBytes {
		return nil, errors.ErrBadRequest("file too large")
	}
	recent, err := s.repo.CountRecentStickersByOwner(uid, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, errors.ErrInternal("failed to upload image")
	}
	if recent >= maxUploadsPerDay {
		return nil, errors.ErrBadRequest("daily upload limit reached")
	}
	if filename == "" {
		filename = "sticker.bin"
	}
	result, err := s.images.UploadWithSub(ctx, body, filename, imagePreset, sub)
	if err != nil {
		return nil, mapImageErr(err)
	}
	return &dto.UploadResult{
		Hash:        result.Hash,
		URL:         result.URL,
		VariantURLs: result.VariantURLs,
		Width:       result.Width,
		Height:      result.Height,
	}, nil
}

func (s *StickerService) AddSticker(sid, uid int, req dto.CreateStickerRequest) (*dto.Sticker, *errors.AppError) {
	if _, appErr := s.requireOwner(sid, uid); appErr != nil {
		return nil, appErr
	}
	hash := strings.TrimSpace(req.ImageHash)
	if len(hash) != 64 {
		return nil, errors.ErrBadRequest("image_hash is required")
	}
	n, err := s.repo.CountStickers(sid)
	if err != nil {
		return nil, errors.ErrInternal("failed to add sticker")
	}
	if n >= maxStickersPerPack {
		return nil, errors.ErrBadRequest("sticker limit reached")
	}
	pid, err := s.repo.NextPid(sid)
	if err != nil {
		return nil, errors.ErrInternal("failed to add sticker")
	}
	now := time.Now()
	row := &model.Sticker{
		Sid:       sid,
		Pid:       pid,
		Game:      mustJSON(sanitizeText(req.Game)),
		Loli:      mustJSON(sanitizeText(req.Loli)),
		Vndb:      req.Vndb,
		Describe:  strings.TrimSpace(req.Describe),
		ImageHash: &hash,
		Created:   now,
		Updated:   now,
	}
	if err := s.repo.CreateSticker(row); err != nil {
		return nil, errors.ErrInternal("failed to add sticker")
	}
	out := s.toStickerDTO(row)
	return &out, nil
}

func (s *StickerService) PatchSticker(sid, pid, uid int, req dto.PatchStickerRequest) (*dto.Sticker, *errors.AppError) {
	if _, appErr := s.requireOwner(sid, uid); appErr != nil {
		return nil, appErr
	}
	row, err := s.repo.FindOne(sid, pid)
	if err != nil {
		return nil, errors.ErrNotFound("sticker not found")
	}
	if req.ImageHash != nil {
		hash := strings.TrimSpace(*req.ImageHash)
		if len(hash) != 64 {
			return nil, errors.ErrBadRequest("image_hash is invalid")
		}
		row.ImageHash = &hash
	}
	if req.Game != nil {
		row.Game = mustJSON(sanitizeText(req.Game))
	}
	if req.Loli != nil {
		row.Loli = mustJSON(sanitizeText(req.Loli))
	}
	if req.Vndb != nil {
		row.Vndb = *req.Vndb
	}
	if req.Describe != nil {
		row.Describe = strings.TrimSpace(*req.Describe)
	}
	row.Updated = time.Now()
	if err := s.repo.SaveSticker(row); err != nil {
		return nil, errors.ErrInternal("failed to update sticker")
	}
	out := s.toStickerDTO(row)
	return &out, nil
}

func (s *StickerService) DeleteSticker(sid, pid, uid int) *errors.AppError {
	if _, appErr := s.requireOwner(sid, uid); appErr != nil {
		return appErr
	}
	if _, err := s.repo.FindOne(sid, pid); err != nil {
		return errors.ErrNotFound("sticker not found")
	}
	if err := s.repo.DeleteSticker(sid, pid); err != nil {
		return errors.ErrInternal("failed to delete sticker")
	}
	return nil
}

func (s *StickerService) PingHashes(ctx context.Context) {
	if s.images == nil {
		return
	}
	hashes, err := s.repo.ListImageHashes()
	if err != nil || len(hashes) == 0 {
		return
	}
	const batch = 1000
	for i := 0; i < len(hashes); i += batch {
		end := i + batch
		if end > len(hashes) {
			end = len(hashes)
		}
		if _, err := s.images.ReferencePing(ctx, hashes[i:end]); err != nil {
			return
		}
	}
}

func (s *StickerService) loadVisiblePack(sid, viewerUID int) (*model.Pack, *errors.AppError) {
	if sid <= 0 {
		return nil, errors.ErrNotFound("sticker pack not found")
	}
	pack, err := s.repo.GetPack(sid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound("sticker pack not found")
		}
		return nil, errors.ErrInternal("failed to load sticker pack")
	}
	if pack.Status == model.PackPublished {
		return pack, nil
	}
	if viewerUID > 0 && pack.OwnerUID == viewerUID {
		return pack, nil
	}
	return nil, errors.ErrNotFound("sticker pack not found")
}

func (s *StickerService) requireOwner(sid, uid int) (*model.Pack, *errors.AppError) {
	if sid <= 0 {
		return nil, errors.ErrNotFound("sticker pack not found")
	}
	pack, err := s.repo.GetPack(sid)
	if err != nil {
		return nil, errors.ErrNotFound("sticker pack not found")
	}
	if pack.OwnerUID != uid {
		return nil, errors.ErrForbidden("not the pack owner")
	}
	return pack, nil
}

func (s *StickerService) toPackDTOs(rows []model.Pack) []dto.Pack {
	counts, _ := s.repo.CountBySid()
	out := make([]dto.Pack, 0, len(rows))
	for i := range rows {
		item := s.toPackDTO(&rows[i])
		item.Count = counts[rows[i].ID]
		out = append(out, item)
	}
	return out
}

func (s *StickerService) toPackDTO(row *model.Pack) dto.Pack {
	hash := s.repo.PreviewHash(row.ID, row.PreviewPid)
	imageURL, thumbURL := s.urls(row.ID, row.PreviewPid, hash)
	preview := thumbURL
	if preview == "" {
		preview = imageURL
	}
	var published *string
	if row.PublishedAt != nil {
		v := row.PublishedAt.UTC().Format(time.RFC3339)
		published = &v
	}
	return dto.Pack{
		Sid:         row.ID,
		OwnerUID:    row.OwnerUID,
		Status:      row.Status,
		Title:       asText(row.Title),
		Description: asText(row.Description),
		PreviewPid:  row.PreviewPid,
		PreviewURL:  preview,
		PublishedAt: published,
	}
}

func (s *StickerService) toStickerDTO(row *model.Sticker) dto.Sticker {
	hash := ""
	if row.ImageHash != nil {
		hash = strings.TrimSpace(*row.ImageHash)
	}
	imageURL, thumbURL := s.urls(row.Sid, row.Pid, hash)
	return dto.Sticker{
		Sid:       row.Sid,
		Pid:       row.Pid,
		Game:      asText(row.Game),
		Loli:      asText(row.Loli),
		Vndb:      row.Vndb,
		Describe:  row.Describe,
		ImageHash: hash,
		ImageURL:  imageURL,
		ThumbURL:  thumbURL,
	}
}

func (s *StickerService) urls(sid, pid int, hash string) (string, string) {
	if hash != "" && s.images != nil {
		return s.images.MainURL(hash), s.images.VariantURL(hash, thumbVariant)
	}
	fallback := fmt.Sprintf("/stickers/KUNgal%d/%d.webp", sid, pid)
	return fallback, fallback
}

func asText(raw []byte) dto.MultilingualText {
	if len(raw) == 0 {
		return dto.MultilingualText{}
	}
	var out dto.MultilingualText
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return dto.MultilingualText{}
	}
	return out
}

func sanitizeText(in dto.MultilingualText) dto.MultilingualText {
	out := dto.MultilingualText{}
	for k, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if len(v) > 200 {
			v = v[:200]
		}
		out[k] = v
	}
	return out
}

func hasAny(in dto.MultilingualText) bool {
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func mustJSON(in dto.MultilingualText) datatypes.JSON {
	if in == nil {
		in = dto.MultilingualText{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

func mapImageErr(err error) *errors.AppError {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, imageclient.ErrQuotaExceeded):
		return errors.ErrBadRequest("image quota exceeded")
	case stderrors.Is(err, imageclient.ErrMIMEDenied):
		return errors.ErrBadRequest("unsupported image type")
	case stderrors.Is(err, imageclient.ErrUnauthorized):
		return errors.ErrUnavailable("image service rejected upload")
	default:
		return errors.ErrUnavailable("image service temporarily unavailable")
	}
}
