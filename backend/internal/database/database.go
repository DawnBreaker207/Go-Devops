// Package database initializes the Postgres connection for GORM.
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Connect opens the Postgres connection and tunes the connection pool.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Info
	if cfg.App.IsProduction() {
		logLevel = gormlogger.Warn
	}

	// ParameterizedQueries: SQL is logged with placeholders only, so emails,
	// password hashes and tokens never reach logs (NFR-LEG-01/02).
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

// AutoMigrate creates/updates the schema from models. For dev environments
// only; production uses `make migrate-up` for versioned schema control.
func AutoMigrate(db *gorm.DB) error {
	entities := []any{
		&models.User{},
		&models.Movie{},
		&models.Hall{},
		&models.Seat{},
		&models.HallPrice{},
		&models.Showtime{},
		&models.ShowtimeSeat{},
		&models.AuditLog{},
		&models.BatchJob{},
		&models.DailyAggregate{},
	}
	if err := db.AutoMigrate(entities...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// Close releases the connection, used on shutdown.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Healthy pings the DB, bounded by timeout, for health endpoints.
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
