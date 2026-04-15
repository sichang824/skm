package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	LogLevel  string
	LogFormat string
	DBDriver  string
	DBDSN     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "console")
	dbDriver := getEnv("DB_DRIVER", "sqlite")
	dsn := getEnv("DB_DSN", "./data/app.db")

	return &Config{
		Port:      port,
		LogLevel:  logLevel,
		LogFormat: logFormat,
		DBDriver:  dbDriver,
		DBDSN:     dsn,
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
