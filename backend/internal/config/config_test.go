package config

import (
	"strings"
	"testing"
	"time"
)

const realSecretA = "k2P9xQ7mV4tR8wZ1nB6cY3hJ5sD0fG2L"
const realSecretB = "u7E4rT1yW9qA3zX6cV8bN2mK5jH0gF4D"

func validConfig() *Config {
	return &Config{
		App:      AppConfig{Env: "development", RoomCleanupMinutes: 20},
		Server:   ServerConfig{Port: 8080, ShutdownTimeout: 10 * time.Second, MaxBodyBytes: 1 << 20},
		Database: DatabaseConfig{Host: "localhost", Name: "cinema"},
		JWT:      JWTConfig{AccessSecret: "dev-access-secret-change-in-prod", RefreshSecret: "dev-refresh-secret-change-in-prod"},
		RateLimit: RateLimitConfig{
			Auth:   RateLimitRule{Capacity: 10, RefillPerSecond: 2},
			Hold:   RateLimitRule{Capacity: 20, RefillPerSecond: 5},
			Public: RateLimitRule{Capacity: 60, RefillPerSecond: 20},
			Events: RateLimitRule{Capacity: 10, RefillPerSecond: 0.2},
		},
		Booking: BookingConfig{HoldTTLMinutes: 10, MaxSeatsPerBooking: 10},
		Checkin: CheckinConfig{OpenBeforeMinutes: 30, CloseAfterMinutes: 20},
		Payment: PaymentConfig{
			LateCaptureWindow: 24 * time.Hour,
			Providers:         PaymentProvidersConfig{Mock: MockProviderConfig{Enabled: true, Secret: "dev-mock-secret-change-in-prod"}},
		},
		Storage: StorageConfig{Driver: "local"},
		Audit:   AuditConfig{RetentionDays: 90},
	}
}

func productionConfig() *Config {
	c := validConfig()
	c.App.Env = "production"
	c.JWT = JWTConfig{AccessSecret: realSecretA, RefreshSecret: realSecretB}
	c.Payment.Providers.Mock = MockProviderConfig{Enabled: false}
	return c
}

func TestValidate_DevelopmentAcceptsDevSecrets(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("dev config: %v", err)
	}
	if err := productionConfig().validate(); err != nil {
		t.Fatalf("production config with real secrets: %v", err)
	}
}

// Production refuses short, placeholder or dev secrets and the mock gateway unless allowed.
func TestValidate_ProductionRejectsDevSecrets(t *testing.T) {
	cases := map[string]func(c *Config){
		"dev jwt secret":     func(c *Config) { c.JWT.AccessSecret = "dev-access-secret-change-in-prod" },
		"placeholder secret": func(c *Config) { c.JWT.RefreshSecret = "change_me_refresh_secret_at_least_32_chars" },
		"short secret":       func(c *Config) { c.JWT.AccessSecret = "short-but-random-9xQ" },
		"mock without allow": func(c *Config) {
			c.Payment.Providers.Mock = MockProviderConfig{Enabled: true, Secret: realSecretA + "x"}
		},
		"mock with dev secret": func(c *Config) {
			c.Payment.Providers.Mock = MockProviderConfig{Enabled: true, AllowInProduction: true, Secret: "dev-mock-secret-change-in-prod"}
		},
		"compose dev secret": func(c *Config) { c.JWT.AccessSecret = "dev_access_secret_change_me_32_chars" },
		"env example secret": func(c *Config) { c.JWT.AccessSecret = "change_me_access_secret_at_least_32_chars" },
		"mock allowed but weak": func(c *Config) {
			c.Payment.Providers.Mock = MockProviderConfig{Enabled: true, AllowInProduction: true, Secret: "abc"}
		},
	}
	for name, mutate := range cases {
		c := productionConfig()
		mutate(c)
		if err := c.validate(); err == nil {
			t.Errorf("%s: accepted in production", name)
		}
	}
	allowed := productionConfig()
	allowed.Payment.Providers.Mock = MockProviderConfig{Enabled: true, AllowInProduction: true, Secret: realSecretA + "mock"}
	if err := allowed.validate(); err != nil {
		t.Fatalf("explicitly allowed mock with a real secret: %v", err)
	}
}

// Trusted proxies must be IPs or CIDRs.
func TestValidate_TrustedProxies(t *testing.T) {
	c := validConfig()
	c.Server.TrustedProxies = []string{"10.0.0.1", "172.16.0.0/12", "::1"}
	if err := c.validate(); err != nil {
		t.Fatalf("valid proxies: %v", err)
	}
	c.Server.TrustedProxies = []string{"proxy.internal"}
	if err := c.validate(); err == nil || !strings.Contains(err.Error(), "trusted_proxies") {
		t.Fatalf("hostname accepted: %v", err)
	}
}

func TestValidate_Bounds(t *testing.T) {
	cases := map[string]func(c *Config){
		"hold ttl 0":          func(c *Config) { c.Booking.HoldTTLMinutes = 0 },
		"max seats 0":         func(c *Config) { c.Booking.MaxSeatsPerBooking = 0 },
		"cleanup negative":    func(c *Config) { c.App.RoomCleanupMinutes = -1 },
		"events refill 0":     func(c *Config) { c.RateLimit.Events.RefillPerSecond = 0 },
		"auth capacity 0":     func(c *Config) { c.RateLimit.Auth.Capacity = 0 },
		"shutdown 0":          func(c *Config) { c.Server.ShutdownTimeout = 0 },
		"tiny body limit":     func(c *Config) { c.Server.MaxBodyBytes = 10 },
		"late window 10m":     func(c *Config) { c.Payment.LateCaptureWindow = 10 * time.Minute },
		"mock without secret": func(c *Config) { c.Payment.Providers.Mock.Secret = "" },
		"checkin opens 0":     func(c *Config) { c.Checkin.OpenBeforeMinutes = 0 },
		"checkin closes 300":  func(c *Config) { c.Checkin.CloseAfterMinutes = 300 },
		"public capacity 0":   func(c *Config) { c.RateLimit.Public.Capacity = 0 },
	}
	for name, mutate := range cases {
		c := validConfig()
		mutate(c)
		if err := c.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Lists split on commas and the removed auto_migrate key is ignored.
func TestLoad_EnvOnlyProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", realSecretA)
	t.Setenv("JWT_REFRESH_SECRET", realSecretB)
	t.Setenv("PAYMENT_PROVIDERS_MOCK_ENABLED", "false")
	t.Setenv("SERVER_TRUSTED_PROXIES", "10.0.0.1,10.0.0.0/8")
	t.Setenv("DATABASE_AUTO_MIGRATE", "true")

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.App.IsProduction() || len(cfg.Server.TrustedProxies) != 2 || cfg.Server.MaxBodyBytes != 1<<20 {
		t.Fatalf("config = env %s proxies %v body %d", cfg.App.Env, cfg.Server.TrustedProxies, cfg.Server.MaxBodyBytes)
	}

	t.Setenv("JWT_ACCESS_SECRET", "dev-access-secret-change-in-prod")
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("production loaded with a dev secret")
	}
}
