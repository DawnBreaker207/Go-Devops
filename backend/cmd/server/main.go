// Package main la diem khoi chay cua BackEnd-CP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/database"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/handlers"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/router"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// @title						BackEnd-CP API
// @version					1.0
// @description				API quan ly rap chieu phim cua Cinema Project.
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Nhap theo dinh dang: Bearer <access_token>
func main() {
	if err := run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load(configPath())
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := logger.Init(cfg.App.Env, cfg.App.LogLevel); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync()

	db, err := database.Connect(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := database.Close(db); closeErr != nil {
			logger.Error("close database failed", logger.Err(closeErr))
		}
	}()

	if cfg.Database.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			return err
		}
		logger.Info("database schema migrated")
	}

	if !cfg.App.IsProduction() {
		if err := database.SeedAdmin(context.Background(), db, cfg.App.AdminEmail, cfg.App.AdminPassword); err != nil {
			return err
		}
	}

	jwtManager := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	// Wire cac layer: repository -> service -> handler.
	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)

	authService := service.NewAuthService(userRepo, jwtManager)
	userService := service.NewUserService(userRepo)
	movieService := service.NewMovieService(movieRepo)

	engine := router.New(cfg, jwtManager, router.Handlers{
		Health: handlers.NewHealthHandler(db, cfg.App.Name),
		Auth:   handlers.NewAuthHandler(authService),
		User:   handlers.NewUserHandler(userService),
		Movie:  handlers.NewMovieHandler(movieService),
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server started",
			logger.String("service", cfg.App.Name),
			logger.String("env", cfg.App.Env),
			logger.Int("port", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Cho tin hieu dung hoac loi tu server.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", logger.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

// configPath cho phep tro toi thu muc chua config.yaml/.env qua bien CONFIG_PATH.
func configPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "."
}
