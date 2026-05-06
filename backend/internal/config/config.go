package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv          string
	Port            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	JWTRefreshSecret string
	JWTExpiry       string
	RefreshExpiry   string
	StripeSecretKey string
	StripeWebhookSecret string
	AllowedOrigins  string
	FirebaseCredentials string
	UploadPath      string
	MaxUploadSize   int64
}

func Load() *Config {
	return &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://deliveroo_user:SecurePass2026!@localhost:5432/deliveroo?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://:RedisPass2026!@localhost:6379/0"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production"),
		JWTExpiry:       getEnv("JWT_EXPIRY", "15m"),
		RefreshExpiry:   getEnv("REFRESH_EXPIRY", "7d"),
		StripeSecretKey: getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		AllowedOrigins:  getEnv("ALLOWED_ORIGINS", "http://localhost,http://localhost:3000,http://localhost:8080"),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", ""),
		UploadPath:      getEnv("UPLOAD_PATH", "./uploads"),
		MaxUploadSize:   10 * 1024 * 1024, // 10MB
	}
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.AppEnv) == "production"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
