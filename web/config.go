package main

import (
	"context"
	"os"
)

type Config struct {
	S3UserUploadBucketName string
	S3Region               string
	S3AccessKey            string
	S3SecretKey            string
	S3PresignDuration      int
	DatabaseHost           string
	DatabaseUser           string
	DatabasePassword       string
	DatabaseName           string
	DatabasePort           string
	DatabaseSSLMode        string
	DatabaseTimeZone       string
}

// Add a key type for context
type configKey struct{}

// Attach config to context
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}

// Retrieve config from context
func ConfigFromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configKey{}).(*Config); ok {
		return cfg
	}
	return nil
}

func LoadConfig() *Config {
	return &Config{
		S3UserUploadBucketName: getEnv("S3_BUCKET_NAME", "user-uploads"),
		S3Region:               getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:            getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:            getEnv("S3_SECRET_KEY", "minioadmin"),
		S3PresignDuration:      15 * 60,

		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabaseUser:     getEnv("DB_USER", "postgres"),
		DatabasePassword: getEnv("DB_PASSWORD", "postgres"),
		DatabaseName:     getEnv("DB_NAME", "testdb"),
		DatabasePort:     getEnv("DB_PORT", "5432"),
		DatabaseSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DatabaseTimeZone: getEnv("DB_TIMEZONE", "UTC"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
