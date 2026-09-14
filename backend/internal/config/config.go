// Package config loads settings from config.yaml, .env and environment vars.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Booking   BookingConfig   `mapstructure:"booking"`
	Checkin   CheckinConfig   `mapstructure:"checkin"`
	Payment   PaymentConfig   `mapstructure:"payment"`
	Mail      MailConfig      `mapstructure:"mail"`
	Account   AccountConfig   `mapstructure:"account"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Audit     AuditConfig     `mapstructure:"audit"`
}

// AccountConfig wires the self-service account flows.
type AccountConfig struct {
	// PasswordResetURL is the frontend page the reset link points to; the token is appended
	// as ?token=<hex>. Reset tokens stay valid PasswordResetTTL (5m-24h).
	PasswordResetURL string        `mapstructure:"password_reset_url"`
	PasswordResetTTL time.Duration `mapstructure:"password_reset_ttl"`
}

type AuditConfig struct {
	// RetentionDays: the cleanup job deletes older audit logs; must be at least 90.
	RetentionDays int `mapstructure:"retention_days"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Env      string `mapstructure:"env"`
	LogLevel string `mapstructure:"log_level"`
	// Admin account seeded when the DB is empty (non-production only).
	AdminEmail    string `mapstructure:"admin_email"`
	AdminPassword string `mapstructure:"admin_password"`
	// RoomCleanupMinutes is added to a showtime's end when checking hall schedule conflicts.
	RoomCleanupMinutes int `mapstructure:"room_cleanup_minutes"`
}

type ServerConfig struct {
	Port              int           `mapstructure:"port"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	// MaxBodyBytes caps JSON request bodies; uploads enforce their own limit.
	MaxBodyBytes int64 `mapstructure:"max_body_bytes"`
	// TrustedProxies (IPs or CIDRs) may set X-Forwarded-For. Empty trusts none, so the
	// client IP is the TCP peer and rate limits can't be dodged.
	TrustedProxies []string `mapstructure:"trusted_proxies"`
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

type RateLimitConfig struct {
	Auth RateLimitRule `mapstructure:"auth"`
	Hold RateLimitRule `mapstructure:"hold"`
	// Public throttles anonymous catalog reads per IP.
	Public RateLimitRule `mapstructure:"public"`
	// Events throttles realtime tokens per user.
	Events RateLimitRule    `mapstructure:"events"`
	Login  LoginGuardConfig `mapstructure:"login"`
}

// LoginGuardConfig locks an email+IP pair out for Lockout after MaxFailures consecutive wrong passwords.
type LoginGuardConfig struct {
	MaxFailures int           `mapstructure:"max_failures"`
	Lockout     time.Duration `mapstructure:"lockout"`
}

// RateLimitRule configures one token bucket.
type RateLimitRule struct {
	Capacity        int     `mapstructure:"capacity"`
	RefillPerSecond float64 `mapstructure:"refill_per_second"`
}

type QueueConfig struct {
	URL string `mapstructure:"url"`
}

type BookingConfig struct {
	HoldTTLMinutes     int `mapstructure:"hold_ttl_minutes"`
	MaxSeatsPerBooking int `mapstructure:"max_seats_per_booking"`
}

// CheckinConfig: tickets are accepted from OpenBeforeMinutes before the showtime to CloseAfterMinutes after its start.
type CheckinConfig struct {
	OpenBeforeMinutes int `mapstructure:"open_before_minutes"`
	CloseAfterMinutes int `mapstructure:"close_after_minutes"`
}

type PaymentConfig struct {
	// PublicBaseURL is how providers and browsers reach this API (IPN, return URLs, mock checkout).
	PublicBaseURL string `mapstructure:"public_base_url"`
	// ReturnRedirectURL is where the browser lands after the return URL; empty answers with JSON.
	ReturnRedirectURL string `mapstructure:"return_redirect_url"`
	// DefaultProvider is used when a pay request names none; it must be enabled.
	DefaultProvider string                 `mapstructure:"default_provider"`
	Providers       PaymentProvidersConfig `mapstructure:"providers"`
	// LateCaptureWindow is how long given-up attempts are rechecked for money collected late.
	LateCaptureWindow time.Duration `mapstructure:"late_capture_window"`
}

type PaymentProvidersConfig struct {
	Mock MockProviderConfig `mapstructure:"mock"`
}

type MockProviderConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	DisplayName string `mapstructure:"display_name"`
	Secret      string `mapstructure:"secret"`
	// AllowInProduction is required to run the mock in production, where it collects no real money.
	AllowInProduction bool `mapstructure:"allow_in_production"`
}

// MailConfig: the mock mailer writes every email as an .html file into OutboxDir.
type MailConfig struct {
	OutboxDir string `mapstructure:"outbox_dir"`
}

// StorageConfig.Driver is "local" (files served under /media) or "cloudinary".
type StorageConfig struct {
	Driver        string           `mapstructure:"driver"`
	PublicBaseURL string           `mapstructure:"public_base_url"`
	LocalDir      string           `mapstructure:"local_dir"`
	MaxUploadMB   int              `mapstructure:"max_upload_mb"`
	Cloudinary    CloudinaryConfig `mapstructure:"cloudinary"`
}

type CloudinaryConfig struct {
	CloudName string `mapstructure:"cloud_name"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	Folder    string `mapstructure:"folder"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone,
	)
}

// MigrateURL is the connection URL for golang-migrate.
func (d DatabaseConfig) MigrateURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

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

// Fragments of the placeholder secrets shipped in config.yaml, .env.example and docker-compose.yml.
var devSecretMarkers = []string{"change_me", "change-me", "changeme", "change-in-prod", "change_in_prod"}

// productionSecret rejects short or placeholder secrets: with a public dev secret anyone
// could sign admin tokens or fake payment notifications.
func productionSecret(name, value string) error {
	if len(value) < 32 {
		return fmt.Errorf("%s must be at least 32 characters in production", name)
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "dev-") || strings.HasPrefix(lower, "dev_") {
		return fmt.Errorf("%s is a development secret; set a real one in production", name)
	}
	for _, marker := range devSecretMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%s is a placeholder secret; set a real one in production", name)
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
	if c.App.IsProduction() {
		if err := productionSecret("jwt.access_secret", c.JWT.AccessSecret); err != nil {
			return err
		}
		if err := productionSecret("jwt.refresh_secret", c.JWT.RefreshSecret); err != nil {
			return err
		}
		if mock := c.Payment.Providers.Mock; mock.Enabled {
			if !mock.AllowInProduction {
				return errors.New("payment.providers.mock is enabled in production but collects no real money: " +
					"disable it or set payment.providers.mock.allow_in_production")
			}
			if err := productionSecret("payment.providers.mock.secret", mock.Secret); err != nil {
				return err
			}
		}
	}
	for _, proxy := range c.Server.TrustedProxies {
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return fmt.Errorf("server.trusted_proxies: %q is not an IP or CIDR", proxy)
			}
		}
	}
	if c.Server.ShutdownTimeout < time.Second {
		return fmt.Errorf("server.shutdown_timeout must be at least 1s, got %s", c.Server.ShutdownTimeout)
	}
	if c.Server.MaxBodyBytes < 1024 {
		return fmt.Errorf("server.max_body_bytes must be at least 1024, got %d", c.Server.MaxBodyBytes)
	}
	if c.Booking.HoldTTLMinutes < 1 || c.Booking.HoldTTLMinutes > 60 {
		return fmt.Errorf("booking.hold_ttl_minutes must be between 1 and 60, got %d", c.Booking.HoldTTLMinutes)
	}
	if c.Booking.MaxSeatsPerBooking < 1 || c.Booking.MaxSeatsPerBooking > 50 {
		return fmt.Errorf("booking.max_seats_per_booking must be between 1 and 50, got %d", c.Booking.MaxSeatsPerBooking)
	}
	if c.App.RoomCleanupMinutes < 0 || c.App.RoomCleanupMinutes > 240 {
		return fmt.Errorf("app.room_cleanup_minutes must be between 0 and 240, got %d", c.App.RoomCleanupMinutes)
	}
	if c.Checkin.OpenBeforeMinutes < 1 || c.Checkin.OpenBeforeMinutes > 240 {
		return fmt.Errorf("checkin.open_before_minutes must be between 1 and 240, got %d", c.Checkin.OpenBeforeMinutes)
	}
	if c.Checkin.CloseAfterMinutes < 0 || c.Checkin.CloseAfterMinutes > 240 {
		return fmt.Errorf("checkin.close_after_minutes must be between 0 and 240, got %d", c.Checkin.CloseAfterMinutes)
	}
	for name, rule := range map[string]RateLimitRule{
		"auth": c.RateLimit.Auth, "hold": c.RateLimit.Hold, "public": c.RateLimit.Public, "events": c.RateLimit.Events,
	} {
		if rule.Capacity < 1 || rule.RefillPerSecond <= 0 {
			return fmt.Errorf("rate_limit.%s needs capacity >= 1 and refill_per_second > 0", name)
		}
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
	if c.Payment.LateCaptureWindow < time.Hour {
		return fmt.Errorf("payment.late_capture_window must be at least 1h, got %s", c.Payment.LateCaptureWindow)
	}
	if c.Payment.Providers.Mock.Enabled && c.Payment.Providers.Mock.Secret == "" {
		return errors.New("payment.providers.mock.secret must not be empty when the mock provider is enabled")
	}
	if c.Account.PasswordResetURL == "" {
		c.Account.PasswordResetURL = "http://localhost:5173/reset-password"
	}
	if c.Account.PasswordResetTTL == 0 {
		c.Account.PasswordResetTTL = 30 * time.Minute
	}
	if c.Account.PasswordResetTTL < 5*time.Minute || c.Account.PasswordResetTTL > 24*time.Hour {
		return fmt.Errorf("account.password_reset_ttl must be between 5m and 24h, got %s", c.Account.PasswordResetTTL)
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
	v.SetDefault("server.read_header_timeout", "5s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.max_body_bytes", 1<<20)
	v.SetDefault("server.trusted_proxies", []string{})

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
	v.SetDefault("rate_limit.public.capacity", 60)
	v.SetDefault("rate_limit.public.refill_per_second", 20)
	v.SetDefault("rate_limit.events.capacity", 10)
	v.SetDefault("rate_limit.events.refill_per_second", 0.2)
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

	v.SetDefault("checkin.open_before_minutes", 30)
	v.SetDefault("checkin.close_after_minutes", 20)

	v.SetDefault("payment.public_base_url", "http://localhost:8080")
	v.SetDefault("payment.return_redirect_url", "")
	v.SetDefault("payment.default_provider", "mock")
	v.SetDefault("payment.late_capture_window", "24h")
	v.SetDefault("payment.providers.mock.enabled", true)
	v.SetDefault("payment.providers.mock.display_name", "Cổng thử nghiệm (mock)")
	// No default mock secret: config.yaml / env must provide one.
	v.SetDefault("payment.providers.mock.secret", "")
	v.SetDefault("payment.providers.mock.allow_in_production", false)

	v.SetDefault("mail.outbox_dir", "tmp/mail")

	v.SetDefault("account.password_reset_url", "http://localhost:5173/reset-password")
	v.SetDefault("account.password_reset_ttl", "30m")

	v.SetDefault("queue.url", "amqp://guest:guest@localhost:5672/")
}
