package controllers

import (
	"avito/database"
	"avito/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetInfo(context *gin.Context) {
	var user models.User
	var inventory []InventorySchema
	var received []ReceivedSchema
	var sent []SentSchema

	if userId, ok := context.Get("user_id"); !ok {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	} else if res := database.PostgresDB.Where("ID = ?", userId).First(&user); res.Error != nil {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	}

	err := database.PostgresDB.Model(models.Purchase{}).
		Select("items.item_name as type, count(purchases.id) as quantity").
		Joins("left join items on items.id = purchases.item_id").
		Where("purchases.user_id = ?", user.ID).
		Group("items.item_name").Scan(&inventory).Error
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		context.Abort()
		return
	}

	err = database.PostgresDB.Model(models.Transaction{}).
		Select("users.username as from_user, amount as amount").
		Joins("left join users on users.id = transactions.sender_id").
		Where("transactions.receiver_id = ?", user.ID).
		Order("transactions.created_at").Scan(&received).Error
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		context.Abort()
		return
	}
	err = database.PostgresDB.Model(models.Transaction{}).
		Select("users.username as to_user, amount as amount").
		Joins("left join users on users.id = transactions.receiver_id").
		Where("transactions.sender_id = ?", user.ID).
		Order("transactions.created_at").Scan(&sent).Error
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		context.Abort()
		return
	}

	var info = InfoSchema{
		user.Balance,
		inventory,
		HistorySchema{received, sent},
	}
	context.JSON(http.StatusOK, info)

}
