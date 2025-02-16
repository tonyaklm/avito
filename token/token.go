package token

import (
	"avito/config"
	"avito/models"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type SignedDetails struct {
	UserID uint
	jwt.StandardClaims
}

func GenerateToken(user models.User) (string, error) {
	claims := &SignedDetails{
		UserID: user.ID,

		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Minute * time.Duration(
				config.Cfg.Server.ExpirationMinutes)).Unix(),
		},
	}

	signedToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(config.Cfg.Server.SecretKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ValidateToken(signedToken string) (claims *SignedDetails, err error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&SignedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Cfg.Server.SecretKey), nil
		},
	)
	if err != nil {
		return
	}
	claims, ok := token.Claims.(*SignedDetails)
	if !ok {
		err = errors.New("the token is invalid")
		return
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("token is expired")
		return
	}
	return
}
