package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	BaseURL        string
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	JWTIssuer      string
	JWTExpiry      time.Duration
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3UseSSL       bool
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	SMTPFrom       string
	OTelEndpoint   string
	OTelEnabled    bool
	AdminEmail     string
	AdminPassword  string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load(".env")

	jwtExpiryHours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "72"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	s3UseSSL, _ := strconv.ParseBool(getEnv("S3_USE_SSL", "false"))
	otelEnabled, _ := strconv.ParseBool(getEnv("OTEL_ENABLED", "false"))

	cfg := &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		Port:          getEnv("PORT", "8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/helpdesk?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:     getEnv("JWT_SECRET", "super-secure-default-jwt-secret-key-change-in-prod"),
		JWTIssuer:     getEnv("JWT_ISSUER", "helpdesk"),
		JWTExpiry:     time.Duration(jwtExpiryHours) * time.Hour,
		S3Endpoint:    getEnv("S3_ENDPOINT", "localhost:9000"),
		S3Region:      getEnv("S3_REGION", "us-east-1"),
		S3Bucket:      getEnv("S3_BUCKET", "helpdesk-attachments"),
		S3AccessKey:   getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:   getEnv("S3_SECRET_KEY", "minioadmin"),
		S3UseSSL:      s3UseSSL,
		SMTPHost:      getEnv("SMTP_HOST", "localhost"),
		SMTPPort:      smtpPort,
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPass:      getEnv("SMTP_PASS", ""),
		SMTPFrom:      getEnv("SMTP_FROM", "support@helpdesk.local"),
		OTelEndpoint:  getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTelEnabled:   otelEnabled,
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@helpdesk.local"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "AdminPass123!"),
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}
