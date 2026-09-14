// Package config loads settings from config.yaml, .env and environment vars.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds every service setting.
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Booking   BookingConfig   `mapstructure:"booking"`
	Payment   PaymentConfig   `mapstructure:"payment"`
	Mail      MailConfig      `mapstructure:"mail"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Audit     AuditConfig     `mapstructure:"audit"`
}

// AuditConfig sets how long audit logs are kept (the cleanup job removes older
// rows). Must be at least 90 days (FR-AUDIT-02).
type AuditConfig struct {
	RetentionDays int `mapstructure:"retention_days"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Env      string `mapstructure:"env"`
	LogLevel string `mapstructure:"log_level"`
	// Default admin account seeded when the DB is empty (non-production only).
	AdminEmail    string `mapstructure:"admin_email"`
	AdminPassword string `mapstructure:"admin_password"`
	// RoomCleanupMinutes is the buffer appended to a showtime's effective end
	// when checking hall-schedule conflicts.
	RoomCleanupMinutes int `mapstructure:"room_cleanup_minutes"`
}

type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	TimeZone        string        `mapstructure:"timezone"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessTTL     time.Duration `mapstructure:"access_ttl"`
	RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
	Issuer        string        `mapstructure:"issuer"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// RateLimitConfig throttles each endpoint group per client.
type RateLimitConfig struct {
	Auth  RateLimitRule    `mapstructure:"auth"`
	Hold  RateLimitRule    `mapstructure:"hold"`
	Login LoginGuardConfig `mapstructure:"login"`
}

// LoginGuardConfig locks an email+IP pair out after consecutive wrong
// passwords (T14).
type LoginGuardConfig struct {
	MaxFailures int           `mapstructure:"max_failures"`
	Lockout     time.Duration `mapstructure:"lockout"`
}

// RateLimitRule is the parameter set of one token bucket.
type RateLimitRule struct {
	Capacity        int     `mapstructure:"capacity"`
	RefillPerSecond float64 `mapstructure:"refill_per_second"`
}

// QueueConfig configures the RabbitMQ connection.
type QueueConfig struct {
	URL string `mapstructure:"url"`
}

// BookingConfig tunes the seat-hold flow.
type BookingConfig struct {
	// HoldTTLMinutes is how long a held seat stays reserved.
	HoldTTLMinutes int `mapstructure:"hold_ttl_minutes"`
	// MaxSeatsPerBooking caps how many seats one customer can hold at once.
	MaxSeatsPerBooking int `mapstructure:"max_seats_per_booking"`
}

// PaymentConfig configures payments. Every provider — the mock as much as a
// real gateway — has its own block under Providers and is enabled on its own.
type PaymentConfig struct {
	// PublicBaseURL is how providers and browsers reach this API (IPN and
	// return URLs, mock checkout page).
	PublicBaseURL string `mapstructure:"public_base_url"`
	// ReturnRedirectURL is where the browser lands after the return URL is
	// processed; empty answers with JSON.
	ReturnRedirectURL string `mapstructure:"return_redirect_url"`
	// DefaultProvider is used when a pay request names none; it must be enabled.
	DefaultProvider string                 `mapstructure:"default_provider"`
	Providers       PaymentProvidersConfig `mapstructure:"providers"`
}

// PaymentProvidersConfig has one block per supported provider. A real gateway
// adds its block here (e.g. VNPay: enabled, tmn_code, hash_secret, endpoint).
type PaymentProvidersConfig struct {
	Mock MockProviderConfig `mapstructure:"mock"`
}

// MockProviderConfig enables the simulated gateway.
type MockProviderConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	DisplayName string `mapstructure:"display_name"`
	// Secret signs the mock IPN and return redirect (HMAC-SHA256).
	Secret string `mapstructure:"secret"`
}

// MailConfig configures ticket emails. This build only has a mock mailer:
// every email is written as an .html file into OutboxDir.
type MailConfig struct {
	OutboxDir string `mapstructure:"outbox_dir"`
}

// StorageConfig selects where uploaded images go: "local" (files served under
// /media) or "cloudinary".
type StorageConfig struct {
	Driver        string           `mapstructure:"driver"`
	PublicBaseURL string           `mapstructure:"public_base_url"`
	LocalDir      string           `mapstructure:"local_dir"`
	MaxUploadMB   int              `mapstructure:"max_upload_mb"`
	Cloudinary    CloudinaryConfig `mapstructure:"cloudinary"`
}

// CloudinaryConfig holds the Cloudinary credentials (driver "cloudinary").
type CloudinaryConfig struct {
	CloudName string `mapstructure:"cloud_name"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	Folder    string `mapstructure:"folder"`
}

// DSN returns the Postgres connection string for GORM.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone,
	)
}

// MigrateURL returns the connection URL used by golang-migrate.
func (d DatabaseConfig) MigrateURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// IsProduction reports whether we run in the production environment.
func (a AppConfig) IsProduction() bool { return a.Env == "production" }

// Load reads config in order: defaults -> config.yaml -> .env -> env vars.
func Load(path string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(path)
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config.yaml: %w", err)
		}
	}

	// .env only fills gaps: real env vars always win.
	if err := loadDotEnv(path + "/.env"); err != nil {
		return nil, err
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// loadDotEnv loads .env into the process env, never overriding existing vars.
func loadDotEnv(file string) error {
	envViper := viper.New()
	envViper.SetConfigFile(file)
	envViper.SetConfigType("env")

	if err := envViper.ReadInConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil
		}
		return fmt.Errorf("read .env: %w", err)
	}

	for _, key := range envViper.AllKeys() {
		envKey := strings.ToUpper(key)
		if _, exists := os.LookupEnv(envKey); exists {
			continue
		}
		if err := os.Setenv(envKey, fmt.Sprint(envViper.Get(key))); err != nil {
			return fmt.Errorf("set env %s: %w", envKey, err)
		}
	}
	return nil
}

func (c *Config) validate() error {
	if c.JWT.AccessSecret == "" || c.JWT.RefreshSecret == "" {
		return errors.New("jwt access_secret and refresh_secret must not be empty")
	}
	if c.JWT.AccessSecret == c.JWT.RefreshSecret {
		return errors.New("jwt access_secret and refresh_secret must be different")
	}
	if c.Database.Host == "" || c.Database.Name == "" {
		return errors.New("database host and name must not be empty")
	}
	if c.Server.Port <= 0 {
		return errors.New("server port must be greater than 0")
	}
	if c.Audit.RetentionDays < 90 {
		return fmt.Errorf("audit.retention_days must be at least 90, got %d", c.Audit.RetentionDays)
	}
	switch c.Storage.Driver {
	case "local":
	case "cloudinary":
		cl := c.Storage.Cloudinary
		if cl.CloudName == "" || cl.APIKey == "" || cl.APISecret == "" {
			return errors.New("storage.cloudinary cloud_name, api_key and api_secret are required for the cloudinary driver")
		}
	default:
		return fmt.Errorf("storage.driver must be local or cloudinary, got %q", c.Storage.Driver)
	}
	if c.Payment.Providers.Mock.Enabled && c.Payment.Providers.Mock.Secret == "" {
		return errors.New("payment.providers.mock.secret must not be empty when the mock provider is enabled")
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "BackEnd-CP")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.admin_email", "admin@cinema.local")
	v.SetDefault("app.admin_password", "admin123")
	v.SetDefault("app.room_cleanup_minutes", 20)

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "10s")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.name", "cinema")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.timezone", "Asia/Ho_Chi_Minh")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")

	v.SetDefault("jwt.access_secret", "")
	v.SetDefault("jwt.refresh_secret", "")
	v.SetDefault("jwt.access_ttl", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")
	v.SetDefault("jwt.issuer", "backend-cp")

	v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000"})

	v.SetDefault("rate_limit.auth.capacity", 10)
	v.SetDefault("rate_limit.auth.refill_per_second", 2)
	v.SetDefault("rate_limit.hold.capacity", 20)
	v.SetDefault("rate_limit.hold.refill_per_second", 5)
	v.SetDefault("rate_limit.login.max_failures", 5)
	v.SetDefault("rate_limit.login.lockout", "5m")

	v.SetDefault("audit.retention_days", 90)

	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.public_base_url", "http://localhost:8080")
	v.SetDefault("storage.local_dir", "tmp/uploads")
	v.SetDefault("storage.max_upload_mb", 5)
	v.SetDefault("storage.cloudinary.cloud_name", "")
	v.SetDefault("storage.cloudinary.api_key", "")
	v.SetDefault("storage.cloudinary.api_secret", "")
	v.SetDefault("storage.cloudinary.folder", "cinema")

	v.SetDefault("booking.hold_ttl_minutes", 10)
	v.SetDefault("booking.max_seats_per_booking", 10)

	v.SetDefault("payment.public_base_url", "http://localhost:8080")
	v.SetDefault("payment.return_redirect_url", "")
	v.SetDefault("payment.default_provider", "mock")
	v.SetDefault("payment.providers.mock.enabled", true)
	v.SetDefault("payment.providers.mock.display_name", "Cổng thử nghiệm (mock)")
	v.SetDefault("payment.providers.mock.secret", "dev-mock-secret-change-in-prod")

	v.SetDefault("mail.outbox_dir", "tmp/mail")

	v.SetDefault("queue.url", "amqp://guest:guest@localhost:5672/")
}
