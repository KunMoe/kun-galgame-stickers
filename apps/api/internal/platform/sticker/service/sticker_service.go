package service

import (
	"encoding/json"

	"kun-galgame-sticker-api/internal/platform/sticker/dto"
	"kun-galgame-sticker-api/internal/platform/sticker/model"
	"kun-galgame-sticker-api/internal/platform/sticker/repository"
	"kun-galgame-sticker-api/pkg/errors"
)

var packPreviews = []struct {
	Sid        int
	PreviewPid int
}{
	{1, 1},
	{2, 18},
	{3, 35},
	{4, 52},
	{5, 69},
	{6, 6},
	{7, 12},
}

type StickerService struct {
	repo *repository.StickerRepo
}

func New(repo *repository.StickerRepo) *StickerService {
	return &StickerService{repo: repo}
}

func (s *StickerService) ListPacks() ([]dto.Pack, *errors.AppError) {
	counts, err := s.repo.CountBySid()
	if err != nil {
		return nil, errors.ErrInternal("failed to list sticker packs")
	}
	packs := make([]dto.Pack, 0, len(packPreviews))
	for _, p := range packPreviews {
		packs = append(packs, dto.Pack{
			Sid:        p.Sid,
			PreviewPid: p.PreviewPid,
			Count:      counts[p.Sid],
		})
	}
	return packs, nil
}

func (s *StickerService) GetPack(sid int) (*dto.PackDetail, *errors.AppError) {
	if sid <= 0 {
		return nil, errors.ErrNotFound("sticker pack not found")
	}
	rows, err := s.repo.FindBySid(sid)
	if err != nil {
		return nil, errors.ErrInternal("failed to load sticker pack")
	}
	if len(rows) == 0 {
		return nil, errors.ErrNotFound("sticker pack not found")
	}
	stickers := make([]dto.Sticker, 0, len(rows))
	for i := range rows {
		stickers = append(stickers, toDTO(&rows[i]))
	}
	return &dto.PackDetail{Sid: sid, Stickers: stickers}, nil
}

func (s *StickerService) GetOne(sid, pid int) (*dto.Sticker, *errors.AppError) {
	if sid <= 0 || pid <= 0 {
		return nil, errors.ErrNotFound("sticker not found")
	}
	row, err := s.repo.FindOne(sid, pid)
	if err != nil {
		return nil, errors.ErrNotFound("sticker not found")
	}
	out := toDTO(row)
	return &out, nil
}

func toDTO(row *model.Sticker) dto.Sticker {
	return dto.Sticker{
		Sid:      row.Sid,
		Pid:      row.Pid,
		Game:     asText(row.Game),
		Loli:     asText(row.Loli),
		Vndb:     row.Vndb,
		Describe: row.Describe,
	}
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
