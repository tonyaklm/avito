package controllers

import (
	"avito/database"
	"avito/models"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"net/http"
)

func SendCoin(context *gin.Context) {
	var user, sendTo models.User
	var err error
	var payload SendToPayload

	if err = context.ShouldBindJSON(&payload); err != nil {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		context.Abort()
		return
	}

	if userId, ok := context.Get("user_id"); !ok {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	} else if res := database.PostgresDB.Where("ID = ?", userId).First(&user); res.Error != nil {
		context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization failed"})
		context.Abort()
		return
	}

	if res := database.PostgresDB.Where("Username = ?", payload.ToUser).First(&sendTo); res.Error != nil {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Incorrect receiver's username"})
		context.Abort()
		return
	}
	if sendTo.ID == user.ID {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Incorrect receiver's username"})
		context.Abort()
		return
	}

	if user.Balance < payload.Amount {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Insufficient funds to complete the transaction"})
		context.Abort()
		return
	}

	transaction := models.Transaction{SenderID: user.ID, ReceiverID: sendTo.ID, Amount: payload.Amount}
	err = database.PostgresDB.Transaction(func(tx *gorm.DB) error {
		if err = tx.Create(&transaction).Error; err != nil {
			return err
		}
		result := tx.Model(&user).Update("Balance", user.Balance-payload.Amount)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("сould not update sender's balance")
		}

		result = tx.Model(&sendTo).Update("Balance", sendTo.Balance+payload.Amount)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("сould not update receiver's balance")
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" || errors.Is(err, gorm.ErrCheckConstraintViolated) {
			context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Incorrect amount of coins to complete the transaction"})
			context.Abort()
			return
		}
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Could not send coins"})
		context.Abort()
		return
	}
	context.JSON(http.StatusOK, gin.H{})
}
