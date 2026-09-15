// Package database opens the Postgres connection for GORM.
package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Info
	if cfg.App.IsProduction() {
		logLevel = gormlogger.Warn
	}

	// ParameterizedQueries logs placeholders only, so emails, password hashes and tokens never reach logs.
	sqlLogger := gormlogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logLevel,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  !cfg.App.IsProduction(),
	})

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger:                 sqlLogger,
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

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func Healthy(db *gorm.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Ping(ctx, db)
}

func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
