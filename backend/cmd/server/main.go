// Package main is the entry point of BackEnd-CP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/database"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/handlers"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/notify"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/router"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/storage"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/queue"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
)

// @title						BackEnd-CP API
// @version					1.0
// @description				API for the Cinema Project box-office system.
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Header format: Bearer <access_token>
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

	location, err := time.LoadLocation(cfg.Database.TimeZone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", cfg.Database.TimeZone, err)
	}

	// Wire the layers: repository -> service -> handler.
	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)
	hallRepo := repository.NewHallRepository(db)
	showtimeRepo := repository.NewShowtimeRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	loginGuard := ratelimit.NewFailureLimiter(cfg.RateLimit.Login.MaxFailures, cfg.RateLimit.Login.Lockout, nil)
	authService := service.NewAuthService(db, userRepo, repository.NewRefreshTokenRepository(db), jwtManager, loginGuard)
	userService := service.NewUserService(db, userRepo)
	movieService := service.NewMovieService(db, movieRepo)
	hallService := service.NewHallService(db, hallRepo)
	showtimeService := service.NewShowtimeService(db, showtimeRepo, hallRepo, movieRepo, cfg.App.RoomCleanupMinutes, location)

	providers, err := buildPaymentProviders(cfg)
	if err != nil {
		return fmt.Errorf("payment providers: %w", err)
	}

	// The broker must not block startup: without it ticket emails are sent by
	// the sendTicketEmails cron instead of right after confirm.
	var queueClient *queue.Client
	if conn, dialErr := queue.Dial(cfg.Queue.URL); dialErr == nil {
		queueClient = conn
		defer queueClient.Close()
	} else {
		logger.Warn("rabbitmq unavailable; ticket emails fall back to the cron job", logger.Err(dialErr))
	}

	// Realtime seat-map: one SSE hub + short-lived connection tokens.
	hub := sse.NewHub()
	tokens := sse.NewTokenStore(sse.DefaultTokenTTL, nil)

	bookingOpts := service.BookingOptions{
		DB:            db,
		Repo:          bookingRepo,
		Payments:      paymentRepo,
		Providers:     providers,
		PublicBaseURL: cfg.Payment.PublicBaseURL,
		HoldTTL:       time.Duration(cfg.Booking.HoldTTLMinutes) * time.Minute,
		MaxSeats:      cfg.Booking.MaxSeatsPerBooking,
		Hub:           hub,
	}
	if queueClient != nil {
		bookingOpts.Publisher = queueClient
	}
	bookingService := service.NewBookingService(bookingOpts)

	// Ticket emails: mock mailer, each email lands as .html in the outbox dir.
	mailer := notify.NewMockMailer(cfg.Mail.OutboxDir)
	emailService := service.NewTicketEmailService(db, bookingRepo, mailer, location)

	// Daily rollups (closeDay) and the staff board.
	reportService := service.NewReportService(repository.NewReportRepository(db), showtimeRepo, location)

	// Poster uploads: local files (dev) or Cloudinary, chosen by config.
	imageStore, mediaDir := buildImageStore(cfg)
	maxUpload := int64(cfg.Storage.MaxUploadMB) << 20
	mediaService := service.NewMediaService(imageStore, maxUpload)

	// Rate limits: /auth against brute force, /orders/hold against seat bots.
	authLimiter := ratelimit.New(cfg.RateLimit.Auth.Capacity, cfg.RateLimit.Auth.RefillPerSecond)
	holdLimiter := ratelimit.New(cfg.RateLimit.Hold.Capacity, cfg.RateLimit.Hold.RefillPerSecond)

	// Background jobs: registry + cron + batch_jobs log.
	batchRepo := repository.NewBatchJobRepository(db)
	batchManager := batch.NewManager(db, batchRepo)
	batchManager.Register(jobs.NewSweepExpiredHolds(bookingService))
	batchManager.Register(jobs.NewSendTicketEmails(emailService))
	batchManager.Register(jobs.NewCloseDay(reportService, location))
	batchManager.Register(jobs.NewCleanup(repository.NewMaintenanceRepository(db), cfg.Audit.RetentionDays, location))
	batchManager.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		batchManager.Stop(ctx)
	}()

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	if queueClient != nil {
		go jobs.ConsumeTicketEmails(workerCtx, queueClient, emailService)
	}

	engine := router.New(cfg, db, jwtManager, authLimiter, holdLimiter, providers, router.Handlers{
		Health:   handlers.NewHealthHandler(db, cfg.App.Name),
		Auth:     handlers.NewAuthHandler(authService),
		User:     handlers.NewUserHandler(userService),
		Movie:    handlers.NewMovieHandler(movieService),
		Batch:    handlers.NewBatchHandler(batchManager, batchRepo, db),
		Hall:     handlers.NewHallHandler(hallService),
		Showtime: handlers.NewShowtimeHandler(showtimeService),
		Booking:  handlers.NewBookingHandler(bookingService),
		SSE:      handlers.NewSSEHandler(hub, tokens, showtimeService),
		Payment:  handlers.NewPaymentHandler(providers, bookingService, cfg.Payment.ReturnRedirectURL),
		Staff:    handlers.NewStaffHandler(reportService),
		Report:   handlers.NewReportHandler(reportService),
		Media:    handlers.NewMediaHandler(mediaService, mediaDir, maxUpload),
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	// SSE streams never go idle; end them as soon as shutdown starts so
	// Shutdown does not wait out its timeout.
	server.RegisterOnShutdown(hub.Close)

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

	// Wait for a shutdown signal or a fatal server error.
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

// buildPaymentProviders registers every enabled payment provider. A real
// gateway is added here exactly like the mock: build its adapter from its
// config block and register it.
func buildPaymentProviders(cfg *config.Config) (*payment.Registry, error) {
	registry := payment.NewRegistry()
	if m := cfg.Payment.Providers.Mock; m.Enabled {
		if cfg.App.IsProduction() {
			logger.Warn("mock payment provider enabled in production: it collects no real money")
		}
		if err := registry.Register(mock.New(mock.Options{
			Name:          "mock",
			DisplayName:   m.DisplayName,
			Secret:        m.Secret,
			PublicBaseURL: cfg.Payment.PublicBaseURL,
		})); err != nil {
			return nil, err
		}
	}
	if err := registry.SetDefault(cfg.Payment.DefaultProvider); err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, p := range registry.List() {
		names = append(names, p.Name())
	}
	if len(names) == 0 {
		logger.Warn("no payment provider enabled: customers can not pay")
	} else {
		logger.Info("payment providers enabled",
			logger.String("providers", strings.Join(names, ",")), logger.String("default", registry.Default()))
	}
	return registry, nil
}

// buildImageStore picks the poster store; the returned dir is served under
// /media for the local store and empty for Cloudinary.
func buildImageStore(cfg *config.Config) (storage.Store, string) {
	if cfg.Storage.Driver == "cloudinary" {
		c := cfg.Storage.Cloudinary
		logger.Info("image storage: cloudinary", logger.String("cloud", c.CloudName))
		return storage.NewCloudinary(storage.CloudinaryOptions{
			CloudName: c.CloudName, APIKey: c.APIKey, APISecret: c.APISecret, Folder: c.Folder,
		}), ""
	}
	logger.Info("image storage: local", logger.String("dir", cfg.Storage.LocalDir))
	return storage.NewLocal(cfg.Storage.LocalDir, cfg.Storage.PublicBaseURL), cfg.Storage.LocalDir
}

// configPath resolves the directory holding config.yaml/.env via CONFIG_PATH.
func configPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "."
}
