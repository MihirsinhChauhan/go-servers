package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func init() {
	// Default to development mode
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// In production, switch to JSON
	if os.Getenv("ENV") == "production" || os.Getenv("PLATFORM") == "prod" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	rawLogger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	Logger = rawLogger.Sugar()
}

// Helper for cleanup
func Sync() {
	_ = Logger.Sync()
}