package models

import "gorm.io/gorm"

type Transaction struct {
	gorm.Model
	ID         uint    `gorm:"primary_key" autoIncrement:"true"`
	SenderID   uint    `json:"sender_id" binding:"required"`
	Sender     User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL; foreignKey:SenderID"`
	ReceiverID uint    `json:"receiver_id" binding:"required"`
	Receiver   User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL; foreignKey:ReceiverID"`
	Amount     float32 `gorm:"check:amount > 0;" json:"amount" binding:"required"`
}
