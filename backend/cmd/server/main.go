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
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
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

	// The well-known seed admin must never exist outside development.
	if cfg.App.Env == "development" {
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

	userRepo := repository.NewUserRepository(db)
	movieRepo := repository.NewMovieRepository(db)
	hallRepo := repository.NewHallRepository(db)
	showtimeRepo := repository.NewShowtimeRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	notifPrefRepo := repository.NewNotificationPreferenceRepository(db)

	loginGuard := ratelimit.NewFailureLimiter(cfg.RateLimit.Login.MaxFailures, cfg.RateLimit.Login.Lockout, nil)
	mailer := notify.NewMockMailer(cfg.Mail.OutboxDir)
	authService := service.NewAuthService(db, userRepo, repository.NewRefreshTokenRepository(db), jwtManager, loginGuard,
		repository.NewPasswordResetTokenRepository(db), mailer, cfg.Account.PasswordResetURL, cfg.Account.PasswordResetTTL,
		cfg.Account.TermsVersion, cfg.JWT.RefreshTTL)
	// Locks and role changes reach already-issued tokens within accountStatusTTL.
	accountStatus := service.NewAccountStatusCache(userRepo, accountStatusTTL)
	userService := service.NewUserService(db, userRepo, notifPrefRepo, accountStatus.Invalidate)
	var catalogCache *cache.Cache
	if cfg.Redis.Addr != "" {
		catalogCache = cache.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		defer catalogCache.Close()
	}
	movieService := service.NewMovieService(db, movieRepo, catalogCache, cfg.Redis.TTL)
	hallService := service.NewHallService(db, hallRepo, catalogCache)
	showtimeService := service.NewShowtimeService(db, showtimeRepo, hallRepo, movieRepo, cfg.App.RoomCleanupMinutes, location, catalogCache, cfg.Redis.ShowtimesTTL)

	providers, err := buildPaymentProviders(cfg)
	if err != nil {
		return fmt.Errorf("payment providers: %w", err)
	}

	// The broker must not block startup: without it the sendTicketEmails cron sends the emails.
	var queueClient *queue.Client
	if conn, dialErr := queue.Dial(cfg.Queue.URL); dialErr == nil {
		queueClient = conn
		defer queueClient.Close()
	} else {
		logger.Warn("rabbitmq unavailable; ticket emails fall back to the cron job", logger.Err(dialErr))
	}

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

		LateCaptureWindow: cfg.Payment.LateCaptureWindow,
		CheckinOpenBefore: time.Duration(cfg.Checkin.OpenBeforeMinutes) * time.Minute,
		CheckinCloseAfter: time.Duration(cfg.Checkin.CloseAfterMinutes) * time.Minute,
		Location:          location,
	}
	if queueClient != nil {
		bookingOpts.Publisher = queueClient
	}
	bookingService := service.NewBookingService(bookingOpts)

	emailService := service.NewTicketEmailService(db, bookingRepo, mailer, location)

	batchRepo := repository.NewBatchJobRepository(db)
	reportService := service.NewReportService(repository.NewReportRepository(db), showtimeRepo,
		paymentRepo, batchRepo, bookingRepo, location)

	comboService := service.NewComboService(repository.NewComboRepository(db), repository.NewComboOrderRepository(db), bookingRepo)

	imageStore, mediaDir := buildImageStore(cfg)
	maxUpload := int64(cfg.Storage.MaxUploadMB) << 20
	mediaService := service.NewMediaService(imageStore, maxUpload)

	limits := router.Limiters{
		Auth:   ratelimit.New(cfg.RateLimit.Auth.Capacity, cfg.RateLimit.Auth.RefillPerSecond),
		Hold:   ratelimit.New(cfg.RateLimit.Hold.Capacity, cfg.RateLimit.Hold.RefillPerSecond),
		Public: ratelimit.New(cfg.RateLimit.Public.Capacity, cfg.RateLimit.Public.RefillPerSecond),
		Events: ratelimit.New(cfg.RateLimit.Events.Capacity, cfg.RateLimit.Events.RefillPerSecond),
	}

	batchManager := batch.NewManager(db, batchRepo)
	batchManager.Register(jobs.NewSweepExpiredHolds(bookingService))
	batchManager.Register(jobs.NewSendTicketEmails(emailService))
	batchManager.Register(jobs.NewCloseDay(reportService, location))
	batchManager.Register(jobs.NewCleanup(repository.NewMaintenanceRepository(db), cfg.Audit.RetentionDays, location))
	if err := batchManager.Start(context.Background()); err != nil {
		return fmt.Errorf("start batch jobs: %w", err)
	}

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	if queueClient != nil {
		go jobs.ConsumeTicketEmails(workerCtx, queueClient, emailService)
	}

	engine := router.New(cfg, db, jwtManager, accountStatus, limits, providers, router.Handlers{
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
		Staff:    handlers.NewStaffHandler(reportService, bookingService, userService),
		Report:   handlers.NewReportHandler(reportService),
		Media:    handlers.NewMediaHandler(mediaService, mediaDir, maxUpload),
		Audit:    handlers.NewAuditHandler(service.NewAuditService(repository.NewAuditRepository(db))),
		Combo:    handlers.NewComboHandler(comboService),
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    1 << 16,
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var runErr error
	select {
	case err := <-serverErr:
		runErr = fmt.Errorf("listen: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", logger.String("signal", sig.String()))
	}

	// One deadline covers the whole shutdown, all before the deferred database close.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if runErr == nil {
		if err := server.Shutdown(ctx); err != nil {
			logger.Warn("graceful shutdown ran out of time; closing connections", logger.Err(err))
			_ = server.Close()
		}
	}
	batchManager.Stop(ctx)
	stopWorkers()

	logger.Info("server stopped")
	return runErr
}

const accountStatusTTL = 30 * time.Second

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

func configPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return "."
}
