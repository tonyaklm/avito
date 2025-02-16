package main

import (
	"avito/config"
	"avito/controllers"
	"avito/database"
	"avito/middleware"
	"avito/models"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"io/ioutil"
)

func initRouter(api *gin.RouterGroup) {

	api.GET("/healthcheck", func(c *gin.Context) {})
	api.POST("/auth", controllers.Auth)
	api.Use(middleware.Authenticate)
	{
		api.GET("/buy/:item", controllers.BuyItem)
		api.POST("/sendCoin", controllers.SendCoin)
		api.GET("/info", controllers.GetInfo)
	}
}
func MigrateDB() error {
	if err := database.PostgresDB.AutoMigrate(&models.User{}, &models.Item{}, &models.Transaction{}, &models.Purchase{}); err != nil {
		return err
	}
	return nil
}
func LoadItems() error {
	content, readErr := ioutil.ReadFile("data/items.json")
	if readErr != nil {
		return readErr
	}
	var items []models.Item
	if err := json.Unmarshal(content, &items); err != nil {
		return err
	}
	for _, item := range items {
		if err := database.PostgresDB.Create(&item).Error; err != nil {
			var pgErr *pgconn.PgError
			if !errors.Is(err, gorm.ErrDuplicatedKey) && (errors.As(err, &pgErr) && pgErr.Code != "23505") {
				return err
			}
		}
	}
	return nil
}

func main() {
	config.Cfg.Init()
	if err := database.InitDatabase(); err != nil {
		panic(err)
	}
	if err := MigrateDB(); err != nil {
		panic(err)
	}
	if err := LoadItems(); err != nil {
		panic(err)
	}
	r := gin.Default()
	api := r.Group("/api")
	initRouter(api)

	if err := r.Run(fmt.Sprintf(":%s", config.Cfg.Server.Port)); err != nil {
		panic("[Error] failed to start Gin server due to: " + err.Error())
	}
}
