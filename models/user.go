package models

import (
	"avito/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uint    `gorm:"primary_key" autoIncrement:"true"`
	Username string  `gorm:"index:idx_username;unique;not null;" json:"username" binding:"required"`
	Password string  `gorm:"unique;not null;" json:"password" binding:"required"`
	Balance  float32 `gorm:"default:1000; check:balance >= 0" json:"-"`
}

func GetUserByUsername(username string) (User, error) {
	var user User
	if res := database.PostgresDB.Where("Username = ?", username).First(&user); res.Error != nil {
		return User{}, res.Error
	}
	return user, nil
}

func (user *User) CreateUser() error {
	res := database.PostgresDB.Create(&user)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (user *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}
