package config

import "os"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}
type ServerConfig struct {
	SecretKey         string
	Port              string
	ExpirationMinutes int
}
type DatabaseConfig struct {
	Host         string
	Username     string
	Password     string
	DatabaseName string
	Port         string
}

var Cfg = Config{}

func (config *Config) Init() {
	config.Server = ServerConfig{
		SecretKey:         os.Getenv("SECRET_KEY"),
		Port:              os.Getenv("SERVER_PORT"),
		ExpirationMinutes: 50,
	}
	config.Database = DatabaseConfig{

		Host:         os.Getenv("DATABASE_HOST"),
		Username:     os.Getenv("DATABASE_USER"),
		Password:     os.Getenv("DATABASE_PASSWORD"),
		DatabaseName: os.Getenv("DATABASE_NAME"),
		Port:         os.Getenv("DATABASE_PORT"),
	}
}
