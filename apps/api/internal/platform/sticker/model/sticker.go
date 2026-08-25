package model

import (
	"time"

	"gorm.io/datatypes"
)

type Sticker struct {
	ID       int            `gorm:"primaryKey"`
	Sid      int            `gorm:"column:sid"`
	Pid      int            `gorm:"column:pid"`
	Src      string         `gorm:"column:src"`
	Game     datatypes.JSON `gorm:"column:game;type:jsonb"`
	Loli     datatypes.JSON `gorm:"column:loli;type:jsonb"`
	Vndb     int            `gorm:"column:vndb"`
	Describe string         `gorm:"column:describe"`
	Created  time.Time      `gorm:"column:created"`
	Updated  time.Time      `gorm:"column:updated"`
}

func (Sticker) TableName() string { return "sticker" }
