// Package database khoi tao ket noi Postgres cho GORM.
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Connect mo ket noi toi Postgres va cau hinh connection pool.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Info
	if cfg.App.IsProduction() {
		logLevel = gormlogger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

// AutoMigrate tao/cap nhat schema tu model. Dung cho moi truong dev;
// production nen chay `make migrate-up` de kiem soat phien ban schema.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.User{}, &models.Movie{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// Close dong ket noi, dung khi shutdown.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Healthy kiem tra ket noi con song khong, dung cho endpoint /health.
func Healthy(db *gorm.DB, timeout time.Duration) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- sqlDB.Ping() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("database ping timeout after %s", timeout)
	}
}
