package controllers

import (
	"avito/models"
	"avito/token"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func Auth(context *gin.Context) {
	var userData models.User
	if err := context.ShouldBindJSON(&userData); err != nil {
		context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Does not bind schema"})
		context.Abort()
		return
	}

	user, getError := models.GetUserByUsername(userData.Username)
	fmt.Println("dfghj")
	fmt.Println(user, getError)

	if getError != nil {
		if !errors.Is(getError, gorm.ErrRecordNotFound) {
			context.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Could not make search result"})
			context.Abort()
			return
		}
		user = userData
		if hashedPassword, err := models.HashPassword(user.Password); err == nil {
			user.Password = hashedPassword
		} else {
			context.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Could not hash password"})
			context.Abort()
			return
		}
		if err := user.CreateUser(); err != nil {
			context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Could not create user"})
			context.Abort()
			return
		}
	} else {
		if !user.ValidatePassword(userData.Password) {
			context.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Incorrect password"})
			context.Abort()
			return
		}
	}

	signedToken, err := token.GenerateToken(user)

	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Error generating tokens"})
	}

	tokenResponse := TokenResponse{
		SignedToken: signedToken}

	context.JSON(http.StatusOK, tokenResponse)
}
