package controllers

import (
	"avito/database"
	"avito/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func BuyItem(context *gin.Context) {
	var item models.Item
	var user models.User
	var err error

	itemName := context.Param("item")

	if userId, ok := context.Get("user_id"); !ok {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	} else if res := database.PostgresDB.Where("ID = ?", userId).First(&user); res.Error != nil {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	}

	if res := database.PostgresDB.Where("item_name = ?", itemName).First(&item); res.Error != nil {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Could not find item"})
		context.Abort()
		return
	}

	if user.Balance < item.Price {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Insufficient funds to complete the transaction"})
		context.Abort()
		return
	}

	purchase := models.Purchase{ItemID: item.ID, UserID: user.ID, Price: item.Price}
	err = database.PostgresDB.Transaction(func(tx *gorm.DB) error {
		if err = tx.Create(&purchase).Error; err != nil {
			return err
		}
		result := tx.Model(&user).Update("Balance", user.Balance-item.Price)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("сould not update user's balance")
		}
		return nil
	})
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Could not make a transaction"})
		context.Abort()
		return
	}
	context.JSON(http.StatusOK, gin.H{})
}
