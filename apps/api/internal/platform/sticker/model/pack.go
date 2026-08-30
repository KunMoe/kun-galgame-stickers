package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	PackDraft     int16 = 0
	PackPublished int16 = 1
	PackHidden    int16 = 2
)

type Pack struct {
	ID          int            `gorm:"primaryKey"`
	OwnerUID    int            `gorm:"column:owner_uid"`
	Status      int16          `gorm:"column:status"`
	Title       datatypes.JSON `gorm:"column:title;type:jsonb"`
	Description datatypes.JSON `gorm:"column:description;type:jsonb"`
	PreviewPid  int            `gorm:"column:preview_pid"`
	Created     time.Time      `gorm:"column:created"`
	Updated     time.Time      `gorm:"column:updated"`
	PublishedAt *time.Time     `gorm:"column:published_at"`
}

func (Pack) TableName() string { return "sticker_pack" }
