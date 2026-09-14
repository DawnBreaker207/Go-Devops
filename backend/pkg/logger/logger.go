// Package logger wraps zap so the system shares one logger.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	global = zap.NewNop()
	// wrapper adds one caller-skip for the helpers below.
	wrapper = zap.NewNop()
)

// Init sets up the logger: JSON in production, colored console otherwise.
func Init(env, level string) error {
	var cfg zap.Config
	if env == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	parsedLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		parsedLevel = zapcore.InfoLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(parsedLevel)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	built, err := cfg.Build()
	if err != nil {
		return err
	}
	global = built
	wrapper = built.WithOptions(zap.AddCallerSkip(1))
	return nil
}

// L returns the current logger.
func L() *zap.Logger { return global }

// Sync flushes buffers before the program exits.
func Sync() { _ = global.Sync() }

func Info(msg string, fields ...zap.Field)  { wrapper.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { wrapper.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { wrapper.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { wrapper.Fatal(msg, fields...) }

// Common field helpers so handlers need not import zap.
func String(key, value string) zap.Field  { return zap.String(key, value) }
func Int(key string, value int) zap.Field { return zap.Int(key, value) }
func Err(err error) zap.Field             { return zap.Error(err) }
