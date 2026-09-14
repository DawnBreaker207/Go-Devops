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
	App       AppConfig        `mapstructure:"app"`
	Server    ServerConfig     `mapstructure:"server"`
	Database  DatabaseConfig   `mapstructure:"database"`
	JWT       JWTConfig        `mapstructure:"jwt"`
	CORS      CORSConfig       `mapstructure:"cors"`
	RateLimit RateLimitConfig  `mapstructure:"rate_limit"`
	Queue     QueueConfig      `mapstructure:"queue"`
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
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
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
	Auth RateLimitRule `mapstructure:"auth"`
	Hold RateLimitRule `mapstructure:"hold"`
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
	v.SetDefault("database.auto_migrate", true)

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

	v.SetDefault("queue.url", "amqp://guest:guest@localhost:5672/")
}
