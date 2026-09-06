// Package logger boc zap de toan he thong dung chung mot logger.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// global duoc dung khi goi L() truc tiep.
	global = zap.NewNop()
	// wrapper co them mot cap caller skip cho cac ham tien ich ben duoi.
	wrapper = zap.NewNop()
)

// Init khoi tao logger: moi truong production dung JSON, con lai dung console mau.
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

// L tra ve logger dang dung.
func L() *zap.Logger { return global }

// Sync day het buffer truoc khi thoat chuong trinh.
func Sync() { _ = global.Sync() }

func Info(msg string, fields ...zap.Field)  { wrapper.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { wrapper.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { wrapper.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { wrapper.Fatal(msg, fields...) }

// Cac helper field hay dung, de handler khong phai import zap.
func String(key, value string) zap.Field  { return zap.String(key, value) }
func Int(key string, value int) zap.Field { return zap.Int(key, value) }
func Err(err error) zap.Field             { return zap.Error(err) }
