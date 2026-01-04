package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port                string
	StoragePath         string
	AIServiceURL        string
	AIServiceKey        string
	MaxUploadSize       int64
	CleanupDelay        int    // in minutes
	MockAIService       bool   // Enable mock AI service for testing
	CloudinaryCloudName string // Cloudinary cloud name
	CloudinaryAPIKey    string // Cloudinary API key
	CloudinaryAPISecret string // Cloudinary API secret
	EnableEmail         bool   // Enable email notifications
	SMTPHost            string // SMTP server host
	SMTPPort            string // SMTP server port
	SMTPUsername        string // SMTP username
	SMTPPassword        string // SMTP password
	FromEmail           string // From email address
	BaseURL             string // Base URL for result links
	DatabaseURL         string // PostgreSQL connection string
	DatabaseMaxConns    int    // Maximum database connections
	DatabaseMaxIdle     int    // Maximum idle database connections
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		StoragePath:         getEnv("STORAGE_PATH", "./tmp/uploads"),
		AIServiceURL:        getEnv("AI_SERVICE_URL", ""),
		AIServiceKey:        getEnv("AI_SERVICE_KEY", ""),
		MaxUploadSize:       10 * 1024 * 1024, // 10MB
		CleanupDelay:        60,                // 60 minutes
		MockAIService:       getEnv("MOCK_AI_SERVICE", "false") == "true",
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
		EnableEmail:         getEnv("ENABLE_EMAIL", "false") == "true",
		SMTPHost:            getEnv("SMTP_HOST", ""),
		SMTPPort:            getEnv("SMTP_PORT", "587"),
		SMTPUsername:        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:        getEnv("SMTP_PASSWORD", ""),
		FromEmail:           getEnv("FROM_EMAIL", "noreply@virtualfitcheck.com"),
		BaseURL:             getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		DatabaseMaxConns:    getEnvInt("DATABASE_MAX_CONNECTIONS", 25),
		DatabaseMaxIdle:     getEnvInt("DATABASE_MAX_IDLE", 5),
	}
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
