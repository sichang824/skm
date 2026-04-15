package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	LogLevel  string
	LogFormat string
	DBDriver  string
	DBDSN     string
	Seed      bool
	SeedOnly  bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "console")
	dbDriver := getEnv("DB_DRIVER", "sqlite")
	dsn := getEnv("DB_DSN", "./data/app.db")
	seed := getEnvBool("SEED", false)
	seedOnly := getEnvBool("SEED_ONLY", false)

	return &Config{
		Port:      port,
		LogLevel:  logLevel,
		LogFormat: logFormat,
		DBDriver:  dbDriver,
		DBDSN:     dsn,
		Seed:      seed,
		SeedOnly:  seedOnly,
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
