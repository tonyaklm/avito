package models

import (
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ID       uint    `gorm:"primary_key" autoIncrement:"true"`
	ItemName string  `gorm:"index:idx_item;unique;not null;" json:"item_name" binding:"required"`
	Price    float32 `gorm:"check:price >= 0"`
}
