package models

import "gorm.io/gorm"

type Purchase struct {
	gorm.Model
	ID     uint    `gorm:"primary_key" autoIncrement:"true"`
	ItemID uint    `json:"item_id" binding:"required"`
	Item   Item    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL; foreignKey:ItemID"`
	UserID uint    `json:"user_id" binding:"required"`
	User   User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL; foreignKey:UserID"`
	Price  float32 `gorm:"check:price >= 0; not null" json:"price" binding:"required"`
}
