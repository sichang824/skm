package db

import (
	"backend-go/internal/models"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Config struct {
	Driver  string
	DSN     string
	LogMode string
}

func Open(cfg Config) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: logger.Default.LogMode(parseLogLevel(cfg.LogMode)),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	}

	var (
		db  *gorm.DB
		err error
	)

	switch cfg.Driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DSN), gcfg)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DSN), gcfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}

	// Run migrations
	originalLogger := db.Logger
	db.Logger = logger.Default.LogMode(logger.Silent)

	if err := db.AutoMigrate(models.ModelsForAutoMigrate...); err != nil {
		return nil, err
	}

	db.Logger = originalLogger

	return db, nil
}

func parseLogLevel(mode string) logger.LogLevel {
	switch strings.ToLower(mode) {
	case "info":
		return logger.Info
	case "", "warn", "warning":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent", "none":
		return logger.Silent
	default:
		return logger.Warn
	}
}
