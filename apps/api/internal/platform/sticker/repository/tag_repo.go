package repository

import (
	"encoding/json"
	"strings"
	"unicode"

	"kun-galgame-sticker-api/internal/platform/sticker/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepo struct{ db *gorm.DB }

func NewTagRepo(db *gorm.DB) *TagRepo { return &TagRepo{db: db} }

// Popular pages the tags in use, most used first. The slug tiebreaker keeps
// the order total, so paging by offset neither repeats nor skips a tag.
func (r *TagRepo) Popular(offset, limit int) ([]model.Tag, error) {
	var rows []model.Tag
	err := r.inUse().
		Order("pack_count DESC, slug ASC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, err
}

// CountPopular counts what Popular pages through.
func (r *TagRepo) CountPopular() (int64, error) {
	var total int64
	err := r.inUse().Count(&total).Error
	return total, err
}

func (r *TagRepo) inUse() *gorm.DB {
	return r.db.Model(&model.Tag{}).Where("pack_count > 0")
}

func (r *TagRepo) BySlug(slug string) (*model.Tag, error) {
	var row model.Tag
	if err := r.db.First(&row, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ForPacks loads the tags of a whole page of packs in one query.
func (r *TagRepo) ForPacks(packIDs []uuid.UUID) (map[uuid.UUID][]model.Tag, error) {
	out := make(map[uuid.UUID][]model.Tag, len(packIDs))
	if len(packIDs) == 0 {
		return out, nil
	}
	type row struct {
		PackID uuid.UUID
		model.Tag
	}
	var rows []row
	err := r.db.Table("pack_tag").
		Select("pack_tag.pack_id, tag.*").
		Joins("JOIN tag ON tag.id = pack_tag.tag_id").
		Where("pack_tag.pack_id IN ?", packIDs).
		Order("tag.slug ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.PackID] = append(out[item.PackID], item.Tag)
	}
	return out, nil
}

// SetPackTags replaces a pack's tags, creating any label that does not exist
// yet. Labels are free text from the author; the slug is what deduplicates
// them, so "Cat Girl" and "cat girl" land on one tag.
func (r *TagRepo) SetPackTags(packID uuid.UUID, labels []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var previous []uuid.UUID
		if err := tx.Table("pack_tag").Where("pack_id = ?", packID).
			Pluck("tag_id", &previous).Error; err != nil {
			return err
		}

		tags := make([]model.Tag, 0, len(labels))
		seen := map[string]bool{}
		for _, label := range labels {
			slug := Slugify(label)
			if slug == "" || seen[slug] {
				continue
			}
			seen[slug] = true

			name, err := json.Marshal(map[string]string{"und": strings.TrimSpace(label)})
			if err != nil {
				return err
			}
			tag := model.Tag{Slug: slug, Name: datatypes.JSON(name)}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "slug"}},
				DoUpdates: clause.Assignments(map[string]any{"slug": slug}),
			}).Create(&tag).Error; err != nil {
				return err
			}
			tags = append(tags, tag)
		}

		if err := tx.Where("pack_id = ?", packID).Delete(&model.PackTag{}).Error; err != nil {
			return err
		}
		touched := append([]uuid.UUID{}, previous...)
		for _, tag := range tags {
			if err := tx.Create(&model.PackTag{PackID: packID, TagID: tag.ID}).Error; err != nil {
				return err
			}
			touched = append(touched, tag.ID)
		}
		return recount(tx, touched)
	})
}

func (r *TagRepo) Recount(tagIDs []uuid.UUID) error { return recount(r.db, tagIDs) }

// RecountForPack refreshes the counters of every tag on a pack. Publishing or
// hiding a pack changes what the tag cloud should show, and pack_count only
// counts published packs.
func (r *TagRepo) RecountForPack(packID uuid.UUID) error {
	var ids []uuid.UUID
	if err := r.db.Table("pack_tag").Where("pack_id = ?", packID).Pluck("tag_id", &ids).Error; err != nil {
		return err
	}
	return recount(r.db, ids)
}

func recount(tx *gorm.DB, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}
	return tx.Exec(`
		UPDATE tag SET pack_count = (
			SELECT count(*) FROM pack_tag pt
			JOIN pack p ON p.id = pt.pack_id
			WHERE pt.tag_id = tag.id AND p.status = ?
		) WHERE id IN ?`, model.PackPublished, tagIDs).Error
}

// Slugify keeps CJK as-is -- a Chinese tag has no ASCII transliteration worth
// guessing -- and collapses everything that is not a letter or digit into "-".
func Slugify(label string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
