package repository

import (
	"time"

	"kun-galgame-sticker-api/internal/platform/sticker/model"

	"gorm.io/gorm"
)

type StickerRepo struct {
	db *gorm.DB
}

func New(db *gorm.DB) *StickerRepo {
	return &StickerRepo{db: db}
}

func (r *StickerRepo) FindBySid(sid int) ([]model.Sticker, error) {
	var rows []model.Sticker
	err := r.db.Where("sid = ?", sid).Order("pid ASC").Find(&rows).Error
	return rows, err
}

func (r *StickerRepo) FindOne(sid, pid int) (*model.Sticker, error) {
	var row model.Sticker
	err := r.db.Where("sid = ? AND pid = ?", sid, pid).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *StickerRepo) GetPack(id int) (*model.Pack, error) {
	var row model.Pack
	err := r.db.First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *StickerRepo) ListPublishedPacks() ([]model.Pack, error) {
	var rows []model.Pack
	err := r.db.Where("status = ?", model.PackPublished).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *StickerRepo) ListPacksByOwner(uid int) ([]model.Pack, error) {
	var rows []model.Pack
	err := r.db.Where("owner_uid = ?", uid).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (r *StickerRepo) CountPacksByOwner(uid int) (int64, error) {
	var n int64
	err := r.db.Model(&model.Pack{}).Where("owner_uid = ?", uid).Count(&n).Error
	return n, err
}

func (r *StickerRepo) CountBySid() (map[int]int, error) {
	type row struct {
		Sid   int
		Count int
	}
	var rows []row
	err := r.db.Model(&model.Sticker{}).
		Select("sid, count(*) as count").
		Group("sid").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]int, len(rows))
	for _, item := range rows {
		out[item.Sid] = item.Count
	}
	return out, nil
}

func (r *StickerRepo) CountStickers(sid int) (int64, error) {
	var n int64
	err := r.db.Model(&model.Sticker{}).Where("sid = ?", sid).Count(&n).Error
	return n, err
}

func (r *StickerRepo) NextPid(sid int) (int, error) {
	var maxPid *int
	err := r.db.Model(&model.Sticker{}).Where("sid = ?", sid).Select("max(pid)").Scan(&maxPid).Error
	if err != nil {
		return 0, err
	}
	if maxPid == nil {
		return 1, nil
	}
	return *maxPid + 1, nil
}

func (r *StickerRepo) CreatePack(row *model.Pack) error {
	return r.db.Create(row).Error
}

func (r *StickerRepo) SavePack(row *model.Pack) error {
	return r.db.Save(row).Error
}

func (r *StickerRepo) CreateSticker(row *model.Sticker) error {
	return r.db.Create(row).Error
}

func (r *StickerRepo) SaveSticker(row *model.Sticker) error {
	return r.db.Save(row).Error
}

func (r *StickerRepo) DeleteSticker(sid, pid int) error {
	return r.db.Where("sid = ? AND pid = ?", sid, pid).Delete(&model.Sticker{}).Error
}

func (r *StickerRepo) PreviewHash(sid, pid int) string {
	var hash *string
	_ = r.db.Model(&model.Sticker{}).
		Where("sid = ? AND pid = ?", sid, pid).
		Select("image_hash").
		Scan(&hash).Error
	if hash == nil {
		return ""
	}
	return *hash
}

func (r *StickerRepo) ListImageHashes() ([]string, error) {
	var hashes []string
	err := r.db.Model(&model.Sticker{}).
		Where("image_hash IS NOT NULL AND image_hash <> ''").
		Pluck("image_hash", &hashes).Error
	return hashes, err
}

func (r *StickerRepo) CountRecentStickersByOwner(uid int, since time.Time) (int64, error) {
	var n int64
	err := r.db.Table("sticker").
		Joins("JOIN sticker_pack ON sticker_pack.id = sticker.sid").
		Where("sticker_pack.owner_uid = ? AND sticker.created >= ?", uid, since).
		Count(&n).Error
	return n, err
}

func (r *StickerRepo) ListWithoutHash() ([]model.Sticker, error) {
	var rows []model.Sticker
	err := r.db.Where("image_hash IS NULL OR btrim(image_hash) = ''").
		Order("sid ASC, pid ASC").
		Find(&rows).Error
	return rows, err
}

func (r *StickerRepo) SetImageHash(sid, pid int, hash string) error {
	return r.db.Model(&model.Sticker{}).
		Where("sid = ? AND pid = ?", sid, pid).
		Update("image_hash", hash).Error
}
