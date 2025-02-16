package middleware

import (
	"avito/controllers"
	"avito/token"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Authenticate(context *gin.Context) {

	clientToken := context.Request.Header.Get("Authorization")
	if clientToken == "" {
		context.JSON(http.StatusUnauthorized,
			controllers.ErrorResponse{Error: fmt.Sprintf("No authorization header provided")})
		context.Abort()
		return
	}
	claims, err := token.ValidateToken(clientToken)
	if err != nil {
		context.JSON(http.StatusUnauthorized, controllers.ErrorResponse{Error: err.Error()})
		context.Abort()
		return
	}

	context.Set("user_id", claims.UserID)
	context.Next()
}
