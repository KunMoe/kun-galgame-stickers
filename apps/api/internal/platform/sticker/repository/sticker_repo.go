package repository

import (
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

func (r *StickerRepo) CountBySid() (map[int]int, error) {
	type row struct {
		Sid   int
		Count int
	}
	var rows []row
	err := r.db.Model(&model.Sticker{}).
		Select("sid, count(*) as count").
		Group("sid").
		Order("sid ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.Sid] = r.Count
	}
	return out, nil
}
